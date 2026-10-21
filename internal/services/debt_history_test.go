package services

import (
	"context"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// SPEC-105: la historia cuenta deudas abiertas por mes, cierra desde el mes del
// pago/eliminación y conserva los meses anteriores.
func TestDebtHistoryMonthlyCounts(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("ejecutar %q: %v", query, err)
		}
	}

	mustExec("INSERT INTO institutions (name) VALUES ('Banco Uno')")
	insertDebt := func(id int, currencyID int, start, status, deletedAt string) {
		t.Helper()
		var deleted any
		if deletedAt != "" {
			deleted = deletedAt
		}
		mustExec(`INSERT INTO debts (id, institution_id, identifier, description, total, principal,
			currency_id, installments_total, installment_amount, interest_rate, payment_day, start_date, status, deleted_at)
			VALUES (?, 1, ?, ?, 100, 100, ?, 1, 100, 0, 5, ?, ?, ?)`,
			id, "D-"+string(rune('0'+id)), "Deuda "+string(rune('0'+id)), currencyID, start, status, deleted)
	}
	insertBill := func(debtID, number int, status, paidAt, dueDate string, amount float64) {
		t.Helper()
		mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status, paid_at)
			VALUES (?, ?, ?, ?, ?, ?)`, debtID, number, dueDate, amount, status, nullableString(paidAt))
	}

	// 1: abierta durante toda la ventana.
	insertDebt(1, 1, "2025-01-01", "activa", "")
	insertBill(1, 1, "pending", "", "2026-11-05", 50)
	insertBill(1, 2, "pending", "", "2026-12-05", 50)

	// 2: finalizada con último pago en marzo 2026.
	insertDebt(2, 1, "2025-06-01", "finalizada", "")
	insertBill(2, 1, "paid", "2026-03-15 00:00:00", "2026-03-05", 100)

	// 3: inactiva pero completa (sin pendientes, con pago) → cerrada en julio.
	insertDebt(3, 1, "2025-01-01", "inactiva", "")
	insertBill(3, 1, "paid", "2026-07-20 00:00:00 +0000 UTC", "2026-07-05", 100)

	// 4: eliminada en septiembre 2026.
	insertDebt(4, 1, "2025-01-01", "activa", "2026-09-10 12:00:00")
	insertBill(4, 1, "pending", "", "2027-01-05", 100)

	// 5: inicia en agosto 2026.
	insertDebt(5, 1, "2026-08-01", "activa", "")
	insertBill(5, 1, "pending", "", "2026-12-05", 30)

	// 6: eliminada antes de la ventana → no aparece.
	insertDebt(6, 1, "2025-01-01", "activa", "2025-05-10 00:00:00")
	insertBill(6, 1, "pending", "", "2026-12-05", 100)

	// 7: inicia en el futuro → no aparece.
	insertDebt(7, 1, "2027-01-01", "activa", "")
	insertBill(7, 1, "pending", "", "2027-02-05", 100)

	// 8: otra moneda, abierta toda la ventana.
	insertDebt(8, 2, "2025-01-01", "activa", "")
	insertBill(8, 1, "pending", "", "2026-11-05", 50)

	svc := NewDebtService(
		storage.NewDebtStorage(database),
		storage.NewDebtBillStorage(database),
		storage.NewInstitutionStorage(database),
		storage.NewCurrencyStorage(database),
	)

	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	history, err := svc.History(ctx, 12, now)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if history.Months != 12 || len(history.Currencies) != 2 {
		t.Fatalf("got months=%d currencies=%d, want 12/2", history.Months, len(history.Currencies))
	}

	nio := findCurrency(t, history.Currencies, "NIO")
	wantOpen := []int{4, 4, 4, 4, 3, 3, 3, 3, 2, 3, 2, 2}
	wantClosed := []int{0, 0, 0, 0, 1, 0, 0, 0, 1, 0, 1, 0}
	for i, p := range nio.Series {
		if p.OpenCount != wantOpen[i] || p.ClosedCount != wantClosed[i] {
			t.Errorf("NIO %s: got open=%d closed=%d, want open=%d closed=%d",
				p.Month, p.OpenCount, p.ClosedCount, wantOpen[i], wantClosed[i])
		}
	}
	if nio.Series[0].Month != "2025-11" || nio.Series[11].Month != "2026-10" {
		t.Errorf("ventana: got %s..%s, want 2025-11..2026-10", nio.Series[0].Month, nio.Series[11].Month)
	}
	if nio.Summary.StartCount != 4 || nio.Summary.EndCount != 2 || nio.Summary.ChangeCount != -2 {
		t.Errorf("summary counts: got start=%d end=%d change=%d", nio.Summary.StartCount, nio.Summary.EndCount, nio.Summary.ChangeCount)
	}
	if nio.Summary.ClosedCount != 3 {
		t.Errorf("closed: got %d, want 3", nio.Summary.ClosedCount)
	}
	if nio.Summary.ChangePct == nil || *nio.Summary.ChangePct != -50 {
		t.Errorf("change_pct: got %v, want -50", nio.Summary.ChangePct)
	}
	if nio.Summary.PendingAmountCurrent != 130 {
		t.Errorf("pending current: got %v, want 130", nio.Summary.PendingAmountCurrent)
	}

	usd := findCurrency(t, history.Currencies, "USD")
	if usd.Summary.StartCount != 1 || usd.Summary.EndCount != 1 || usd.Summary.ChangeCount != 0 {
		t.Errorf("USD summary: got %+v", usd.Summary)
	}
	if usd.Summary.PendingAmountCurrent != 50 {
		t.Errorf("USD pending: got %v, want 50", usd.Summary.PendingAmountCurrent)
	}
}

func TestDebtHistoryMonthPrefixFormats(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"2026-10-09 00:00:00", "2026-10"},
		{"2026-10-09", "2026-10"},
		{"2026-10-09 00:00:00 +0000 UTC", "2026-10"},
		{"", ""},
		{"2026/10/09", ""},
	}
	for _, tc := range cases {
		if got := monthPrefix(tc.in); got != tc.want {
			t.Errorf("monthPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func findCurrency(t *testing.T, currencies []models.DebtHistoryCurrency, code string) models.DebtHistoryCurrency {
	t.Helper()
	for _, c := range currencies {
		if c.Code == code {
			return c
		}
	}
	t.Fatalf("moneda %s no encontrada", code)
	return models.DebtHistoryCurrency{}
}
