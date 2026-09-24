package storage

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func strPtr(s string) *string { return &s }

func setupCycleTest(t *testing.T) (*ServiceCycleStorage, *BillStorage, *ServiceStorage, int64) {
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
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring, start_date, end_date)
		VALUES (1, 'Seguro', 'ASSA', 2, 'monthly', 100, 1, 'insurance', 'fixed', 1, '2025-10-20', '2026-10-19')`)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Internet', 'Claro', 2, 'monthly', 50, 1, 'internet', 'fixed', 0)`)

	return NewServiceCycleStorage(database), NewBillStorage(database), NewServiceStorage(database), 1
}

func TestCycleCreateAndNextSequence(t *testing.T) {
	store, _, _, svcID := setupCycleTest(t)
	ctx := context.Background()

	c1, err := store.Create(ctx, &models.ServiceCycle{ServiceID: svcID, Sequence: 1, StartDate: strPtr("2025-10-20"), EndDate: strPtr("2026-10-19")})
	if err != nil {
		t.Fatalf("crear ciclo 1: %v", err)
	}
	if c1.ID == 0 {
		t.Error("el ciclo debería tener id")
	}

	seq, err := store.NextSequence(ctx, svcID)
	if err != nil {
		t.Fatalf("next sequence: %v", err)
	}
	if seq != 2 {
		t.Errorf("se esperaba sequence 2, got %d", seq)
	}

	// Sin ciclos → sequence 1
	emptySeq, err := store.NextSequence(ctx, 999)
	if err != nil {
		t.Fatalf("next sequence vacío: %v", err)
	}
	if emptySeq != 1 {
		t.Errorf("sin ciclos se esperaba 1, got %d", emptySeq)
	}
}

func TestFindForPeriod(t *testing.T) {
	store, _, _, svcID := setupCycleTest(t)
	ctx := context.Background()

	c1, _ := store.Create(ctx, &models.ServiceCycle{ServiceID: svcID, Sequence: 1, StartDate: strPtr("2025-10-20"), EndDate: strPtr("2026-10-19")})
	c2, _ := store.Create(ctx, &models.ServiceCycle{ServiceID: svcID, Sequence: 2, StartDate: strPtr("2026-10-20"), EndDate: strPtr("2027-10-19")})

	// Marzo 2026 → ciclo 1
	got, err := store.FindForPeriod(ctx, svcID, 2026, 3)
	if err != nil || got == nil {
		t.Fatalf("FindForPeriod 2026/3: %v (cycle=%v)", err, got)
	}
	if got.ID != c1.ID {
		t.Errorf("2026/3 debería caer en ciclo 1, got ciclo %d", got.Sequence)
	}

	// Noviembre 2026 → ciclo 2
	got, err = store.FindForPeriod(ctx, svcID, 2026, 11)
	if err != nil || got == nil {
		t.Fatalf("FindForPeriod 2026/11: %v (cycle=%v)", err, got)
	}
	if got.ID != c2.ID {
		t.Errorf("2026/11 debería caer en ciclo 2, got ciclo %d", got.Sequence)
	}

	// Período fuera de todo rango → fallback al último ciclo (c2)
	got, err = store.FindForPeriod(ctx, svcID, 2024, 1)
	if err != nil || got == nil {
		t.Fatalf("FindForPeriod 2024/1: %v (cycle=%v)", err, got)
	}
	if got.ID != c2.ID {
		t.Errorf("2024/1 debería caer al último ciclo, got ciclo %d", got.Sequence)
	}

	// Servicio sin ciclos → nil
	got, err = store.FindForPeriod(ctx, 999, 2026, 3)
	if err != nil || got != nil {
		t.Fatalf("servicio sin ciclos debería ser nil (err=%v)", err)
	}
}

