package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newTestAutoService(t *testing.T) *AutoService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewAutoService(storage.NewAutoStorage(database))
}

func TestAutoServiceCreateConCamposOpcionalesVacios(t *testing.T) {
	svc := newTestAutoService(t)
	ctx := context.Background()

	auto, err := svc.Create(ctx, 2026, "Corolla", "Toyota", "WHITE", "", "", "", "", "")
	if err != nil {
		t.Fatalf("crear auto con opcionales vacíos: %v", err)
	}
	if auto.Color != "white" {
		t.Errorf("color normalizado: esperado %q, got %q", "white", auto.Color)
	}
	if auto.Icon != "vehicle" {
		t.Errorf("icon por defecto: esperado vehicle, got %q", auto.Icon)
	}
	if auto.Motor != "" || auto.Chasis != "" || auto.VIN != "" || auto.Placa != "" {
		t.Errorf("campos opcionales deberían quedar vacíos: %+v", auto)
	}
}

func TestAutoServiceCreateColorInvalido(t *testing.T) {
	svc := newTestAutoService(t)
	ctx := context.Background()

	cases := []struct {
		name  string
		color string
		want  string
	}{
		{"vacío", "  ", "el color es requerido"},
		{"fuera de paleta", "blanco", "el color no es válido"},
		{"código inexistente", "neon", "el color no es válido"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(ctx, 2026, "Corolla", "Toyota", tc.color, "vehicle", "", "", "", "")
			if err == nil || err.Error() != tc.want {
				t.Fatalf("esperado error %q, got %v", tc.want, err)
			}
		})
	}
}

func TestAutoServiceUpdatePermiteLimpiarOpcionales(t *testing.T) {
	svc := newTestAutoService(t)
	ctx := context.Background()

	auto, err := svc.Create(ctx, 2020, "Hilux", "Toyota", "black", "truck", "2ZR12345678", "JTDBU4EE1B9123456", "JTDBU4EE1B9123456", "ABC123")
	if err != nil {
		t.Fatalf("crear auto: %v", err)
	}

	updated, err := svc.Update(ctx, auto.ID, 2020, "Hilux", "Toyota", "silver", "truck", "", "", "", "")
	if err != nil {
		t.Fatalf("actualizar auto: %v", err)
	}
	if updated.Color != "silver" {
		t.Errorf("color: esperado silver, got %q", updated.Color)
	}
	if updated.Motor != "" || updated.Chasis != "" || updated.VIN != "" || updated.Placa != "" {
		t.Errorf("los campos opcionales deberían quedar vacíos: %+v", updated)
	}
}

func TestAutoServiceRequiereCamposObligatorios(t *testing.T) {
	svc := newTestAutoService(t)
	ctx := context.Background()

	cases := []struct {
		name  string
		year  int64
		model string
		brand string
		color string
		want  string
	}{
		{"modelo", 2026, "  ", "Toyota", "white", "el modelo es requerido"},
		{"marca", 2026, "Corolla", " ", "white", "la marca es requerida"},
		{"año", 1800, "Corolla", "Toyota", "white", "el año debe estar entre 1900 y 2100"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(ctx, tc.year, tc.model, tc.brand, tc.color, "vehicle", "", "", "", "")
			if err == nil || err.Error() != tc.want {
				t.Fatalf("esperado error %q, got %v", tc.want, err)
			}
		})
	}
}
