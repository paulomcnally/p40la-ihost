package storage

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
)

func setupBillPeriod(t *testing.T) (*BillStorage, int64) {
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

	mustExec("INSERT INTO homes (name) VALUES ('Casa')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Internet', 'Claro', 1, 'monthly', 100, 1, 'internet', 'fixed', 1)`)
	mustExec("INSERT INTO bills (service_id, year, month, amount, status) VALUES (1, 2026, 8, 1500, 'pending')")

	return NewBillStorage(database), 1
}

func TestFindByServicePeriodIncludingDeleted(t *testing.T) {
	store, _ := setupBillPeriod(t)
	ctx := context.Background()

	// Sin soft-delete: el período se encuentra por ambas queries.
	active, err := store.FindByServicePeriod(ctx, 1, 2026, 8)
	if err != nil || active == nil {
		t.Fatalf("FindByServicePeriod activo: %v (bill=%v)", err, active)
	}
	including, err := store.FindByServicePeriodIncludingDeleted(ctx, 1, 2026, 8)
	if err != nil || including == nil {
		t.Fatalf("FindByServicePeriodIncludingDeleted activo: %v (bill=%v)", err, including)
	}
	if including.ID != active.ID {
		t.Errorf("IDs deberían coincidir: %d vs %d", including.ID, active.ID)
	}

	// Soft-delete: la query normal ya no lo ve, la including sí.
	if err := store.SoftDelete(ctx, active.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	hidden, err := store.FindByServicePeriod(ctx, 1, 2026, 8)
	if err != nil {
		t.Fatalf("FindByServicePeriod tras delete: %v", err)
	}
	if hidden != nil {
		t.Errorf("FindByServicePeriod no debería devolver la fila soft-deleted, got %+v", hidden)
	}
	found, err := store.FindByServicePeriodIncludingDeleted(ctx, 1, 2026, 8)
	if err != nil {
		t.Fatalf("FindByServicePeriodIncludingDeleted tras delete: %v", err)
	}
	if found == nil {
		t.Fatal("FindByServicePeriodIncludingDeleted debería devolver la fila soft-deleted")
	}
	if found.DeletedAt == nil {
		t.Error("la fila recuperada debería tener deleted_at seteado")
	}

	// Período inexistente → nil en ambas.
	none, err := store.FindByServicePeriodIncludingDeleted(ctx, 1, 2025, 1)
	if err != nil || none != nil {
		t.Fatalf("período inexistente debería ser nil (err=%v)", err)
	}
}

func TestReactivateBill(t *testing.T) {
	store, _ := setupBillPeriod(t)
	ctx := context.Background()

	bill, err := store.FindByServicePeriodIncludingDeleted(ctx, 1, 2026, 8)
	if err != nil || bill == nil {
		t.Fatalf("buscar bill: %v", err)
	}
	if err := store.SoftDelete(ctx, bill.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	if err := store.Reactivate(ctx, bill.ID); err != nil {
		t.Fatalf("Reactivate: %v", err)
	}

	active, err := store.FindByServicePeriod(ctx, 1, 2026, 8)
	if err != nil {
		t.Fatalf("FindByServicePeriod tras reactivar: %v", err)
	}
	if active == nil {
		t.Fatal("tras reactivar, la factura debería ser visible en FindByServicePeriod")
	}
	if active.DeletedAt != nil {
		t.Error("deleted_at debería ser NULL tras reactivar")
	}
	if active.ID != bill.ID {
		t.Errorf("se debería conservar el id original, got %d", active.ID)
	}
}

func TestReactivateNonDeletedIsNoop(t *testing.T) {
	store, _ := setupBillPeriod(t)
	ctx := context.Background()

	bill, err := store.FindByServicePeriod(ctx, 1, 2026, 8)
	if err != nil || bill == nil {
		t.Fatalf("buscar bill: %v", err)
	}
	// Reactivar una fila que no está borrada no debe romper nada.
	if err := store.Reactivate(ctx, bill.ID); err != nil {
		t.Fatalf("Reactivate sobre fila activa: %v", err)
	}
}
