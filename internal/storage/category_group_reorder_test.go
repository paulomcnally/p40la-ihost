package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func TestCategoryGroupReorderPersistsOrder(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	store := NewCategoryGroupStorage(database)
	ctx := context.Background()

	next, err := store.NextSortOrder(ctx)
	if err != nil {
		t.Fatalf("NextSortOrder falló: %v", err)
	}
	if next != 0 {
		t.Fatalf("primer NextSortOrder debería ser 0, got %d", next)
	}

	gA := mustCreateGroup(t, store, ctx, "Grupo A")
	gB := mustCreateGroup(t, store, ctx, "Grupo B")
	gC := mustCreateGroup(t, store, ctx, "Grupo C")

	// Orden inicial: por sort_order (insert secuencial vía NextSortOrder).
	groups, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List falló: %v", err)
	}
	if got := groupNames(groups); strings.Join(got, ",") != "Grupo A,Grupo B,Grupo C" {
		t.Fatalf("orden inicial incorrecto: %v", got)
	}

	// Reorder: C, A, B.
	if err := store.Reorder(ctx, []int64{gC.ID, gA.ID, gB.ID}); err != nil {
		t.Fatalf("Reorder falló: %v", err)
	}
	groups, err = store.List(ctx)
	if err != nil {
		t.Fatalf("List falló: %v", err)
	}
	if got := groupNames(groups); strings.Join(got, ",") != "Grupo C,Grupo A,Grupo B" {
		t.Fatalf("orden tras reorder incorrecto: %v", got)
	}

	// Un grupo nuevo se agrega al final.
	mustCreateGroup(t, store, ctx, "Grupo D")
	groups, err = store.List(ctx)
	if err != nil {
		t.Fatalf("List falló: %v", err)
	}
	if got := groupNames(groups); strings.Join(got, ",") != "Grupo C,Grupo A,Grupo B,Grupo D" {
		t.Fatalf("grupo nuevo debería ir al final: %v", got)
	}
}

func TestCategoryGroupNextSortOrderAfterReorder(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	store := NewCategoryGroupStorage(database)
	ctx := context.Background()

	gA := mustCreateGroup(t, store, ctx, "Grupo A")
	gB := mustCreateGroup(t, store, ctx, "Grupo B")

	// Reorder cambia los sort_order a 1,0.
	if err := store.Reorder(ctx, []int64{gB.ID, gA.ID}); err != nil {
		t.Fatalf("Reorder falló: %v", err)
	}

	next, err := store.NextSortOrder(ctx)
	if err != nil {
		t.Fatalf("NextSortOrder falló: %v", err)
	}
	if next != 2 {
		t.Fatalf("NextSortOrder tras reorder debería ser 2, got %d", next)
	}
}

func mustCreateGroup(t *testing.T, store *CategoryGroupStorage, ctx context.Context, name string) *models.CategoryGroup {
	t.Helper()
	next, err := store.NextSortOrder(ctx)
	if err != nil {
		t.Fatalf("NextSortOrder falló: %v", err)
	}
	group, err := store.Create(ctx, name, "tag", next)
	if err != nil {
		t.Fatalf("crear grupo %q: %v", name, err)
	}
	return group
}

func groupNames(groups []models.CategoryGroup) []string {
	names := make([]string, 0, len(groups))
	for _, g := range groups {
		names = append(names, g.Name)
	}
	return names
}