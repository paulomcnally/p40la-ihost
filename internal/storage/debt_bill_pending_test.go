package storage

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func newDebtBillPendingTestDB(t *testing.T) *DebtBillStorage {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("insertar dato de prueba (%q): %v", query, err)
		}
	}

	mustExec("INSERT INTO institutions (name) VALUES ('Banco LAFISE')")
	mustExec("INSERT INTO institutions (name) VALUES ('BAC Credomatic')")
	mustExec(`INSERT INTO debts (institution_id, identifier, description, total, principal, currency_id, installments_total, installment_amount, interest_rate, payment_day, start_date, status)
		VALUES (1, 'PREST-001', 'Préstamo LAFISE', 150, 150, 1, 3, 50, 0, 5, '2026-01-01', 'activa')`)
	mustExec(`INSERT INTO debts (institution_id, identifier, description, total, principal, currency_id, installments_total, installment_amount, interest_rate, payment_day, start_date, status)
		VALUES (2, 'TARJ-002', 'Tarjeta BAC', 500, 500, 1, 1, 500, 0, 10, '2026-01-01', 'activa')`)
	mustExec(`INSERT INTO debts (institution_id, identifier, description, total, principal, currency_id, installments_total, installment_amount, interest_rate, payment_day, start_date, status, deleted_at)
		VALUES (1, 'ELIM-003', 'Deuda eliminada', 50, 50, 1, 1, 50, 0, 15, '2026-01-01', 'activa', CURRENT_TIMESTAMP)`)

	// Deuda 1: 2 cuotas pendientes + 1 pagada.
	mustExec("INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status) VALUES (1, 1, '2026-02-05', 50, 'pending')")
	mustExec("INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status) VALUES (1, 2, '2026-03-05', 50, 'pending')")
	mustExec("INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status) VALUES (1, 3, '2026-04-05', 50, 'paid')")
	// Deuda 2: 1 cuota pendiente.
	mustExec("INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status) VALUES (2, 1, '2026-02-10', 500, 'pending')")
	// Deuda eliminada: cuota pendiente que NO debe aparecer.
	mustExec("INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status) VALUES (3, 1, '2026-02-15', 50, 'pending')")
	// Cuota soft-deleted pendiente que NO debe aparecer.
	mustExec("INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status, deleted_at) VALUES (1, 4, '2026-05-05', 50, 'pending', CURRENT_TIMESTAMP)")

	return NewDebtBillStorage(database)
}

func TestDebtBillListPendingWithDetails(t *testing.T) {
	store := newDebtBillPendingTestDB(t)
	ctx := context.Background()

	pending, err := store.ListPendingWithDetails(ctx, "2026-12-31")
	if err != nil {
		t.Fatalf("ListPendingWithDetails: %v", err)
	}

	// 3 cuotas pendientes válidas (excluye deuda eliminada y cuota borrada).
	if len(pending) != 3 {
		t.Fatalf("se esperaban 3 cuotas pendientes, got %d (%+v)", len(pending), pending)
	}

	byDebt := map[int64][]models.PendingDebtDetail{}
	for _, d := range pending {
		byDebt[d.DebtID] = append(byDebt[d.DebtID], d)
	}

	rows1, ok := byDebt[1]
	if !ok || len(rows1) != 2 {
		t.Fatalf("deuda 1: se esperaban 2 cuotas, got %d", len(rows1))
	}
	for _, d := range rows1 {
		if d.DebtDescription != "Préstamo LAFISE" {
			t.Errorf("description esperado 'Préstamo LAFISE', got %q", d.DebtDescription)
		}
		if d.InstitutionName != "Banco LAFISE" {
			t.Errorf("institución esperada 'Banco LAFISE', got %q", d.InstitutionName)
		}
		if d.CurrencySymbol != "C$" {
			t.Errorf("currency_symbol esperado 'C$', got %q", d.CurrencySymbol)
		}
		if d.CurrencyCode != "NIO" {
			t.Errorf("currency_code esperado 'NIO', got %q", d.CurrencyCode)
		}
	}

	rows2, ok := byDebt[2]
	if !ok || len(rows2) != 1 {
		t.Fatalf("deuda 2: se esperaba 1 cuota, got %d", len(rows2))
	}
	if rows2[0].Amount != 500 {
		t.Errorf("amount esperado 500, got %v", rows2[0].Amount)
	}

	if _, ok := byDebt[3]; ok {
		t.Error("cuotas de deuda eliminada no deberían aparecer")
	}
}

func TestDebtBillListPendingWithDetailsCutoff(t *testing.T) {
	store := newDebtBillPendingTestDB(t)
	ctx := context.Background()

	// Cutoff = 2026-03-31 (fin de marzo): solo cuotas con due_date <= esa fecha.
	pending, err := store.ListPendingWithDetails(ctx, "2026-03-31")
	if err != nil {
		t.Fatalf("ListPendingWithDetails: %v", err)
	}

	// Deuda 1: cuota 2026-02-05 (sí) y 2026-03-05 (sí, límite del mes).
	// Deuda 2: cuota 2026-02-10 (sí). Cuota 2026-04-05 (pagada, excluida igual).
	if len(pending) != 3 {
		t.Fatalf("se esperaban 3 cuotas con cutoff 2026-03-31, got %d (%+v)", len(pending), pending)
	}
	for _, d := range pending {
		if d.DueDate > "2026-03-31" {
			t.Errorf("cuota con due_date %s no debería aparecer con cutoff 2026-03-31", d.DueDate)
		}
	}

	// Cutoff al día anterior al vencimiento de la cuota 2026-03-05: se excluye.
	pending, err = store.ListPendingWithDetails(ctx, "2026-03-04")
	if err != nil {
		t.Fatalf("ListPendingWithDetails: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("se esperaban 2 cuotas con cutoff 2026-03-04, got %d (%+v)", len(pending), pending)
	}
	for _, d := range pending {
		if d.DueDate > "2026-03-04" {
			t.Errorf("cuota con due_date %s no debería aparecer con cutoff 2026-03-04", d.DueDate)
		}
	}

	// Cutoff anterior a todo: no hay cuotas.
	pending, err = store.ListPendingWithDetails(ctx, "2025-12-31")
	if err != nil {
		t.Fatalf("ListPendingWithDetails: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("se esperaban 0 cuotas con cutoff 2025-12-31, got %d (%+v)", len(pending), pending)
	}
}
