package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func openMigrateTest(t *testing.T, suffix string) *sql.DB {
	t.Helper()
	dir, err := os.MkdirTemp("", suffix)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

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
	t.Cleanup(func() { db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db, migrationsDir); err != nil {
		t.Fatal(err)
	}
	return db
}

func assertTableColumns(t *testing.T, db *sql.DB, table string, cols []string) {
	t.Helper()
	for _, col := range cols {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('"+table+"') WHERE name = ?", col).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("columna %s.%s no encontrada, count=%d", table, col, count)
		}
	}
}

func assertTableRemoved(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("la tabla %s debería haber sido removida, count=%d", table, count)
	}
}

func TestMigrate020CreateSupportRecords(t *testing.T) {
	db := openMigrateTest(t, "mig020")

	assertTableColumns(t, db, "support_records", []string{
		"id", "child_id", "pension_category_id", "year", "month", "amount",
		"currency", "status", "paid_at", "payment_method", "payment_reference",
		"evidence_notes", "notes", "proof_file_name", "original_amount",
		"original_currency", "exchange_rate", "created_at", "updated_at",
	})

	// Insert con dependencias: children y pension_categories ya existen.
	_, err := db.Exec(`INSERT INTO children (first_name, last_name, birth_date) VALUES ('Juan', 'Pérez', '2015-01-01')`)
	if err != nil {
		t.Fatalf("insert hijo falló: %v", err)
	}
	_, err = db.Exec(`INSERT INTO pension_categories (name) VALUES ('Colegio')`)
	if err != nil {
		t.Fatalf("insert categoría falló: %v", err)
	}
	_, err = db.Exec(`INSERT INTO support_records (child_id, pension_category_id, year, month, amount) VALUES (1, 1, 2026, 8, 1500.00)`)
	if err != nil {
		t.Fatalf("insert registro falló: %v", err)
	}

	var status, currency string
	if err := db.QueryRow(`SELECT status, currency FROM support_records WHERE id = 1`).Scan(&status, &currency); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || currency != "NIO" {
		t.Fatalf("defaults incorrectos: status=%s currency=%s", status, currency)
	}

	// UNIQUE (child_id, pension_category_id, year, month)
	_, err = db.Exec(`INSERT INTO support_records (child_id, pension_category_id, year, month, amount) VALUES (1, 1, 2026, 8, 900.00)`)
	if err == nil {
		t.Fatal("se esperaba error UNIQUE por duplicado de hijo+categoría+período")
	}

	down, err := os.ReadFile(filepath.Join("../../migrations", "0020_create_support_records.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}
	assertTableRemoved(t, db, "support_records")
}

func TestMigrate021CreateSalaryPayments(t *testing.T) {
	db := openMigrateTest(t, "mig021")

	assertTableColumns(t, db, "salary_payments", []string{
		"id", "salary_id", "year", "month", "amount", "currency",
		"status", "received_amount", "received_at", "notes", "created_at", "updated_at",
	})

	_, err := db.Exec(`INSERT INTO salaries (employer, amount, currency_id, payment_day) VALUES ('Empresa XYZ', 15000, 1, 15)`)
	if err != nil {
		t.Fatalf("insert salario falló: %v", err)
	}
	_, err = db.Exec(`INSERT INTO salary_payments (salary_id, year, month, amount, currency) VALUES (1, 2026, 8, 15000, 'NIO')`)
	if err != nil {
		t.Fatalf("insert pago falló: %v", err)
	}

	// UNIQUE (salary_id, year, month)
	_, err = db.Exec(`INSERT INTO salary_payments (salary_id, year, month, amount) VALUES (1, 2026, 8, 15000)`)
	if err == nil {
		t.Fatal("se esperaba error UNIQUE por duplicado de salario+período")
	}

	down, err := os.ReadFile(filepath.Join("../../migrations", "0021_create_salary_payments.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}
	assertTableRemoved(t, db, "salary_payments")
}

func TestMigrate022CreateMonthClosings(t *testing.T) {
	db := openMigrateTest(t, "mig022")

	assertTableColumns(t, db, "month_closings", []string{"id", "year", "month", "closed_at"})

	_, err := db.Exec(`INSERT INTO month_closings (year, month) VALUES (2026, 8)`)
	if err != nil {
		t.Fatalf("insert cierre falló: %v", err)
	}

	// UNIQUE (year, month)
	_, err = db.Exec(`INSERT INTO month_closings (year, month) VALUES (2026, 8)`)
	if err == nil {
		t.Fatal("se esperaba error UNIQUE por duplicado de año+mes")
	}

	var closedAt string
	if err := db.QueryRow(`SELECT closed_at FROM month_closings WHERE year = 2026 AND month = 8`).Scan(&closedAt); err != nil {
		t.Fatal(err)
	}
	if closedAt == "" {
		t.Fatal("closed_at debería tener valor por defecto")
	}

	down, err := os.ReadFile(filepath.Join("../../migrations", "0022_create_month_closings.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}
	assertTableRemoved(t, db, "month_closings")
}
