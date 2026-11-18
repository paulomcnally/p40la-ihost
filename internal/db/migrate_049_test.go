package db

import "testing"

func TestMigrate049CreditCardLimitCurrency(t *testing.T) {
	database := openMigrateTest(t, "mig049")

	assertTableColumns(t, database, "credit_cards", []string{
		"limit_currency",
		"exchange_rate",
		"bonus_due_day",
		"category",
	})

	// Los defaults respetan las tarjetas existentes (SPEC-138).
	var limitCurrency string
	var exchangeRate float64
	var bonusDueDay int
	var category string
	if err := database.QueryRow(`
		SELECT limit_currency, exchange_rate, bonus_due_day, category
		FROM credit_cards LIMIT 1
	`).Scan(&limitCurrency, &exchangeRate, &bonusDueDay, &category); err == nil {
		if limitCurrency != "NIO" {
			t.Fatalf("limit_currency default = %q, esperaba NIO", limitCurrency)
		}
		if exchangeRate != 0 || bonusDueDay != 0 || category != "" {
			t.Fatalf("defaults inesperados: %v %v %q", exchangeRate, bonusDueDay, category)
		}
	}

	// La moneda del límite está acotada por CHECK.
	if _, err := database.Exec(`
		INSERT INTO credit_cards (bank_id, alias, network, last4, currency_id, limit_currency)
		VALUES (999, 'Demo', 'visa', '0000', 1, 'EUR')
	`); err == nil {
		t.Fatal("esperaba CHECK de limit_currency")
	}

	// bonus_due_day no admite valores fuera de 0..31.
	if _, err := database.Exec(`
		INSERT INTO credit_cards (bank_id, alias, network, last4, currency_id, bonus_due_day)
		VALUES (999, 'Demo 2', 'visa', '0001', 1, 32)
	`); err == nil {
		t.Fatal("esperaba CHECK de bonus_due_day")
	}
}
