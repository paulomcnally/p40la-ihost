package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func TestCategoryGroupServiceCreateAppendsAtEnd(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	groupSvc := NewCategoryGroupService(storage.NewCategoryGroupStorage(database), storage.NewCategoryStorage(database))
	ctx := context.Background()

	g1, err := groupSvc.Create(ctx, "Grupo 1", "tag")
	if err != nil {
		t.Fatalf("crear grupo 1: %v", err)
	}
	g2, err := groupSvc.Create(ctx, "Grupo 2", "tag")
	if err != nil {
		t.Fatalf("crear grupo 2: %v", err)
	}
	g3, err := groupSvc.Create(ctx, "Grupo 3", "tag")
	if err != nil {
		t.Fatalf("crear grupo 3: %v", err)
	}

	// Reordenar: 3, 1, 2.
	if err := groupSvc.Reorder(ctx, []int64{g3.ID, g1.ID, g2.ID}); err != nil {
		t.Fatalf("Reorder falló: %v", err)
	}

	// Un grupo nuevo debe quedar al final (sort_order = 3).
	g4, err := groupSvc.Create(ctx, "Grupo 4", "tag")
	if err != nil {
		t.Fatalf("crear grupo 4: %v", err)
	}
	if g4.SortOrder != 3 {
		t.Fatalf("grupo nuevo debería tener sort_order 3, got %d", g4.SortOrder)
	}

	groups, err := groupSvc.List(ctx)
	if err != nil {
		t.Fatalf("List falló: %v", err)
	}
	names := make([]string, 0, len(groups))
	for _, g := range groups {
		names = append(names, g.Name)
	}
	expected := []string{"Grupo 3", "Grupo 1", "Grupo 2", "Grupo 4"}
	if len(names) != len(expected) {
		t.Fatalf("cantidad de grupos incorrecta: got %v", names)
	}
	for i := range expected {
		if names[i] != expected[i] {
			t.Fatalf("orden final incorrecto: got %v, expected %v", names, expected)
		}
	}
}

func TestCategoryGroupServiceReorderValidation(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	groupSvc := NewCategoryGroupService(storage.NewCategoryGroupStorage(database), storage.NewCategoryStorage(database))
	ctx := context.Background()

	// Lista vacía → error.
	if err := groupSvc.Reorder(ctx, nil); err == nil {
		t.Fatal("se esperaba error con lista vacía")
	}

	// IDs duplicados → error.
	if err := groupSvc.Reorder(ctx, []int64{1, 1}); err == nil {
		t.Fatal("se esperaba error con IDs duplicados")
	}
}