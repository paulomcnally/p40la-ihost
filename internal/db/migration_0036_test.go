package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestMigration0036UpDown verifica que la migración 0036 aplique (up) y
// revierta (down) limpiamente. Como la 0037 (SPEC-096) elimina luego
// categories.service_id, el down de la 0036 se prueba contra una DB migrada
// únicamente hasta la 0036 (rollback fiel del punto de la historia).
func TestMigration0036UpDown(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "app.db")
	migrationsDir, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("abs migraciones: %v", err)
	}

	// Aplicar migraciones hasta la 0036 inclusive.
	db, err := openMigrateUpTo(migrationsDir, dbPath, "0036")
	if err != nil {
		t.Fatalf("open+migrate up to 0036: %v", err)
	}

	hasColumn := func(table, col string) bool {
		rows, err := db.Query("PRAGMA table_info(" + table + ")")
		if err != nil {
			t.Fatalf("pragma %s: %v", table, err)
		}
		defer rows.Close()
		for rows.Next() {
			var cid, notnull, pk int
			var name, ctype string
			var dflt any
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
				t.Fatalf("scan: %v", err)
			}
			if name == col {
				return true
			}
		}
		return false
	}

	if !hasColumn("categories", "service_id") {
		t.Error("categories.service_id no existe tras up de 0036")
	}
	if !hasColumn("categories", "account_id") {
		t.Error("categories.account_id no existe tras up de 0036")
	}
	if !hasColumn("transactions", "source_bill_id") {
		t.Error("transactions.source_bill_id no existe tras up de 0036")
	}
	db.Close()

	// Revertir: aplicar el .down.sql de la 0036 (rollback).
	db2, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	downData, err := os.ReadFile(filepath.Join(migrationsDir, "0036_add_category_service_link.down.sql"))
	if err != nil {
		t.Fatalf("leer down: %v", err)
	}
	if _, err := db2.Exec(string(downData)); err != nil {
		t.Fatalf("aplicar down: %v", err)
	}
	db2.Close()

	// Verificar que las columnas se eliminaron (DROP COLUMN ok).
	db3, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("reabrir 3: %v", err)
	}
	defer db3.Close()
	rows, err := db3.Query("PRAGMA table_info(categories)")
	if err != nil {
		t.Fatalf("pragma post-down: %v", err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan post-down: %v", err)
		}
		if name == "service_id" || name == "account_id" {
			found = true
		}
	}
	if found {
		t.Error("categories.service_id/account_id siguen existiendo tras down de 0036")
	}
}

// openMigrateUpTo crea una DB y aplica solo las migraciones .up.sql cuyo nombre
// es <= al prefijo dado (ej. "0036").
func openMigrateUpTo(migrationsDir, dbPath, upTo string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)`); err != nil {
		db.Close()
		return nil, err
	}
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		db.Close()
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		prefix := name[:4]
		if prefix > upTo {
			continue
		}
		data, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			db.Close()
			return nil, err
		}
		if _, err := db.Exec(string(data)); err != nil {
			db.Close()
			return nil, err
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", name); err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}
