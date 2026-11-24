package storage

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
)

func TestListAvailableServicesScansServiceColumns(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("ejecutar %q: %v", query, err)
		}
	}

	exec("INSERT INTO homes (name) VALUES ('Hogar Demo')")
	exec(`INSERT INTO institutions (name, category_id)
		VALUES ('Aseguradora Demo', (SELECT id FROM institution_categories WHERE key = 'insurance'))`)
	exec(`INSERT INTO institutions (name, category_id)
		VALUES ('Taller Demo', (SELECT id FROM institution_categories WHERE key = 'workshop'))`)
	exec(`INSERT INTO services (home_id, name, institution_id, currency_id, frequency, suggested_amount, active, icon_key, billing_type)
		VALUES (1, 'Seguro Demo', 1, 1, 'yearly', 1000, 1, 'shield', 'fixed')`)
	exec(`INSERT INTO services (home_id, name, institution_id, currency_id, frequency, suggested_amount, active, icon_key, billing_type)
		VALUES (1, 'Seguro Asociado Demo', 1, 1, 'yearly', 1000, 1, 'shield', 'fixed')`)
	exec(`INSERT INTO services (home_id, name, institution_id, currency_id, frequency, suggested_amount, active, icon_key, billing_type)
		VALUES (1, 'Taller Demo Servicio', 2, 1, 'monthly', 500, 1, 'wrench', 'fixed')`)
	exec("INSERT INTO autos (year, model, brand, color, placa) VALUES (2024, 'Modelo Demo', 'Marca Demo', 'white', 'DEMO-001')")
	exec(`INSERT INTO auto_services (auto_id, service_id, coverage_type, policy_number, insurer_number)
		VALUES (1, 2, 'full_cover', 'POL-DEMO-0001', 'ASEG-DEMO-001')`)

	store := NewAutoServiceStorage(database)
	services, err := store.ListAvailableServices(ctx, 1)
	if err != nil {
		t.Fatalf("ListAvailableServices: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("se esperaba 1 servicio disponible, got %d", len(services))
	}
	service := services[0]
	if service.ID != 1 || service.Name != "Seguro Demo" {
		t.Fatalf("servicio inesperado: %+v", service)
	}
	if !service.IsInsurance {
		t.Error("is_insurance debería ser true para el servicio de la categoría insurance")
	}
	if service.AutomationAccountID != nil {
		t.Errorf("automation_account_id debería ser nil, got %v", *service.AutomationAccountID)
	}
}
