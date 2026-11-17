package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newCountryTestService(t *testing.T) *SystemSettingsService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
}

func TestGetCountryCode_Default(t *testing.T) {
	ctx := context.Background()
	s := newCountryTestService(t)

	got, err := s.GetCountryCode(ctx)
	if err != nil {
		t.Fatalf("GetCountryCode: %v", err)
	}
	if got != DefaultCountryCode {
		t.Errorf("sin setting se esperaba %q, got %q", DefaultCountryCode, got)
	}
}

func TestGetCountryCode_EmptyFallsBackToDefault(t *testing.T) {
	ctx := context.Background()
	s := newCountryTestService(t)

	if err := s.Set(ctx, CountryCodeKey, ""); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := s.GetCountryCode(ctx)
	if err != nil {
		t.Fatalf("GetCountryCode: %v", err)
	}
	if got != DefaultCountryCode {
		t.Errorf("con setting vacío se esperaba %q, got %q", DefaultCountryCode, got)
	}
}

func TestSetCountryCode_NormalizesAndPersists(t *testing.T) {
	ctx := context.Background()
	s := newCountryTestService(t)

	if err := s.SetCountryCode(ctx, " us "); err != nil {
		t.Fatalf("SetCountryCode: %v", err)
	}
	got, err := s.GetCountryCode(ctx)
	if err != nil {
		t.Fatalf("GetCountryCode: %v", err)
	}
	if got != "US" {
		t.Errorf("se esperaba US, got %q", got)
	}
}

func TestSetCountryCode_Invalid(t *testing.T) {
	ctx := context.Background()
	s := newCountryTestService(t)

	for _, code := range []string{"USA", "1", "N!", "Nicaragua"} {
		if err := s.SetCountryCode(ctx, code); err == nil {
			t.Errorf("SetCountryCode(%q) debió fallar", code)
		}
	}

	got, err := s.GetCountryCode(ctx)
	if err != nil {
		t.Fatalf("GetCountryCode: %v", err)
	}
	if got != DefaultCountryCode {
		t.Errorf("un valor inválido no debió persistirse, got %q", got)
	}
}

func TestSetCountryCode_EmptyClears(t *testing.T) {
	ctx := context.Background()
	s := newCountryTestService(t)

	if err := s.SetCountryCode(ctx, "ES"); err != nil {
		t.Fatalf("SetCountryCode: %v", err)
	}
	if err := s.SetCountryCode(ctx, ""); err != nil {
		t.Fatalf("SetCountryCode vacío: %v", err)
	}
	got, err := s.GetCountryCode(ctx)
	if err != nil {
		t.Fatalf("GetCountryCode: %v", err)
	}
	if got != DefaultCountryCode {
		t.Errorf("tras limpiar se esperaba %q, got %q", DefaultCountryCode, got)
	}
}

func TestGetCountryCode_InvalidStoredValueFallsBack(t *testing.T) {
	ctx := context.Background()
	s := newCountryTestService(t)

	if err := s.Set(ctx, CountryCodeKey, "Nicaragua"); err != nil {
		t.Fatalf("Set raw: %v", err)
	}
	got, err := s.GetCountryCode(ctx)
	if err != nil {
		t.Fatalf("GetCountryCode: %v", err)
	}
	if got != DefaultCountryCode {
		t.Errorf("valor corrupto se esperaba %q, got %q", DefaultCountryCode, got)
	}
}
