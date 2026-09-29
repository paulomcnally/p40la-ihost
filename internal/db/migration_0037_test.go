package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// TestMigration0037UpDown verifica que la migración 0037 (SPEC-096) aplique
// (up) y revierta (down) limpiamente, y que migre los vínculos existentes de
// categories.service_id a la tabla category_service_links.
func TestMigration0037UpDown(t *testing.T) {
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

	if hasColumn("categories", "service_id") {
		t.Error("categories.service_id no debería existir tras up de 0037")
	}
	if !hasColumn("categories", "account_id") {
		t.Error("categories.account_id debería conservarse tras up de 0037")
	}

	var linkCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM category_service_links").Scan(&linkCount); err != nil {
		t.Fatalf("contar links: %v", err)
	}
	if linkCount != 0 {
		t.Errorf("esperaba 0 links (DB limpia), got %d", linkCount)
	}
	db.Close()

	// Revertir: aplicar el .down.sql de la 0037 (simula rollback).
	db2, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	downData, err := os.ReadFile(filepath.Join(migrationsDir, "0037_add_category_service_links.down.sql"))
	if err != nil {
		t.Fatalf("leer down: %v", err)
	}
	if _, err := db2.Exec(string(downData)); err != nil {
		t.Fatalf("aplicar down: %v", err)
	}
	db2.Close()

	// Verificar que la tabla de enlace se eliminó y service_id volvió.
	db3, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("reabrir 3: %v", err)
	}
	defer db3.Close()
	var tableCount int
	if err := db3.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='category_service_links'").Scan(&tableCount); err != nil {
		t.Fatalf("contar tablas post-down: %v", err)
	}
	if tableCount != 0 {
		t.Error("category_service_links debería haber sido eliminada tras down")
	}
	if !hasColumnOn(db3, "categories", "service_id") {
		t.Error("categories.service_id debería existir tras down de 0037")
	}
}

// TestMigration0037MigratesData verifica que los vínculos existentes en
// categories.service_id (schema pre-0037) se migran a category_service_links.
func TestMigration0037MigratesData(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "app.db")
	migrationsDir, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatalf("abs migraciones: %v", err)
	}

	// DB migrada hasta la 0036 (todavía con categories.service_id).
	db, err := openMigrateUpTo(migrationsDir, dbPath, "0036")
	if err != nil {
		t.Fatalf("open+migrate up to 0036: %v", err)
	}

	// Datos de prueba: un servicio y una categoría con service_id=1.
	if _, err := db.Exec(`INSERT INTO homes (name) VALUES ('Casa')`); err != nil {
		t.Fatalf("insert home: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Internet', 'Claro', 1, 'monthly', 100, 1, 'internet', 'fixed', 1)`); err != nil {
		t.Fatalf("insert service: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO category_groups (name, icon) VALUES ('Servicios', 'home')`); err != nil {
		t.Fatalf("insert group: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO categories (category_group_id, name, icon, service_id) VALUES (1, 'Internet', 'wifi', 1)`); err != nil {
		t.Fatalf("insert category: %v", err)
	}
	db.Close()

	// Aplicar la 0037 manualmente.
	db2, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	upData, err := os.ReadFile(filepath.Join(migrationsDir, "0037_add_category_service_links.up.sql"))
	if err != nil {
		t.Fatalf("leer up: %v", err)
	}
	if _, err := db2.Exec(string(upData)); err != nil {
		t.Fatalf("aplicar up 0037: %v", err)
	}

	var linkCount int
	if err := db2.QueryRow("SELECT COUNT(*) FROM category_service_links WHERE category_id = 1 AND service_id = 1").Scan(&linkCount); err != nil {
		t.Fatalf("contar links: %v", err)
	}
	if linkCount != 1 {
		t.Errorf("esperaba 1 link migrado (categoría 1 → servicio 1), got %d", linkCount)
	}
	if hasColumnOn(db2, "categories", "service_id") {
		t.Error("categories.service_id debería haber sido eliminada tras 0037")
	}
	db2.Close()
}

func hasColumnOn(db *sql.DB, table, col string) bool {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false
		}
		if name == col {
			return true
		}
	}
	return false
}
