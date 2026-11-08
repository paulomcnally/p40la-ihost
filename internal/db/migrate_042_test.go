package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrate042DocumentRecognitions(t *testing.T) {
	dir, err := os.MkdirTemp("", "mig042")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	migrationsDir, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(dir, "test.db")
	database, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(database, migrationsDir); err != nil {
		t.Fatal(err)
	}

	var tableCount int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='document_recognitions'",
	).Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if tableCount != 1 {
		t.Fatalf("tabla document_recognitions no encontrada, count=%d", tableCount)
	}

	for _, index := range []string{
		"idx_document_recognitions_status",
		"idx_document_recognitions_reference",
		"idx_document_recognitions_entity",
	} {
		var count int
		if err := database.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", index,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("índice %s no encontrado", index)
		}
	}

	if _, err := database.Exec(`
		INSERT INTO document_recognitions (document_hash, recognizer_id, document_type, status, reference)
		VALUES ('demo-hash-001', 'bank_receipt_v1', 'nota_debito', 'recognized', '902000001')
	`); err != nil {
		t.Fatalf("insertar reconocimiento: %v", err)
	}

	if _, err := database.Exec(`
		INSERT INTO document_recognitions (document_hash, recognizer_id, document_type, status, reference)
		VALUES ('demo-hash-001', 'bank_receipt_v1', 'nota_debito', 'recognized', '902000002')
	`); err == nil {
		t.Fatal("el hash duplicado debería fallar por restricción única")
	}

	if _, err := database.Exec(`
		INSERT INTO document_recognitions (document_hash, recognizer_id, document_type, status)
		VALUES ('demo-hash-002', 'bank_receipt_v1', 'nota_debito', 'inventado')
	`); err == nil {
		t.Fatal("un estado inválido debería fallar por CHECK")
	}

	// Down: la tabla se elimina y no se tocan las tablas de deudas.
	down, err := os.ReadFile(filepath.Join(migrationsDir, "0042_create_document_recognitions.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='document_recognitions'",
	).Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if tableCount != 0 {
		t.Fatalf("la tabla document_recognitions debería haberse eliminado, count=%d", tableCount)
	}

	var debtBillsCount int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='debt_bills'",
	).Scan(&debtBillsCount); err != nil {
		t.Fatal(err)
	}
	if debtBillsCount != 1 {
		t.Fatalf("la tabla debt_bills no debería verse afectada, count=%d", debtBillsCount)
	}
}
