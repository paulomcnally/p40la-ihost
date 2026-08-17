package storage

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// newServiceNullTestDB crea una DB con servicios cuyo institution es NULL o con
// valor (escenario de la SPEC-042: servicios legacy/manuales en iHost).
func newServiceNullTestDB(t *testing.T) *ServiceStorage {
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
	mustExec("INSERT INTO institutions (name) VALUES ('Claro')")
	// institution NULL (legacy/manual) y con valor.
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Internet', NULL, 1, 'monthly', 100, 1, 'internet', 'fixed', 1)`)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring, institution_id)
		VALUES (1, 'Telefonía', 'Claro', 1, 'monthly', 80, 1, 'signal', 'fixed', 1, 1)`)

	return NewServiceStorage(database)
}

func TestServiceListHandlesNullInstitution(t *testing.T) {
	store := newServiceNullTestDB(t)
	ctx := context.Background()

	services, err := store.List(ctx, nil)
	if err != nil {
		t.Fatalf("List con institution NULL: %v", err)
	}
	if len(services) != 2 {
		t.Fatalf("se esperaban 2 servicios, got %d", len(services))
	}

	byName := map[string]models.Service{}
	for _, s := range services {
		byName[s.Name] = s
	}

	if s := byName["Internet"]; s.Institution != "" {
		t.Errorf("institution NULL debería escanearse como vacío, got %q", s.Institution)
	}
	if s := byName["Telefonía"]; s.Institution != "Claro" {
		t.Errorf("institution con valor debería escanearse correctamente, got %q", s.Institution)
	}
}

func TestServiceGetByIDHandlesNullInstitution(t *testing.T) {
	store := newServiceNullTestDB(t)
	ctx := context.Background()

	svc, err := store.GetByID(ctx, 1)
	if err != nil {
		t.Fatalf("GetByID con institution NULL: %v", err)
	}
	if svc == nil {
		t.Fatalf("GetByID no devolvió el servicio")
	}
	if svc.Institution != "" {
		t.Errorf("institution NULL debería escanearse como vacío, got %q", svc.Institution)
	}
}
