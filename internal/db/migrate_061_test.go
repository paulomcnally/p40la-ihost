package db

import "testing"

// TestMigrate061BankAccountIBAN verifica la columna iban de bank_accounts
// (SPEC-162): existe tras migrar, arranca en ” y el down la remueve.
func TestMigrate061BankAccountIBAN(t *testing.T) {
	database := openMigrateTest(t, "mig061")

	assertTableColumns(t, database, "bank_accounts", []string{"iban"})

	var defaultValue string
	var notNull int
	if err := database.QueryRow(`
		SELECT COALESCE(dflt_value, ''), "notnull"
		FROM pragma_table_info('bank_accounts')
		WHERE name = 'iban'
	`).Scan(&defaultValue, &notNull); err != nil {
		t.Fatalf("leer columna iban: %v", err)
	}
	if defaultValue != "''" {
		t.Fatalf("default de iban: esperado '', got %q", defaultValue)
	}
	if notNull != 1 {
		t.Fatalf("iban debería ser NOT NULL, notnull=%d", notNull)
	}

	if _, err := database.Exec(`
		INSERT INTO banks (name, short_code, country, notes, active)
		VALUES ('Banco Demo', 'BDEMO', 'NI', 'Banco sintético', 1)
	`); err != nil {
		t.Fatalf("insertar banco: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO bank_accounts
			(bank_id, alias, account_number, account_type, currency_id, beneficiary, notes, active)
		SELECT id, 'Cuenta Demo', 'demo-account-001', 'savings',
		       (SELECT id FROM currencies WHERE code = 'NIO'), 'Titular Demo', '', 1
		FROM banks WHERE name = 'Banco Demo'
	`); err != nil {
		t.Fatalf("insertar cuenta sin iban: %v", err)
	}
	var iban string
	if err := database.QueryRow(`SELECT iban FROM bank_accounts WHERE account_number = 'demo-account-001'`).Scan(&iban); err != nil {
		t.Fatalf("leer iban: %v", err)
	}
	if iban != "" {
		t.Fatalf("iban por defecto: esperado vacío, got %q", iban)
	}

	down, err := readMigrationFile(t, "0061_add_bank_account_iban.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(down)); err != nil {
		t.Fatalf("ejecutar migración 0061 down: %v", err)
	}
	var count int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('bank_accounts') WHERE name = 'iban'",
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("iban debería haberse removido, count=%d", count)
	}
}
