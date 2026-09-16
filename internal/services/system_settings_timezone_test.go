package services

import (
	"context"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newTimezoneTestService(t *testing.T) *SystemSettingsService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
}

func TestSetTimezone_Valid(t *testing.T) {
	ctx := context.Background()
	s := newTimezoneTestService(t)

	if err := s.SetTimezone(ctx, "America/Argentina/Buenos_Aires"); err != nil {
		t.Fatalf("SetTimezone con IANA válido falló: %v", err)
	}
	got, err := s.GetTimezone(ctx)
	if err != nil {
		t.Fatalf("GetTimezone: %v", err)
	}
	if got != "America/Argentina/Buenos_Aires" {
		t.Errorf("timezone esperada America/Argentina/Buenos_Aires, got %q", got)
	}
}

func TestSetTimezone_Invalid(t *testing.T) {
	ctx := context.Background()
	s := newTimezoneTestService(t)

	if err := s.SetTimezone(ctx, "Foo/Bar"); err == nil {
		t.Fatalf("SetTimezone con IANA inválido NO devolvió error")
	}
	got, err := s.GetTimezone(ctx)
	if err != nil {
		t.Fatalf("GetTimezone: %v", err)
	}
	if got != "" {
		t.Errorf("timezone inválida no debió persistirse, got %q", got)
	}
}

func TestSetTimezone_EmptyClears(t *testing.T) {
	ctx := context.Background()
	s := newTimezoneTestService(t)

	if err := s.SetTimezone(ctx, "America/Managua"); err != nil {
		t.Fatalf("SetTimezone: %v", err)
	}
	if err := s.SetTimezone(ctx, ""); err != nil {
		t.Fatalf("SetTimezone vacío: %v", err)
	}
	got, err := s.GetTimezone(ctx)
	if err != nil {
		t.Fatalf("GetTimezone: %v", err)
	}
	if got != "" {
		t.Errorf("timezone debió limpiarse, got %q", got)
	}
}

func TestGetTimezoneLocation_FallbackUTC(t *testing.T) {
	ctx := context.Background()
	s := newTimezoneTestService(t)

	// Sin setting → UTC.
	loc, err := s.GetTimezoneLocation(ctx)
	if err != nil {
		t.Fatalf("GetTimezoneLocation: %v", err)
	}
	if loc != time.UTC {
		t.Errorf("sin timezone se esperaba UTC, got %v", loc)
	}

	// Valor corrupto en DB → UTC.
	if err := s.Set(ctx, TimezoneKey, "Not/AZone"); err != nil {
		t.Fatalf("Set raw: %v", err)
	}
	loc, err = s.GetTimezoneLocation(ctx)
	if err != nil {
		t.Fatalf("GetTimezoneLocation: %v", err)
	}
	if loc != time.UTC {
		t.Errorf("timezone inválida en DB se esperaba UTC, got %v", loc)
	}
}

func TestCurrentUserNow_ArgentinaOffset(t *testing.T) {
	ctx := context.Background()
	s := newTimezoneTestService(t)

	// Buenos Aires es UTC-3 fijo (sin DST desde 2009).
	if err := s.SetTimezone(ctx, "America/Argentina/Buenos_Aires"); err != nil {
		t.Fatalf("SetTimezone: %v", err)
	}
	local, err := currentUserNow(ctx, s)
	if err != nil {
		t.Fatalf("currentUserNow: %v", err)
	}
	utcHour := time.Now().UTC().Hour()
	localHour := local.Hour()
	want := (utcHour - 3 + 24) % 24
	if localHour != want {
		t.Errorf("hora Buenos Aires esperada %d, got %d (utc %d)", want, localHour, utcHour)
	}
}

func TestCurrentUserNow_DefaultUTC(t *testing.T) {
	ctx := context.Background()
	s := newTimezoneTestService(t)

	local, err := currentUserNow(ctx, s)
	if err != nil {
		t.Fatalf("currentUserNow: %v", err)
	}
	if local.Location() != time.UTC {
		t.Errorf("sin timezone se esperaba UTC, got %v", local.Location())
	}
}