func TestBillCreateTagsCycle(t *testing.T) {
	store, billStore, _, svcID := setupCycleTest(t)
	ctx := context.Background()

	cycle, _ := store.Create(ctx, &models.ServiceCycle{ServiceID: svcID, Sequence: 1, StartDate: strPtr("2025-10-20"), EndDate: strPtr("2026-10-19")})

	bill, err := billStore.Create(ctx, &models.Bill{ServiceID: svcID, Year: 2026, Month: 3, Amount: 67.39, Status: "pending"})
	if err != nil {
		t.Fatalf("crear factura: %v", err)
	}
	if bill.CycleID == nil || *bill.CycleID != cycle.ID {
		t.Errorf("la factura debería quedar etiquetada con el ciclo %d, got %v", cycle.ID, bill.CycleID)
	}

	// Período fuera del rango → fallback al último ciclo.
	bill2, err := billStore.Create(ctx, &models.Bill{ServiceID: svcID, Year: 2024, Month: 1, Amount: 10, Status: "pending"})
	if err != nil {
		t.Fatalf("crear factura fuera de rango: %v", err)
	}
	if bill2.CycleID == nil || *bill2.CycleID != cycle.ID {
		t.Errorf("la factura fuera de rango debería caer al último ciclo, got %v", bill2.CycleID)
	}

	// Servicio sin ciclos → cycle_id NULL.
	bill3, err := billStore.Create(ctx, &models.Bill{ServiceID: 2, Year: 2026, Month: 3, Amount: 10, Status: "pending"})
	if err != nil {
		t.Fatalf("crear factura sin ciclos: %v", err)
	}
	if bill3.CycleID != nil {
		t.Errorf("servicio sin ciclos debería tener cycle_id nil, got %v", bill3.CycleID)
	}
}

func TestBackfillExistingCycles(t *testing.T) {
	store, billStore, _, svcID := setupCycleTest(t)
	ctx := context.Background()

	// El servicio con vigencia aún no tiene ciclo (la migración corrió sobre DB
	// vacía): el backfill manual debe crear el ciclo 1 y asociar sus facturas.
	bill, err := billStore.Create(ctx, &models.Bill{ServiceID: svcID, Year: 2026, Month: 2, Amount: 50, Status: "paid"})
	if err != nil {
		t.Fatalf("crear factura: %v", err)
	}
	if bill.CycleID != nil {
		t.Fatal("sin backfill, la factura no debería tener ciclo aún")
	}

	if err := store.BackfillExistingCycles(ctx); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	cycles, err := store.ListByService(ctx, svcID)
	if err != nil {
		t.Fatalf("listar ciclos: %v", err)
	}
	if len(cycles) != 1 || cycles[0].Sequence != 1 {
		t.Fatalf("el backfill debería crear ciclo 1, got %+v", cycles)
	}

	// La factura existente quedó asociada al ciclo 1.
	bill2, err := billStore.GetByID(ctx, bill.ID)
	if err != nil || bill2 == nil {
		t.Fatalf("obtener factura: %v", err)
	}
	if bill2.CycleID == nil || *bill2.CycleID != cycles[0].ID {
		t.Errorf("la factura debería quedar asociada al ciclo %d, got %v", cycles[0].ID, bill2.CycleID)
	}

	// Re-ejecución idempotente (no duplica ciclos).
	if err := store.BackfillExistingCycles(ctx); err != nil {
		t.Fatalf("backfill idempotente: %v", err)
	}
	cycles, err = store.ListByService(ctx, svcID)
	if err != nil {
		t.Fatalf("listar ciclos tras backfill: %v", err)
	}
	if len(cycles) != 1 {
		t.Errorf("el backfill no debería duplicar ciclos, got %d", len(cycles))
	}
}

func TestIsInsuranceField(t *testing.T) {
	_, _, svcStore, _ := setupCycleTest(t)
	ctx := context.Background()

	// El servicio de prueba no tiene institución → is_insurance false.
	svc, err := svcStore.GetByID(ctx, 1)
	if err != nil || svc == nil {
		t.Fatalf("obtener servicio: %v", err)
	}
	if svc.IsInsurance {
		t.Error("servicio sin institución no debería ser insurance")
	}

	// Crear institución de categoría insurance y asociarla.
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	mustExec := func(q string, a ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, q, a...); err != nil {
			t.Fatalf("insertar (%q): %v", q, err)
		}
	}
	mustExec("INSERT INTO homes (name) VALUES ('Casa')")
	mustExec("INSERT INTO institutions (name, category_id) VALUES ('ASSA', 1)")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring, institution_id)
		VALUES (1, 'Seguro Autos', 'ASSA', 2, 'monthly', 67.39, 1, 'insurance', 'fixed', 1, 1)`)

	svc2, err := NewServiceStorage(database).GetByID(ctx, 1)
	if err != nil || svc2 == nil {
		t.Fatalf("obtener servicio con institución: %v", err)
	}
	if !svc2.IsInsurance {
		t.Error("servicio con institución de categoría insurance debería ser is_insurance=true")
	}
}