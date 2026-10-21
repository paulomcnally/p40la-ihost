package storage

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
)

// SPEC-104: finalización automática de deudas completas y limpieza de cuotas sobrantes.
func TestFinalizeCompletedAndOverflow(t *testing.T) {
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
	insertDebt := func(id int, status string, deleted bool) {
		t.Helper()
		del := "NULL"
		if deleted {
			del = "CURRENT_TIMESTAMP"
		}
		mustExec(`INSERT INTO debts (id, institution_id, identifier, description, total, principal, currency_id,
			installments_total, installment_amount, interest_rate, payment_day, start_date, status, deleted_at)
			VALUES (?, 1, ?, ?, 100, 100, 1, 2, 50, 0, 5, '2026-01-01', ?, `+del+`)`,
			id, "ID-"+status, "Deuda "+status, status)
	}

	insertDebt(1, "activa", false)   // con pendiente -> NO se finaliza
	insertDebt(2, "activa", false)   // todo pagado -> se finaliza
	insertDebt(3, "inactiva", false) // pausada -> NO se toca
	insertDebt(4, "activa", false)   // sin cuotas -> NO se finaliza
	insertDebt(5, "activa", true)    // eliminada -> NO se toca

	mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status)
		VALUES (1, 1, '2026-02-05', 50, 'paid')`)
	mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status)
		VALUES (1, 2, '2026-03-05', 50, 'pending')`)
	mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status)
		VALUES (2, 1, '2026-02-05', 50, 'paid')`)
	mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status)
		VALUES (2, 2, '2026-03-05', 50, 'paid')`)
	mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status)
		VALUES (3, 1, '2026-02-05', 50, 'paid')`)
	mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status)
		VALUES (3, 2, '2026-03-05', 50, 'paid')`)
	mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status)
		VALUES (5, 1, '2026-02-05', 50, 'paid')`)

	bills := NewDebtBillStorage(database)

	affected, err := bills.FinalizeCompleted(ctx)
	if err != nil {
		t.Fatalf("FinalizeCompleted: %v", err)
	}
	if affected != 1 {
		t.Errorf("se esperaba finalizar 1 deuda, got %d", affected)
	}

	assertStatus := func(id int64, want string) {
		t.Helper()
		var got string
		if err := database.QueryRowContext(ctx, "SELECT status FROM debts WHERE id = ?", id).Scan(&got); err != nil {
			t.Fatalf("leer estado de deuda %d: %v", id, err)
		}
		if got != want {
			t.Errorf("deuda %d: got %q, want %q", id, got, want)
		}
	}
	assertStatus(1, "activa")
	assertStatus(2, "finalizada")
	assertStatus(3, "inactiva")
	assertStatus(4, "activa")
	assertStatus(5, "activa")

	// Idempotente: una segunda pasada no cambia nada.
	if again, err := bills.FinalizeCompleted(ctx); err != nil || again != 0 {
		t.Errorf("segunda pasada: got %d, err %v", again, err)
	}

	// FinalizeIfCompleted sobre una deuda puntual.
	if ok, err := bills.FinalizeIfCompleted(ctx, 1); err != nil || ok {
		t.Errorf("deuda 1 no debía finalizarse: ok=%v err=%v", ok, err)
	}
	mustExec(`UPDATE debt_bills SET status = 'paid', paid_at = '2026-03-05'
		WHERE debt_id = 1 AND installment_number = 2`)
	if ok, err := bills.FinalizeIfCompleted(ctx, 1); err != nil || !ok {
		t.Errorf("deuda 1 debía finalizarse: ok=%v err=%v", ok, err)
	}
	assertStatus(1, "finalizada")

	// Soft delete de cuotas sobrantes (REQ-009b).
	insertDebt(6, "activa", false)
	for k := 1; k <= 4; k++ {
		mustExec(`INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status)
			VALUES (6, ?, '2026-0' || ? || '-05', 25, 'pending')`, k, k+1)
	}
	if err := bills.SoftDeleteOverflow(ctx, 6, 2); err != nil {
		t.Fatalf("SoftDeleteOverflow: %v", err)
	}
	var alive int
	if err := database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM debt_bills WHERE debt_id = 6 AND deleted_at IS NULL").Scan(&alive); err != nil {
		t.Fatalf("contar cuotas vivas: %v", err)
	}
	if alive != 2 {
		t.Errorf("se esperaban 2 cuotas vivas, got %d", alive)
	}
}
