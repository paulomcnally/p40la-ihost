package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func TestHomeReorderPersistsOrder(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	store := NewHomeStorage(database)
	ctx := context.Background()

	casaA := mustCreateHome(t, store, ctx, "Casa A")
	casaB := mustCreateHome(t, store, ctx, "Casa B")
	casaC := mustCreateHome(t, store, ctx, "Casa C")

	// Orden inicial: por sort_order (insert secuencial).
	homes, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List falló: %v", err)
	}
	if got := homeNames(homes); strings.Join(got, ",") != "Casa A,Casa B,Casa C" {
		t.Fatalf("orden inicial incorrecto: %v", got)
	}

	// Reorder: C, A, B.
	if err := store.Reorder(ctx, []int64{casaC.ID, casaA.ID, casaB.ID}); err != nil {
		t.Fatalf("Reorder falló: %v", err)
	}
	homes, err = store.List(ctx)
	if err != nil {
		t.Fatalf("List falló: %v", err)
	}
	if got := homeNames(homes); strings.Join(got, ",") != "Casa C,Casa A,Casa B" {
		t.Fatalf("orden tras reorder incorrecto: %v", got)
	}

	// Una casa nueva se agrega al final.
	mustCreateHome(t, store, ctx, "Casa D")
	homes, err = store.List(ctx)
	if err != nil {
		t.Fatalf("List falló: %v", err)
	}
	if got := homeNames(homes); strings.Join(got, ",") != "Casa C,Casa A,Casa B,Casa D" {
		t.Fatalf("casa nueva debería ir al final: %v", got)
	}
}

func TestHomeReorderRejectsInvalidIDAndRollsBack(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	store := NewHomeStorage(database)
	ctx := context.Background()

	casaA := mustCreateHome(t, store, ctx, "Casa A")
	casaB := mustCreateHome(t, store, ctx, "Casa B")

	// Un ID inexistente debe fallar y no cambiar el orden previo.
	if err := store.Reorder(ctx, []int64{casaB.ID, 9999, casaA.ID}); err == nil {
		t.Fatal("se esperaba error al reordenar con ID inexistente")
	}

	homes, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List falló: %v", err)
	}
	if got := homeNames(homes); strings.Join(got, ",") != "Casa A,Casa B" {
		t.Fatalf("el orden no debería haber cambiado tras el error: %v", got)
	}
}

func mustCreateHome(t *testing.T, store *HomeStorage, ctx context.Context, name string) *models.Home {
	t.Helper()
	home, err := store.Create(ctx, name, "")
	if err != nil {
		t.Fatalf("crear casa %q: %v", name, err)
	}
	return home
}

func homeNames(homes []models.Home) []string {
	names := make([]string, 0, len(homes))
	for _, h := range homes {
		names = append(names, h.Name)
	}
	return names
}
