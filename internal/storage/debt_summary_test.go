package storage

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
)

// SPEC-104: el listado de deudas incluye el resumen de cuotas en una sola consulta.
func TestDebtListSummary(t *testing.T) {
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
	mustExec(`INSERT INTO debts (institution_id, identifier, description, total, principal, currency_id,
		installments_total, installment_amount, interest_rate, payment_day, start_date, status)
		VALUES (1, 'D-1', 'Deuda con pendientes', 150, 150, 1, 3, 50, 0, 5, '2026-01-01', 'activa')`)
	mustExec(`INSERT INTO debts (institution_id, identifier, description, total, principal, currency_id,
		installments_total, installment_amount, interest_rate, payment_day, start_date, status)
		VALUES (1, 'D-2', 'Deuda cerrada', 500, 500, 1, 1, 500, 0, 10, '2026-01-01', 'finalizada')`)

	// Deuda 1: 2 pagadas + 1 pendiente.
	mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status, paid_at)
		VALUES (1, 1, '2026-09-05', 50, 'paid', '2026-09-04 00:00:00 +0000 UTC')`)
	mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status, paid_at)
		VALUES (1, 2, '2026-10-05', 50, 'paid', '2026-10-09 00:00:00')`)
	mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status)
		VALUES (1, 3, '2026-11-05', 50, 'pending')`)
	// Cuota soft-deleted: no debe contar.
	mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status, deleted_at)
		VALUES (1, 4, '2026-12-05', 99, 'pending', CURRENT_TIMESTAMP)`)
	// Deuda 2: todo pagado.
	mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status, paid_at)
		VALUES (2, 1, '2026-06-10', 500, 'paid', '2026-06-09 00:00:00')`)

	store := NewDebtStorage(database)

	debts, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(debts) != 2 {
		t.Fatalf("se esperaban 2 deudas, got %d", len(debts))
	}

	d1 := debts[0]
	d2 := debts[1]
	if d1.ID != 1 {
		d1, d2 = d2, d1
	}
	if d1.PendingCount != 1 || d1.PendingAmount != 50 {
		t.Errorf("deuda 1 pendientes: got %d/%v, want 1/50", d1.PendingCount, d1.PendingAmount)
	}
	if d1.PaidCount != 2 || d1.PaidAmount != 100 {
		t.Errorf("deuda 1 pagadas: got %d/%v, want 2/100", d1.PaidCount, d1.PaidAmount)
	}
	if d1.NextDueDate != "2026-11-05" || d1.LastDueDate != "2026-11-05" {
		t.Errorf("deuda 1 vencimientos: got %q/%q, want 2026-11-05/2026-11-05", d1.NextDueDate, d1.LastDueDate)
	}
	if d1.LastPaidAt != "2026-10-09 00:00:00" {
		t.Errorf("deuda 1 último pago: got %q", d1.LastPaidAt)
	}

	if d2.PendingCount != 0 || d2.PendingAmount != 0 {
		t.Errorf("deuda 2 pendientes: got %d/%v, want 0/0", d2.PendingCount, d2.PendingAmount)
	}
	if d2.PaidCount != 1 || d2.PaidAmount != 500 {
		t.Errorf("deuda 2 pagadas: got %d/%v, want 1/500", d2.PaidCount, d2.PaidAmount)
	}
	if d2.NextDueDate != "" || d2.LastDueDate != "" {
		t.Errorf("deuda 2 sin pendientes no debería tener fechas: %q/%q", d2.NextDueDate, d2.LastDueDate)
	}

	got, err := store.GetByID(ctx, d1.ID)
	if err != nil || got == nil {
		t.Fatalf("GetByID: %v, %v", got, err)
	}
	if got.PendingAmount != 50 || got.PaidCount != 2 {
		t.Errorf("GetByID resumen: got pending %v / paid %d", got.PendingAmount, got.PaidCount)
	}
}
