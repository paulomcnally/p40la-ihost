package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newPensionCategoryService(t *testing.T) *PensionCategoryService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewPensionCategoryService(storage.NewPensionCategoryStorage(database))
}

func TestPensionCategoryService_CreateValidation(t *testing.T) {
	s := newPensionCategoryService(t)
	ctx := context.Background()

	cases := []struct {
		name         string
		catName      string
		desc         string
		autoGenerate bool
		wantErr      bool
	}{
		{"nombre requerido ok", "Alimentación", "", false, false},
		{"con descripción y auto_generate", "Alimentación", "Gastos de supermercado", true, false},
		{"nombre vacío", "", "", false, true},
		{"nombre solo espacios", "   ", "", false, true},
		{"descripción vacía ok", "Educación", "", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cat, err := s.Create(ctx, tc.catName, tc.desc, tc.autoGenerate)
			if tc.wantErr && err == nil {
				t.Fatalf("se esperaba error pero no hubo")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("no se esperaba error, got: %v", err)
			}
			if tc.wantErr {
				return
			}
			if cat.AutoGenerate != tc.autoGenerate {
				t.Fatalf("auto_generate incorrecto: got=%v want=%v", cat.AutoGenerate, tc.autoGenerate)
			}
		})
	}
}

func TestPensionCategoryService_CRUD(t *testing.T) {
	s := newPensionCategoryService(t)
	ctx := context.Background()

	cat, err := s.Create(ctx, "Alimentación", "Gastos de supermercado y comidas", true)
	if err != nil {
		t.Fatalf("crear categoría: %v", err)
	}
	if cat.ID == 0 {
		t.Fatalf("se esperaba id asignado")
	}
	if cat.Name != "Alimentación" || !cat.AutoGenerate {
		t.Fatalf("datos incorrectos tras crear: %+v", cat)
	}

	got, err := s.GetByID(ctx, cat.ID)
	if err != nil {
		t.Fatalf("get categoría: %v", err)
	}
	if got == nil || got.ID != cat.ID {
		t.Fatalf("get no devolvió la esperada: %+v", got)
	}

	updated, err := s.Update(ctx, cat.ID, "Educación", "", false)
	if err != nil {
		t.Fatalf("update categoría: %v", err)
	}
	if updated.Name != "Educación" || updated.AutoGenerate {
		t.Fatalf("update no aplicó cambios: %+v", updated)
	}

	categories, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list categorías: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("se esperaba 1 categoría, count=%d", len(categories))
	}

	if err := s.Delete(ctx, cat.ID); err != nil {
		t.Fatalf("delete categoría: %v", err)
	}
	gone, err := s.GetByID(ctx, cat.ID)
	if err != nil {
		t.Fatalf("get tras delete: %v", err)
	}
	if gone != nil {
		t.Fatalf("la categoría debía haber sido eliminada")
	}
}