package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const syntheticExport = `{
  "export": {
    "format_version": 1,
    "app": "p40la-ihost-automation",
    "tables": {
      "settings": [
        {"key": "webhook_enabled", "value": "1"}
      ],
      "apps": [
        {"id": 1, "name": "Proveedor Demo"}
      ],
      "plugins": [
        {"id": 1, "name": "demo.provider", "version": "1.0.0", "source": "demo",
         "description": "Plugin demo", "status": "active"}
      ],
      "accounts": [
        {"id": 1, "app_id": 1, "identifier": "demo-account-001", "label": "Cuenta Demo",
         "credentials": {"user": "demo"}, "plugin_name": "demo.provider",
         "webhook_url": "http://demo.local/webhooks/00000000-0000-4000-8000-000000000000",
         "schedule_time": "07:00", "schedule_enabled": 1, "last_run_at": null,
         "schedule_frequency": "daily", "schedule_interval": null, "schedule_days": null}
      ],
      "bills": [
        {"id": 1, "account_id": 1, "plugin_name": "demo.provider", "plugin_version": "1.0.0",
         "source": "demo", "status": "ok", "error": null,
         "raw": {"numFactura": "F000000000000001"}}
      ],
      "webhook_logs": [
        {"id": 1, "account_id": 1,
         "webhook_url": "http://demo.local/webhooks/00000000-0000-4000-8000-000000000000",
         "status": "ok", "payload": {"year": 2026, "month": 9},
         "response": "upsert en proceso", "http_status": null}
      ]
    }
  }
}`

func writeJSONFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "export.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("escribir fixture: %v", err)
	}
	return path
}

func openDestination(t *testing.T, dataDir string) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", "file:"+filepath.Join(dataDir, "app.db")+"?mode=ro")
	if err != nil {
		t.Fatalf("abrir destino: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func countRows(t *testing.T, database *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("contar %s: %v", table, err)
	}
	return n
}

func TestImportJSONSource(t *testing.T) {
	jsonPath := writeJSONFixture(t, syntheticExport)
	dataDir := t.TempDir()
	opts := options{
		sourceJSONPath: jsonPath,
		dataDir:        dataDir,
		migrationsDir:  migrationsDir,
		verify:         true,
	}

	if err := runImport(context.Background(), opts); err != nil {
		t.Fatalf("importar JSON: %v", err)
	}

	dst := openDestination(t, dataDir)
	if got := countRows(t, dst, "automation_accounts"); got != 1 {
		t.Errorf("accounts = %d, want 1", got)
	}
	if got := countRows(t, dst, "automation_bills"); got != 1 {
		t.Errorf("bills = %d, want 1", got)
	}
	if got := countRows(t, dst, "automation_webhook_logs"); got != 1 {
		t.Errorf("logs = %d, want 1", got)
	}
	if got := countRows(t, dst, "automation_plugins"); got != 1 {
		t.Errorf("plugins = %d, want 1", got)
	}

	var credentials, raw string
	if err := dst.QueryRow("SELECT credentials FROM automation_accounts WHERE id = 1").Scan(&credentials); err != nil {
		t.Fatalf("leer credenciales: %v", err)
	}
	if !strings.Contains(credentials, "demo") {
		t.Errorf("credenciales no serializadas: %q", credentials)
	}
	if err := dst.QueryRow("SELECT raw FROM automation_bills WHERE id = 1").Scan(&raw); err != nil {
		t.Fatalf("leer raw: %v", err)
	}
	if !strings.Contains(raw, "F000000000000001") {
		t.Errorf("raw no serializado: %q", raw)
	}

	// Idempotencia: una segunda corrida no duplica.
	if err := runImport(context.Background(), opts); err != nil {
		t.Fatalf("reejecutar importación: %v", err)
	}
	dst2 := openDestination(t, dataDir)
	if got := countRows(t, dst2, "automation_accounts"); got != 1 {
		t.Errorf("accounts tras reejecutar = %d, want 1", got)
	}
	if got := countRows(t, dst2, "automation_bills"); got != 1 {
		t.Errorf("bills tras reejecutar = %d, want 1", got)
	}
}

func TestMaterializeJSONSourceCleanup(t *testing.T) {
	jsonPath := writeJSONFixture(t, syntheticExport)
	dbPath, cleanup, err := materializeJSONSource(jsonPath)
	if err != nil {
		t.Fatalf("materializar: %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("la SQLite temporal no existe: %v", err)
	}
	cleanup()
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Errorf("la SQLite temporal no se eliminó: %v", err)
	}
}

func TestMaterializeJSONSourceRejectsInvalidExport(t *testing.T) {
	cases := map[string]string{
		"app":     strings.Replace(syntheticExport, `"p40la-ihost-automation"`, `"otra-app"`, 1),
		"version": strings.Replace(syntheticExport, `"format_version": 1`, `"format_version": 99`, 1),
		"tables":  strings.Replace(syntheticExport, `"bills":`, `"bills_ausentes":`, 1),
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			jsonPath := writeJSONFixture(t, content)
			if _, cleanup, err := materializeJSONSource(jsonPath); err == nil {
				cleanup()
				t.Fatal("se esperaba error")
			}
		})
	}
}

func TestImportRequiresExactlyOneSource(t *testing.T) {
	if err := runImport(context.Background(), options{}); err == nil {
		t.Fatal("sin fuente se esperaba error")
	}
	if err := runImport(context.Background(), options{
		sourcePath:     "a.db",
		sourceJSONPath: "a.json",
	}); err == nil {
		t.Fatal("con dos fuentes se esperaba error")
	}
}
