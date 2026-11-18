package db

import "testing"

func TestMigrate050CardStatementAggregates(t *testing.T) {
	database := openMigrateTest(t, "mig050")

	cols := []string{
		"interest_bonifiable",
		"interest_bonifiable_usd",
		"mora_amount",
		"mora_amount_usd",
		"arrears_count",
		"over_limit_amount",
		"over_limit_amount_usd",
		"points_previous",
		"points_earned",
		"points_used",
		"points_available",
	}
	assertTableColumns(t, database, "card_statements", cols)

	// Los estados existentes conservan sus datos y los nuevos arrancan en 0.
	if _, err := database.Exec(`
		INSERT INTO card_statements (
			card_id, period_end, min_payment_due_date, full_payment_due_date
		) VALUES (1, '2026-11-17', '2026-12-14', '2026-11-17')
	`); err != nil {
		t.Fatalf("insertar estado: %v", err)
	}

	var bonifiable, mora, overLimit float64
	var arrears, points int
	if err := database.QueryRow(`
		SELECT interest_bonifiable, mora_amount, over_limit_amount, arrears_count, points_available
		FROM card_statements WHERE period_end = '2026-11-17'
	`).Scan(&bonifiable, &mora, &overLimit, &arrears, &points); err != nil {
		t.Fatalf("leer defaults: %v", err)
	}
	if bonifiable != 0 || mora != 0 || overLimit != 0 || arrears != 0 || points != 0 {
		t.Fatalf("defaults inesperados: %v %v %v %v %v", bonifiable, mora, overLimit, arrears, points)
	}
}
