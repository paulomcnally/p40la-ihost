package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// TestMigration0038UpDown verifica que la migración 0038 (SPEC-099) cree la
// tabla transfers (up) y la elimine (down) limpiamente, con sus checks.
func TestMigration0038UpDown(t *testing.T) {
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

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='transfers'").Scan(&count); err != nil {
		t.Fatalf("contar tabla transfers: %v", err)
	}
	if count != 1 {
		t.Fatalf("tabla transfers debería existir tras up de 0038")
	}

	// Checks: origen != destino y amount > 0.
	if _, err := db.Exec(`INSERT INTO accounts (name, type, currency_id) VALUES ('A','checking',1), ('B','checking',1)`); err != nil {
		t.Fatalf("insertar cuentas de prueba: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO transfers (from_account_id, to_account_id, currency_id, date, amount) VALUES (1, 1, 1, '2026-10-15', 100)`); err == nil {
		t.Error("debería rechazar transferencia con misma cuenta origen/destino")
	}
	if _, err := db.Exec(`INSERT INTO transfers (from_account_id, to_account_id, currency_id, date, amount) VALUES (1, 2, 1, '2026-10-15', 0)`); err == nil {
		t.Error("debería rechazar transferencia con amount <= 0")
	}
	if _, err := db.Exec(`INSERT INTO transfers (from_account_id, to_account_id, currency_id, date, amount) VALUES (1, 2, 1, '2026-10-15', 100)`); err != nil {
		t.Fatalf("debería aceptar transferencia válida: %v", err)
	}
	db.Close()

	// Revertir.
	db2, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	downData, err := os.ReadFile(filepath.Join(migrationsDir, "0038_create_transfers.down.sql"))
	if err != nil {
		t.Fatalf("leer down: %v", err)
	}
	if _, err := db2.Exec(string(downData)); err != nil {
		t.Fatalf("aplicar down: %v", err)
	}
	db2.Close()

	db3, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("reabrir 3: %v", err)
	}
	defer db3.Close()
	if err := db3.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='transfers'").Scan(&count); err != nil {
		t.Fatalf("contar tablas post-down: %v", err)
	}
	if count != 0 {
		t.Error("transfers debería haber sido eliminada tras down")
	}
}