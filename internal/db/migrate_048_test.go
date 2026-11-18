package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrate048CreditCards(t *testing.T) {
	dir, err := os.MkdirTemp("", "mig048")
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

	for _, table := range []string{"credit_cards", "card_statements", "card_transactions"} {
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

	for _, index := range []string{
		"idx_credit_cards_bank",
		"idx_card_statements_card",
		"idx_card_statements_import",
		"idx_card_transactions_card",
		"idx_card_transactions_statement",
		"idx_card_transactions_ref",
	} {
		var count int
		if err := database.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", index,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("índice %s no encontrado", index)
		}
	}

	// last4 debe rechazar algo que no sean 4 dígitos.
	if _, err := database.Exec(`
		INSERT INTO credit_cards (bank_id, alias, network, last4, currency_id)
		VALUES (999, 'Demo', 'visa', '12', 1)
	`); err == nil {
		t.Fatal("esperaba CHECK de last4")
	}
}
