package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

const expectedExportApp = "p40la-ihost-automation"
const expectedExportFormat = 1

// jsonSourceTable describe una tabla del export y las columnas que el
// importador necesita. Las columnas desconocidas del JSON se ignoran.
type jsonSourceTable struct {
	name    string
	columns []string
}

// Orden respetando foreign keys: apps/plugins antes de accounts, y accounts
// antes de bills/webhook_logs.
var jsonSourceTables = []jsonSourceTable{
	{name: "settings", columns: []string{"key", "value"}},
	{name: "apps", columns: []string{"id", "name", "created_at", "updated_at"}},
	{name: "plugins", columns: []string{
		"id", "name", "version", "source", "description", "status", "created_at", "updated_at",
	}},
	{name: "accounts", columns: []string{
		"id", "app_id", "identifier", "label", "credentials", "plugin_name",
		"webhook_url", "schedule_time", "schedule_enabled", "last_run_at",
		"schedule_frequency", "schedule_interval", "schedule_days", "created_at", "updated_at",
	}},
	{name: "bills", columns: []string{
		"id", "account_id", "plugin_name", "plugin_version", "source", "status",
		"error", "raw", "fetched_at", "created_at",
	}},
	{name: "webhook_logs", columns: []string{
		"id", "account_id", "webhook_url", "status", "payload", "response", "ran_at", "http_status",
	}},
}

// automationExport es el sobre del export de p40la-ihost-automation.
type automationExport struct {
	Export struct {
		FormatVersion int                         `json:"format_version"`
		App           string                      `json:"app"`
		Tables        map[string][]map[string]any `json:"tables"`
	} `json:"export"`
}

// materializeJSONSource convierte el export JSON en una SQLite temporal con el
// esquema viejo de automation. Devuelve la ruta y un cleanup que cierra y
// elimina el directorio temporal (siempre debe ejecutarse).
func materializeJSONSource(path string) (string, func(), error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", func() {}, fmt.Errorf("leer export JSON %s: %w", path, err)
	}

	var export automationExport
	if err := json.Unmarshal(raw, &export); err != nil {
		return "", func() {}, fmt.Errorf("parsear export JSON: %w", err)
	}
	if err := validateJSONExport(export); err != nil {
		return "", func() {}, err
	}

	dir, err := os.MkdirTemp("", "p40la-automation-json-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("crear directorio temporal: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		os.RemoveAll(dir)
		return "", func() {}, fmt.Errorf("proteger directorio temporal: %w", err)
	}

	dbPath := filepath.Join(dir, "automation.db")
	database, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		os.RemoveAll(dir)
		return "", func() {}, fmt.Errorf("abrir SQLite temporal: %w", err)
	}

	cleanup := func() {
		database.Close()
		os.RemoveAll(dir)
	}

	if err := createLegacySchema(database); err != nil {
		cleanup()
		return "", func() {}, err
	}
	if err := insertJSONTables(database, export); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return dbPath, cleanup, nil
}

func validateJSONExport(export automationExport) error {
	if export.Export.App != expectedExportApp {
		return fmt.Errorf("export inesperado: app=%q (se esperaba %q)", export.Export.App, expectedExportApp)
	}
	if export.Export.FormatVersion != expectedExportFormat {
		return fmt.Errorf("format_version no soportada: %d (se esperaba %d)",
			export.Export.FormatVersion, expectedExportFormat)
	}
	for _, table := range jsonSourceTables {
		rows, ok := export.Export.Tables[table.name]
		if !ok {
			return fmt.Errorf("el export no tiene la tabla %q", table.name)
		}
		_ = rows
	}
	return nil
}

func createLegacySchema(database *sql.DB) error {
	script := `
CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS apps (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS plugins (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  version TEXT NOT NULL,
  source TEXT NOT NULL,
  description TEXT,
  status TEXT NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS accounts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  app_id INTEGER NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
  identifier TEXT NOT NULL,
  label TEXT,
  credentials TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  plugin_name TEXT REFERENCES plugins(name),
  webhook_url TEXT,
  schedule_time TEXT,
  schedule_enabled INTEGER NOT NULL DEFAULT 0,
  last_run_at DATETIME,
  schedule_frequency TEXT NOT NULL DEFAULT 'daily',
  schedule_interval INTEGER,
  schedule_days TEXT,
  UNIQUE(app_id, identifier)
);
CREATE TABLE IF NOT EXISTS bills (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  plugin_name TEXT NOT NULL,
  plugin_version TEXT NOT NULL,
  source TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ok',
  error TEXT,
  raw TEXT NOT NULL,
  fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS webhook_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
  webhook_url TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ok',
  payload TEXT NOT NULL,
  response TEXT NOT NULL,
  ran_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  http_status INTEGER
);
`
	if _, err := database.Exec(script); err != nil {
		return fmt.Errorf("crear esquema temporal: %w", err)
	}
	return nil
}

func insertJSONTables(database *sql.DB, export automationExport) error {
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("iniciar transacción temporal: %w", err)
	}
	defer tx.Rollback()

	for _, table := range jsonSourceTables {
		rows := export.Export.Tables[table.name]
		for _, row := range rows {
			// Insertar solo las columnas presentes en el JSON: las ausentes
			// usan el default de la tabla (p. ej. created_at).
			columns := make([]string, 0, len(table.columns))
			values := make([]any, 0, len(table.columns))
			placeholders := make([]string, 0, len(table.columns))
			for _, column := range table.columns {
				value, ok := row[column]
				if !ok {
					continue
				}
				columns = append(columns, column)
				values = append(values, normalizeJSONValue(value))
				placeholders = append(placeholders, "?")
			}
			if len(columns) == 0 {
				continue
			}
			query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
				table.name, strings.Join(columns, ","), strings.Join(placeholders, ","))
			if _, err := tx.Exec(query, values...); err != nil {
				return fmt.Errorf("insertar fila de %s en SQLite temporal: %w", table.name, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar SQLite temporal: %w", err)
	}
	return nil
}

// normalizeJSONValue adapta valores de encoding/json a tipos aceptados por
// SQLite. Objetos/arrays se guardan como JSON textual (credentials, raw,
// payload); los números enteros se convierten a int64.
func normalizeJSONValue(value any) any {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		return v
	case bool:
		if v {
			return int64(1)
		}
		return int64(0)
	case float64:
		if v == float64(int64(v)) {
			return int64(v)
		}
		return v
	case json.Number:
		if n, err := strconv.ParseInt(v.String(), 10, 64); err == nil {
			return n
		}
		return v.String()
	case map[string]any, []any:
		encoded, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(encoded)
	default:
		return v
	}
}
