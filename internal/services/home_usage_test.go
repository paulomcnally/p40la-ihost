package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type fakeRentalGuard struct {
	active bool
	calls  int
}

func (f *fakeRentalGuard) HasActiveRental(ctx context.Context, homeID int64) (bool, error) {
	f.calls++
	return f.active, nil
}

func TestHomeUsageLifecycle(t *testing.T) {
	database, err := db.OpenDB(filepath.Join(t.TempDir(), "home-usage.db"), "../../migrations")
	if err != nil {
		t.Fatalf("abrir db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	svc := NewHomeService(storage.NewHomeStorage(database))
	guard := &fakeRentalGuard{}
	svc.SetRentalGuard(guard)
	ctx := context.Background()

	home, err := svc.Create(ctx, "Casa Demo", "Calle Demo 123")
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	if home.Usage != models.HomeUsageRental {
		t.Fatalf("uso inicial = %s, esperaba rental", home.Usage)
	}

	if _, err := svc.Create(ctx, "Casa Inválida", "", "deposito"); !errors.Is(err, ErrInvalidHomeUsage) {
		t.Fatalf("uso inválido: error = %v", err)
	}

	// Editar nombre sin indicar uso conserva la clasificación.
	updated, err := svc.Update(ctx, home.ID, "Casa Demo Renombrada", "Calle Demo 456")
	if err != nil {
		t.Fatalf("actualizar: %v", err)
	}
	if updated.Usage != models.HomeUsageRental {
		t.Fatalf("el uso se perdió al renombrar: %s", updated.Usage)
	}

	// Con alquiler activo no se puede pasar a Habitada.
	guard.active = true
	if _, err := svc.Update(ctx, home.ID, home.Name, home.Address, models.HomeUsageInhabited); !errors.Is(err, ErrHomeHasActiveRental) {
		t.Fatalf("cambio bloqueado: error = %v", err)
	}
	current, err := svc.GetByID(ctx, home.ID)
	if err != nil {
		t.Fatalf("leer: %v", err)
	}
	if current.Usage != models.HomeUsageRental {
		t.Fatalf("el uso cambió pese al bloqueo: %s", current.Usage)
	}

	// Sin alquiler activo se permite.
	guard.active = false
	if _, err := svc.Update(ctx, home.ID, home.Name, home.Address, models.HomeUsageInhabited); err != nil {
		t.Fatalf("cambiar a habitada: %v", err)
	}

	// Volver a «En alquiler» siempre se permite.
	if _, err := svc.Update(ctx, home.ID, home.Name, home.Address, models.HomeUsageRental); err != nil {
		t.Fatalf("volver a rental: %v", err)
	}
}
