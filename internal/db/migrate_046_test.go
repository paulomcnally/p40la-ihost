package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrate046DebtInformationTemplates(t *testing.T) {
	dir, err := os.MkdirTemp("", "mig046")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	migrationsDir, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(dir, "test.db")
	database, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)")
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

	for _, table := range []string{"debt_information_templates", "debt_payment_information"} {
		var count int
		if err := database.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("tabla %s no encontrada, count=%d", table, count)
		}
	}

	var indexCount int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_debt_payment_information_template'",
	).Scan(&indexCount); err != nil {
		t.Fatal(err)
	}
	if indexCount != 1 {
		t.Fatal("índice idx_debt_payment_information_template no encontrado")
	}

	if _, err := database.Exec(`INSERT INTO institutions (name) VALUES ('Banco Demo')`); err != nil {
		t.Fatalf("insertar institución: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO debts (institution_id, identifier, description, total, principal,
		                   currency_id, installments_total, installment_amount,
		                   interest_rate, payment_day, start_date, status)
		VALUES (1, 'demo-account-001', 'Lote Demo 1', 100, 100, 1, 1, 100, 0, 5, '2026-09-01', 'activa')
	`); err != nil {
		t.Fatalf("insertar deuda: %v", err)
	}

	if _, err := database.Exec(`
		INSERT INTO debt_information_templates (name, description, fields_json)
		VALUES ('Transferencia bancaria', 'Datos de prueba', '[{"key":"banco","label":"Banco","type":"text"}]')
	`); err != nil {
		t.Fatalf("insertar plantilla: %v", err)
	}

	if _, err := database.Exec(`
		INSERT INTO debt_information_templates (name, description, fields_json)
		VALUES ('transferencia BANCARIA', 'Duplicada', '[]')
	`); err == nil {
		t.Fatal("el nombre duplicado sin distinguir mayúsculas debería fallar")
	}

	if _, err := database.Exec(`
		INSERT INTO debt_payment_information (debt_id, template_id, values_json)
		VALUES (1, 1, '{"banco":"Banco Demo"}')
	`); err != nil {
		t.Fatalf("insertar forma de pago: %v", err)
	}

	if _, err := database.Exec(`DELETE FROM debt_information_templates WHERE id = 1`); err == nil {
		t.Fatal("no se debería poder borrar una plantilla en uso (FK RESTRICT)")
	}

	if _, err := database.Exec(`DELETE FROM debts WHERE id = 1`); err != nil {
		t.Fatalf("borrar deuda: %v", err)
	}
	var paymentCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM debt_payment_information`).Scan(&paymentCount); err != nil {
		t.Fatal(err)
	}
	if paymentCount != 0 {
		t.Fatalf("la forma de pago debería eliminarse en cascada, count=%d", paymentCount)
	}

	// Down: se eliminan solo las tablas nuevas y no se tocan debts/debt_bills.
	down, err := os.ReadFile(filepath.Join(migrationsDir, "0046_create_debt_information_templates.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}
	for _, table := range []string{"debt_information_templates", "debt_payment_information"} {
		var count int
		if err := database.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("la tabla %s debería haberse eliminado, count=%d", table, count)
		}
	}
	for _, table := range []string{"debts", "debt_bills"} {
		var count int
		if err := database.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("la tabla %s no debería verse afectada, count=%d", table, count)
		}
	}
}
