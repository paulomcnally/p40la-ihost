package storage

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// newHomeNullTestDB crea una DB con hogares cuyo address es NULL o con valor
// (escenario de la SPEC-042: hogares legacy/manuales en iHost).
func newHomeNullTestDB(t *testing.T) *HomeStorage {
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
	mustExec("INSERT INTO homes (name, address) VALUES ('Casa Playa', 'Managua, Nicaragua')")

	return NewHomeStorage(database)
}

func TestHomeListHandlesNullAddress(t *testing.T) {
	store := newHomeNullTestDB(t)
	ctx := context.Background()

	homes, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List con address NULL: %v", err)
	}
	if len(homes) != 2 {
		t.Fatalf("se esperaban 2 hogares, got %d", len(homes))
	}

	byID := map[int64]models.Home{}
	for _, h := range homes {
		byID[h.ID] = h
	}

	if h := byID[1]; h.Address != "" {
		t.Errorf("address NULL debería escanearse como vacío, got %q", h.Address)
	}
	if h := byID[2]; h.Address != "Managua, Nicaragua" {
		t.Errorf("address con valor debería escanearse correctamente, got %q", h.Address)
	}
}

func TestHomeGetByIDHandlesNullAddress(t *testing.T) {
	store := newHomeNullTestDB(t)
	ctx := context.Background()

	home, err := store.GetByID(ctx, 1)
	if err != nil {
		t.Fatalf("GetByID con address NULL: %v", err)
	}
	if home == nil {
		t.Fatalf("GetByID no devolvió el hogar")
	}
	if home.Address != "" {
		t.Errorf("address NULL debería escanearse como vacío, got %q", home.Address)
	}
}
