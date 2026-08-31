package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrate015PaidAtAndReference(t *testing.T) {
	dir, err := os.MkdirTemp("", "mig015")
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

	for _, col := range []string{"paid_at", "payment_reference"} {
		var colCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('bills') WHERE name = ?", col).Scan(&colCount); err != nil {
			t.Fatal(err)
		}
		if colCount != 1 {
			t.Fatalf("columna %s no encontrada, count=%d", col, colCount)
		}
	}

	// La tabla conserva las columnas previas.
	for _, col := range []string{"status", "drive_url", "file_hash"} {
		var colCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('bills') WHERE name = ?", col).Scan(&colCount); err != nil {
			t.Fatal(err)
		}
		if colCount != 1 {
			t.Fatalf("columna %s no encontrada tras migración, count=%d", col, colCount)
		}
	}

	// Insert con paid_at y payment_reference.
	_, err = db.Exec(`INSERT INTO bills (service_id, year, month, amount, status, paid_at, payment_reference)
		VALUES (1, 2026, 8, 100, 'paid', '2026-08-30 00:00:00', 'TRX-001')`)
	if err != nil {
		t.Fatalf("insert con paid_at falló: %v", err)
	}
	var paidAt, ref string
	if err := db.QueryRow(`SELECT paid_at, payment_reference FROM bills WHERE service_id = 1`).Scan(&paidAt, &ref); err != nil {
		t.Fatal(err)
	}
	if paidAt == "" || ref != "TRX-001" {
		t.Fatalf("valores de pago no persistidos: paid_at=%q ref=%q", paidAt, ref)
	}

	// Down: revertir preservando datos.
	down, err := os.ReadFile(filepath.Join(migrationsDir, "0015_add_bills_paid_at_payment_reference.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}

	for _, col := range []string{"paid_at", "payment_reference"} {
		var colCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('bills') WHERE name = ?", col).Scan(&colCount); err != nil {
			t.Fatal(err)
		}
		if colCount != 0 {
			t.Fatalf("columna %s debería haber sido removida, count=%d", col, colCount)
		}
	}

	var billCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM bills").Scan(&billCount); err != nil {
		t.Fatal(err)
	}
	if billCount != 1 {
		t.Fatalf("se perdieron datos en el down, count=%d", billCount)
	}
}
