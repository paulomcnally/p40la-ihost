package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type configFixture struct {
	svc     *ChildSupportConfigService
	childID int64
	catID   int64
}

func newConfigFixture(t *testing.T) *configFixture {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	childStorage := storage.NewChildStorage(database)
	catStorage := storage.NewPensionCategoryStorage(database)
	svc := NewChildSupportConfigService(storage.NewChildSupportConfigStorage(database), childStorage, catStorage)

	ctx := context.Background()
	childSvc := NewChildService(childStorage)
	child, err := childSvc.Create(ctx, "Juan", "Pérez", "2015-01-01", "")
	if err != nil {
		t.Fatalf("crear hijo: %v", err)
	}
	catSvc := NewPensionCategoryService(catStorage)
	cat, err := catSvc.Create(ctx, "Colegio", "", false)
	if err != nil {
		t.Fatalf("crear categoría: %v", err)
	}

	return &configFixture{svc: svc, childID: child.ID, catID: cat.ID}
}

func TestChildSupportConfigService_CRUD(t *testing.T) {
	f := newConfigFixture(t)
	ctx := context.Background()

	cfg, err := f.svc.Create(ctx, f.childID, f.catID, 1500, "NIO", true, true)
	if err != nil {
		t.Fatalf("crear config: %v", err)
	}
	if cfg.ID == 0 || !cfg.IsActive || !cfg.AutoGenerate || cfg.ChildName == "" || cfg.CategoryName == "" {
		t.Fatalf("datos incorrectos tras crear: %+v", cfg)
	}

	// Duplicado
	_, err = f.svc.Create(ctx, f.childID, f.catID, 1500, "NIO", true, true)
	if err == nil {
		t.Fatal("se esperaba error por duplicado")
	}

	// Update
	updated, err := f.svc.Update(ctx, cfg.ID, f.catID, 1800, "usd", false, false)
	if err != nil {
		t.Fatalf("update config: %v", err)
	}
	if updated.Amount != 1800 || updated.Currency != "USD" || updated.IsActive || updated.AutoGenerate {
		t.Fatalf("update no aplicó cambios: %+v", updated)
	}

	// List
	configs, err := f.svc.List(ctx)
	if err != nil {
		t.Fatalf("list configs: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("se esperaba 1 config, count=%d", len(configs))
	}

	// Delete
	if err := f.svc.Delete(ctx, cfg.ID); err != nil {
		t.Fatalf("delete config: %v", err)
	}
	gone, err := f.svc.GetByID(ctx, cfg.ID)
	if err != nil {
		t.Fatalf("get config tras delete: %v", err)
	}
	if gone != nil {
		t.Fatal("la config debía haber sido eliminada")
	}
}

func TestChildSupportConfigService_Validation(t *testing.T) {
	f := newConfigFixture(t)
	ctx := context.Background()

	cases := []struct {
		name    string
		childID int64
		catID   int64
		amount  float64
		cur     string
		wantErr bool
	}{
		{"ok", f.childID, f.catID, 1500, "NIO", false},
		{"hijo inexistente", 9999, f.catID, 1500, "NIO", true},
		{"categoría inexistente", f.childID, 9999, 1500, "NIO", true},
		{"monto cero", f.childID, f.catID, 0, "NIO", true},
		{"moneda inválida", f.childID, f.catID, 1500, "NIOOO", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.svc.Create(ctx, tc.childID, tc.catID, tc.amount, tc.cur, true, false)
			if tc.wantErr && err == nil {
				t.Fatalf("se esperaba error pero no hubo")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("no se esperaba error, got: %v", err)
			}
		})
	}
}
