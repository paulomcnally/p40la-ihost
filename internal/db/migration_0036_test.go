package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// TestMigration0036UpDown verifica que la migración 0036 aplique (up) y
// revierta (down) limpiamente en una DB temporal en disco.
func TestMigration0036UpDown(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "app.db")
	migrationsDir, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("abs migraciones: %v", err)
	}

	db, err := OpenDB(dbPath, migrationsDir)
	if err != nil {
		t.Fatalf("open+migrate up: %v", err)
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
		t.Error("categories.service_id no existe tras up")
	}
	if !hasColumn("categories", "account_id") {
		t.Error("categories.account_id no existe tras up")
	}
	if !hasColumn("transactions", "source_bill_id") {
		t.Error("transactions.source_bill_id no existe tras up")
	}
	db.Close()

	// Revertir: aplicar el .down.sql de la 0036 manualmente (simula rollback).
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
		t.Error("categories.service_id/account_id siguen existiendo tras down")
	}
}
