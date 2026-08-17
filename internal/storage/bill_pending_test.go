package storage

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func newBillSummaryTestDB(t *testing.T) *BillStorage {
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

	mustExec("INSERT INTO homes (name) VALUES ('Casa Central')")
	mustExec("INSERT INTO homes (name) VALUES ('Casa Playa')")
	mustExec("INSERT INTO institutions (name) VALUES ('Claro')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, institution_id, is_recurring)
		VALUES (1, 'Internet', 'Claro', 1, 'monthly', 100, 1, 'internet', 'fixed', 1, 1)`)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (2, 'Luz', '', 1, 'monthly', 80, 1, 'electricity', 'variable', 1)`)

	// Facturas: 2 pendientes + 1 pagada + 1 de servicio eliminado.
	mustExec("INSERT INTO bills (service_id, year, month, amount, status) VALUES (1, 2026, 8, 1500, 'pending')")
	mustExec("INSERT INTO bills (service_id, year, month, amount, status) VALUES (2, 2026, 1, 100, 'pending')")
	mustExec("INSERT INTO bills (service_id, year, month, amount, status) VALUES (1, 2026, 7, 1490, 'paid')")
	mustExec("INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring, deleted_at) VALUES (1, 'Eliminado', '', 1, 'monthly', 50, 1, 'other', 'fixed', 1, CURRENT_TIMESTAMP)")
	mustExec("INSERT INTO bills (service_id, year, month, amount, status) VALUES (3, 2026, 8, 50, 'pending')")

	return NewBillStorage(database)
}

func TestListPendingWithDetails(t *testing.T) {
	store := newBillSummaryTestDB(t)
	ctx := context.Background()

	pending, err := store.ListPendingWithDetails(ctx)
	if err != nil {
		t.Fatalf("ListPendingWithDetails: %v", err)
	}

	// 2 pendientes (la del servicio eliminado no debe aparecer).
	if len(pending) != 2 {
		t.Fatalf("se esperaban 2 facturas pendientes, got %d (%+v)", len(pending), pending)
	}

	byService := map[int64]models.PendingBillDetail{}
	for _, d := range pending {
		byService[d.ServiceID] = d
	}

	svc1, ok := byService[1]
	if !ok {
		t.Fatalf("factura de servicio 1 no encontrada")
	}
	if svc1.HomeName != "Casa Central" {
		t.Errorf("home_name esperado 'Casa Central', got %q", svc1.HomeName)
	}
	if svc1.Institution != "Claro" {
		t.Errorf("institución esperada 'Claro', got %q", svc1.Institution)
	}
	if svc1.ServiceName != "Internet" {
		t.Errorf("service_name esperado 'Internet', got %q", svc1.ServiceName)
	}
	if svc1.CurrencySymbol != "C$" {
		t.Errorf("currency_symbol esperado 'C$', got %q", svc1.CurrencySymbol)
	}
	if svc1.Status != "pending" {
		t.Errorf("status esperado pending, got %q", svc1.Status)
	}

	svc2, ok := byService[2]
	if !ok {
		t.Fatalf("factura de servicio 2 no encontrada")
	}
	// Sin institución (institution vacía y sin institution_id) → vacío.
	if svc2.Institution != "" {
		t.Errorf("institución esperada vacía, got %q", svc2.Institution)
	}
	if svc2.HomeName != "Casa Playa" {
		t.Errorf("home_name esperado 'Casa Playa', got %q", svc2.HomeName)
	}
}