package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrate030BillsDebtBillsDates(t *testing.T) {
	dir, err := os.MkdirTemp("", "mig030")
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

	// Columnas nuevas presentes.
	for _, col := range []string{"issue_date", "due_date"} {
		var colCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('bills') WHERE name = ?", col).Scan(&colCount); err != nil {
			t.Fatal(err)
		}
		if colCount != 1 {
			t.Fatalf("columna %s no encontrada en bills, count=%d", col, colCount)
		}
	}
	var issueCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('debt_bills') WHERE name = 'issue_date'").Scan(&issueCount); err != nil {
		t.Fatal(err)
	}
	if issueCount != 1 {
		t.Fatalf("columna issue_date no encontrada en debt_bills, count=%d", issueCount)
	}

	// Insert con las fechas nuevas en bills.
	if _, err := db.Exec(`INSERT INTO bills (service_id, year, month, amount, status, issue_date, due_date)
		VALUES (1, 2026, 8, 100, 'pending', '2026-09-10', '2026-10-05')`); err != nil {
		t.Fatalf("insert con fechas falló: %v", err)
	}
	var issue, due string
	if err := db.QueryRow(`SELECT issue_date, due_date FROM bills WHERE service_id = 1`).Scan(&issue, &due); err != nil {
		t.Fatal(err)
	}
	if issue != "2026-09-10" || due != "2026-10-05" {
		t.Fatalf("fechas no persistidas: issue=%q due=%q", issue, due)
	}

	// Insert con issue_date en debt_bills (due_date preexistente se conserva).
	if _, err := db.Exec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, issue_date, amount, status)
		VALUES (1, 1, '2026-09-05', '2026-09-01', 100, 'pending')`); err != nil {
		t.Fatalf("insert debt_bills con issue_date falló: %v", err)
	}

	// Down: revertir preservando datos.
	down, err := os.ReadFile(filepath.Join(migrationsDir, "0030_add_bills_debt_bills_dates.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}

	for _, col := range []string{"issue_date", "due_date"} {
		var colCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('bills') WHERE name = ?", col).Scan(&colCount); err != nil {
			t.Fatal(err)
		}
		if colCount != 0 {
			t.Fatalf("columna %s debería haber sido removida de bills, count=%d", col, colCount)
		}
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('debt_bills') WHERE name = 'issue_date'").Scan(&issueCount); err != nil {
		t.Fatal(err)
	}
	if issueCount != 0 {
		t.Fatalf("columna issue_date debería haber sido removida de debt_bills, count=%d", issueCount)
	}

	var billCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM bills").Scan(&billCount); err != nil {
		t.Fatal(err)
	}
	if billCount != 1 {
		t.Fatalf("se perdieron datos en el down, count=%d", billCount)
	}
	var debtBillCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM debt_bills").Scan(&debtBillCount); err != nil {
		t.Fatal(err)
	}
	if debtBillCount != 1 {
		t.Fatalf("se perdieron cuotas en el down, count=%d", debtBillCount)
	}
}
