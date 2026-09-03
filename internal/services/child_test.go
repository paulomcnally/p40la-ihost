package services

import (
	"context"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newChildService(t *testing.T) *ChildService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewChildService(storage.NewChildStorage(database))
}

func TestChildService_CreateValidation(t *testing.T) {
	s := newChildService(t)
	ctx := context.Background()

	cases := []struct {
		name      string
		firstName string
		lastName  string
		birthDate string
		notes     string
		wantErr   bool
	}{
		{"campos requeridos ok", "María", "Pérez", "2019-05-12", "", false},
		{"nombres vacío", "", "Pérez", "2019-05-12", "", true},
		{"nombres solo espacios", "   ", "Pérez", "2019-05-12", "", true},
		{"apellidos vacío", "María", "", "2019-05-12", "", true},
		{"fecha vacía", "María", "Pérez", "", "", true},
		{"fecha formato inválido", "María", "Pérez", "12/05/2019", "", true},
		{"fecha futura", "María", "Pérez", time.Now().AddDate(1, 0, 0).Format("2006-01-02"), "", true},
		{"fecha de hoy ok", "María", "Pérez", time.Now().Format("2006-01-02"), "", false},
		{"con notas opcionales", "Juan", "Gómez", "2015-03-01", "Alergia al maní", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.Create(ctx, tc.firstName, tc.lastName, tc.birthDate, tc.notes)
			if tc.wantErr && err == nil {
				t.Fatalf("se esperaba error pero no hubo")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("no se esperaba error, got: %v", err)
			}
		})
	}
}

func TestChildService_CRUD(t *testing.T) {
	s := newChildService(t)
	ctx := context.Background()

	child, err := s.Create(ctx, "María", "Pérez", "2019-05-12", "Alergia al maní")
	if err != nil {
		t.Fatalf("crear hijo: %v", err)
	}
	if child.ID == 0 {
		t.Fatalf("se esperaba id asignado")
	}
	if child.FirstName != "María" || child.LastName != "Pérez" || child.BirthDate != "2019-05-12" || child.Notes != "Alergia al maní" {
		t.Fatalf("datos incorrectos tras crear: %+v", child)
	}

	got, err := s.GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("get hijo: %v", err)
	}
	if got == nil || got.ID != child.ID {
		t.Fatalf("get hijo no devolvió el esperado: %+v", got)
	}

	updated, err := s.Update(ctx, child.ID, "María Elena", "Pérez", "2019-05-12", "")
	if err != nil {
		t.Fatalf("update hijo: %v", err)
	}
	if updated.FirstName != "María Elena" || updated.Notes != "" {
		t.Fatalf("update no aplicó cambios: %+v", updated)
	}

	children, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list hijos: %v", err)
	}
	if len(children) != 1 {
		t.Fatalf("se esperaba 1 hijo, count=%d", len(children))
	}

	if err := s.Delete(ctx, child.ID); err != nil {
		t.Fatalf("delete hijo: %v", err)
	}
	gone, err := s.GetByID(ctx, child.ID)
	if err != nil {
		t.Fatalf("get hijo tras delete: %v", err)
	}
	if gone != nil {
		t.Fatalf("el hijo debía haber sido eliminado")
	}
}