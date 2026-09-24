package storage

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
)

func setupAutoServiceByService(t *testing.T) (*AutoServiceStorage, int64) {
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
		VALUES (1, 'Seguro Autos', 'ASSA', 2, 'monthly', 67.39, 1, 'insurance', 'fixed', 1)`)
	mustExec("INSERT INTO autos (year, model, brand, color, placa) VALUES (2026, 'Hilux', 'Toyota', 'Blanco', 'P123ABC')")
	mustExec("INSERT INTO autos (year, model, brand, color, placa) VALUES (2024, 'N400', 'Chevrolet', 'Rojo', 'P456DEF')")
	mustExec(`INSERT INTO auto_services (auto_id, service_id, coverage_type, policy_number, certificate, insurer_number)
		VALUES (1, 1, 'full_cover', '02B 128265', '1', '1800 9911')`)
	mustExec(`INSERT INTO auto_services (auto_id, service_id, coverage_type, policy_number, certificate, insurer_number)
		VALUES (2, 1, 'daños_a_terceros', '02B 103715', NULL, '1800')`)

	return NewAutoServiceStorage(database), 1
}

func TestListByService(t *testing.T) {
	store, serviceID := setupAutoServiceByService(t)
	ctx := context.Background()

	rows, err := store.ListByService(ctx, serviceID)
	if err != nil {
		t.Fatalf("ListByService: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("se esperaban 2 autos, got %d", len(rows))
	}

	var hilux, n400 bool
	for _, r := range rows {
		switch r.Brand {
		case "Toyota":
			hilux = true
			if r.Placa != "P123ABC" {
				t.Errorf("placa Hilux: %q", r.Placa)
			}
			if r.PolicyNumber != "02B 128265" {
				t.Errorf("policy Hilux: %q", r.PolicyNumber)
			}
			if r.Certificate == nil || *r.Certificate != "1" {
				t.Errorf("certificate Hilux debería ser '1', got %v", r.Certificate)
			}
			if r.CoverageType != "full_cover" {
				t.Errorf("coverage Hilux: %q", r.CoverageType)
			}
		case "Chevrolet":
			n400 = true
			if r.CoverageType != "daños_a_terceros" {
				t.Errorf("coverage N400: %q", r.CoverageType)
			}
			if r.Certificate != nil {
				t.Errorf("certificate N400 debería ser nil, got %v", *r.Certificate)
			}
			if r.InsurerNumber != "1800" {
				t.Errorf("insurer N400: %q", r.InsurerNumber)
			}
		}
	}
	if !hilux || !n400 {
		t.Errorf("faltaron autos: hilux=%v n400=%v", hilux, n400)
	}
}

func TestListByServiceEmpty(t *testing.T) {
	store, _ := setupAutoServiceByService(t)
	ctx := context.Background()

	rows, err := store.ListByService(ctx, 999)
	if err != nil {
		t.Fatalf("ListByService sin asociaciones: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("se esperaba lista vacía, got %d", len(rows))
	}
}
