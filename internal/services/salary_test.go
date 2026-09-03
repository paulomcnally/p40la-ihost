package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newSalaryService(t *testing.T) *SalaryService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewSalaryService(storage.NewSalaryStorage(database), storage.NewCurrencyStorage(database))
}

func TestSalaryService_CreateValidation(t *testing.T) {
	s := newSalaryService(t)
	ctx := context.Background()

	cases := []struct {
		name       string
		employer   string
		amount     float64
		currencyID int64
		paymentDay int
		active     bool
		note       string
		wantErr    bool
	}{
		{"campos requeridos ok", "Empresa XYZ", 15000, 1, 15, true, "", false},
		{"empleador vacío", "", 15000, 1, 15, true, "", true},
		{"empleador solo espacios", "   ", 15000, 1, 15, true, "", true},
		{"monto cero", "Empresa XYZ", 0, 1, 15, true, "", true},
		{"monto negativo", "Empresa XYZ", -100, 1, 15, true, "", true},
		{"día de pago cero", "Empresa XYZ", 15000, 1, 0, true, "", true},
		{"día de pago mayor a 31", "Empresa XYZ", 15000, 1, 32, true, "", true},
		{"moneda inexistente", "Empresa XYZ", 15000, 9999, 15, true, "", true},
		{"moneda cero", "Empresa XYZ", 15000, 0, 15, true, "", true},
		{"active false ok", "Empresa XYZ", 15000, 1, 15, false, "", false},
		{"con nota opcional", "Empresa XYZ", 15000, 1, 15, true, "Pago mensual", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.Create(ctx, tc.employer, tc.amount, tc.currencyID, tc.paymentDay, tc.active, tc.note)
			if tc.wantErr && err == nil {
				t.Fatalf("se esperaba error pero no hubo")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("no se esperaba error, got: %v", err)
			}
		})
	}
}

func TestSalaryService_CRUD(t *testing.T) {
	s := newSalaryService(t)
	ctx := context.Background()

	salary, err := s.Create(ctx, "Empresa XYZ", 15000, 1, 15, true, "Pago mensual")
	if err != nil {
		t.Fatalf("crear salario: %v", err)
	}
	if salary.ID == 0 {
		t.Fatalf("se esperaba id asignado")
	}
	if !salary.Active {
		t.Fatalf("se esperaba active=true por defecto")
	}
	if salary.Employer != "Empresa XYZ" || salary.Amount != 15000 || salary.CurrencyID != 1 || salary.PaymentDay != 15 || salary.Note != "Pago mensual" {
		t.Fatalf("datos incorrectos tras crear: %+v", salary)
	}

	got, err := s.GetByID(ctx, salary.ID)
	if err != nil {
		t.Fatalf("get salario: %v", err)
	}
	if got == nil || got.ID != salary.ID {
		t.Fatalf("get salario no devolvió el esperado: %+v", got)
	}

	updated, err := s.Update(ctx, salary.ID, "Otra Empresa", 20000, 2, 1, false, "")
	if err != nil {
		t.Fatalf("update salario: %v", err)
	}
	if updated.Employer != "Otra Empresa" || updated.Amount != 20000 || updated.CurrencyID != 2 || updated.PaymentDay != 1 || updated.Active {
		t.Fatalf("update no aplicó cambios: %+v", updated)
	}

	salaries, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list salarios: %v", err)
	}
	if len(salaries) != 1 {
		t.Fatalf("se esperaba 1 salario, count=%d", len(salaries))
	}

	if err := s.Delete(ctx, salary.ID); err != nil {
		t.Fatalf("delete salario: %v", err)
	}
	gone, err := s.GetByID(ctx, salary.ID)
	if err != nil {
		t.Fatalf("get salario tras delete: %v", err)
	}
	if gone != nil {
		t.Fatalf("el salario debía haber sido eliminado")
	}
}

func TestSalaryService_DefaultActive(t *testing.T) {
	s := newSalaryService(t)
	ctx := context.Background()

	salary, err := s.Create(ctx, "Empresa XYZ", 15000, 1, 15, true, "")
	if err != nil {
		t.Fatalf("crear salario: %v", err)
	}
	if !salary.Active {
		t.Fatalf("se esperaba active=true")
	}
}