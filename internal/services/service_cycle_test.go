package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func setupServiceCycleTest(t *testing.T) (*ServiceCycleService, *storage.ServiceStorage, int64, int64) {
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
	mustExec("INSERT INTO institutions (name, category_id) VALUES ('ASSA', 1)")     // categoría insurance (seed id 1)
	mustExec("INSERT INTO institutions (name, category_id) VALUES ('Claro', 2)")    // categoría telecomunicaciones
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring, institution_id, start_date, end_date, webhook_uuid)
		VALUES (1, 'Seguro Hilux', 'ASSA', 2, 'monthly', 67.39, 1, 'insurance', 'fixed', 1, 1, '2025-10-20', '2026-10-19', 'wbh-0001')`)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring, institution_id)
		VALUES (1, 'Internet', 'Claro', 2, 'monthly', 50, 1, 'internet', 'fixed', 0, 2)`)

	svcStorage := storage.NewServiceStorage(database)
	cycleStorage := storage.NewServiceCycleStorage(database)
	return NewServiceCycleService(svcStorage, cycleStorage), svcStorage, 1, 2
}

func TestRenewService(t *testing.T) {
	svc, _, insuranceID, _ := setupServiceCycleTest(t)
	ctx := context.Background()

	amount := 70.00
	updated, cycle, err := svc.Renew(ctx, insuranceID, "2026-10-20", "2027-10-19", &amount)
	if err != nil {
		t.Fatalf("Renew: %v", err)
	}
	if updated == nil || cycle == nil {
		t.Fatal("Renew debería devolver servicio y ciclo")
	}
	if cycle.Sequence != 1 {
		t.Errorf("primer ciclo sequence 1, got %d", cycle.Sequence)
	}
	if updated.StartDate == nil || *updated.StartDate != "2026-10-20" {
		t.Errorf("start_date actualizado: %v", updated.StartDate)
	}
	if updated.EndDate == nil || *updated.EndDate != "2027-10-19" {
		t.Errorf("end_date actualizado: %v", updated.EndDate)
	}
	if updated.SuggestedAmount != 70.00 {
		t.Errorf("suggested_amount debería ser 70, got %v", updated.SuggestedAmount)
	}

	// Segunda renovación → sequence 2; monto sin cambio conserva el actual.
	updated2, cycle2, err := svc.Renew(ctx, insuranceID, "2027-10-20", "2028-10-19", nil)
	if err != nil {
		t.Fatalf("Renew 2: %v", err)
	}
	if cycle2.Sequence != 2 {
		t.Errorf("segundo ciclo sequence 2, got %d", cycle2.Sequence)
	}
	if updated2.SuggestedAmount != 70.00 {
		t.Errorf("sin suggested_amount debería conservar 70, got %v", updated2.SuggestedAmount)
	}
}

func TestRenewAppliesToAllServices(t *testing.T) {
	svc, _, _, nonInsuranceID := setupServiceCycleTest(t)
	ctx := context.Background()

	// La renovación aplica a todos los servicios, no solo a los de seguro.
	updated, cycle, err := svc.Renew(ctx, nonInsuranceID, "2026-01-01", "2027-01-01", nil)
	if err != nil {
		t.Fatalf("Renew de servicio no-seguro: %v", err)
	}
	if updated == nil || cycle == nil {
		t.Fatal("Renew debería devolver servicio y ciclo")
	}
	if cycle.Sequence != 1 {
		t.Errorf("primer ciclo sequence 1, got %d", cycle.Sequence)
	}
}

func TestRenewInvalidDates(t *testing.T) {
	svc, _, insuranceID, _ := setupServiceCycleTest(t)
	ctx := context.Background()

	cases := []struct {
		name    string
		start   string
		end     string
		wantErr string
	}{
		{"start >= end", "2027-10-19", "2026-10-20", "start_date debe ser anterior a end_date"},
		{"fechas mal formadas", "2026/10/20", "2027/10/19", "fechas válidas"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := svc.Renew(ctx, insuranceID, tc.start, tc.end, nil)
			if err == nil {
				t.Fatalf("se esperaba error: %s", tc.wantErr)
			}
		})
	}
}

func TestRenewPreservesWebhook(t *testing.T) {
	svc, svcStorage, insuranceID, _ := setupServiceCycleTest(t)
	ctx := context.Background()

	before, err := svcStorage.GetByID(ctx, insuranceID)
	if err != nil || before == nil {
		t.Fatalf("obtener servicio: %v", err)
	}
	if before.WebhookUUID == "" {
		t.Fatal("el servicio debería tener webhook_uuid")
	}

	if _, _, err := svc.Renew(ctx, insuranceID, "2026-10-20", "2027-10-19", nil); err != nil {
		t.Fatalf("Renew: %v", err)
	}

	after, err := svcStorage.GetByID(ctx, insuranceID)
	if err != nil || after == nil {
		t.Fatalf("obtener servicio renovado: %v", err)
	}
	if after.WebhookUUID != before.WebhookUUID {
		t.Errorf("el webhook_uuid no debería cambiar en la renovación: %q → %q", before.WebhookUUID, after.WebhookUUID)
	}
}