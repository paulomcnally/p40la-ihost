package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrate014FileHash(t *testing.T) {
	dir, err := os.MkdirTemp("", "mig014")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	migrationsDir, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(dir, "test.db")
	dsn := "file:" + dbPath
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db, migrationsDir); err != nil {
		t.Fatal(err)
	}

	var colCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('bills') WHERE name = 'file_hash'").Scan(&colCount); err != nil {
		t.Fatal(err)
	}
	if colCount != 1 {
		t.Fatalf("columna file_hash no encontrada, count=%d", colCount)
	}

	var idxCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_bills_service_file_hash'").Scan(&idxCount); err != nil {
		t.Fatal(err)
	}
	if idxCount != 1 {
		t.Fatalf("indice idx_bills_service_file_hash no encontrado, count=%d", idxCount)
	}

	_, err = db.Exec(`INSERT INTO bills (service_id, year, month, amount, file_hash) VALUES (1, 2026, 1, 100, 'abc')`)
	if err != nil {
		t.Fatalf("insert con file_hash falló: %v", err)
	}
	_, err = db.Exec(`INSERT INTO bills (service_id, year, month, amount, file_hash) VALUES (1, 2026, 2, 100, 'abc')`)
	if err == nil {
		t.Fatalf("insert duplicado con mismo service_id+file_hash debería fallar por índice único")
	}
	_, err = db.Exec(`INSERT INTO bills (service_id, year, month, amount, file_hash) VALUES (2, 2026, 1, 100, 'abc')`)
	if err != nil {
		t.Fatalf("mismo hash en otro servicio debería permitirse: %v", err)
	}
	_, err = db.Exec(`INSERT INTO bills (service_id, year, month, amount, file_hash) VALUES (3, 2026, 1, 100, NULL)`)
	if err != nil {
		t.Fatalf("file_hash NULL debería permitirse: %v", err)
	}
	_, err = db.Exec(`INSERT INTO bills (service_id, year, month, amount, file_hash) VALUES (3, 2026, 2, 100, '')`)
	if err != nil {
		t.Fatalf("file_hash vacío debería permitirse (manual): %v", err)
	}
	_, err = db.Exec(`INSERT INTO bills (service_id, year, month, amount, file_hash) VALUES (3, 2026, 3, 100, '')`)
	if err != nil {
		t.Fatalf("varias facturas manuales con hash vacío deberían permitirse: %v", err)
	}

	// Down: revertir la migración preservando los datos.
	down, err := os.ReadFile(filepath.Join(migrationsDir, "0014_add_bills_file_hash.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}

	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('bills') WHERE name = 'file_hash'").Scan(&colCount); err != nil {
		t.Fatal(err)
	}
	if colCount != 0 {
		t.Fatalf("columna file_hash debería haber sido removida, count=%d", colCount)
	}

	var billCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM bills").Scan(&billCount); err != nil {
		t.Fatal(err)
	}
	if billCount != 5 {
		t.Fatalf("se perdieron datos en el down, count=%d", billCount)
	}
}
