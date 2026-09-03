package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newMonthClosingService(t *testing.T) *MonthClosingService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewMonthClosingService(storage.NewMonthClosingStorage(database))
}

func TestMonthClosingService_CloseReopen(t *testing.T) {
	s := newMonthClosingService(t)
	ctx := context.Background()

	closed, err := s.IsClosed(ctx, 2026, 8)
	if err != nil {
		t.Fatalf("is closed: %v", err)
	}
	if closed {
		t.Fatal("mes no debería estar cerrado al inicio")
	}

	closing, err := s.Close(ctx, 2026, 8)
	if err != nil {
		t.Fatalf("cerrar mes: %v", err)
	}
	if closing.ID == 0 {
		t.Fatal("se esperaba id de cierre")
	}

	closed, err = s.IsClosed(ctx, 2026, 8)
	if err != nil {
		t.Fatalf("is closed tras cerrar: %v", err)
	}
	if !closed {
		t.Fatal("el mes debería estar cerrado")
	}

	// Cerrar de nuevo falla
	if _, err := s.Close(ctx, 2026, 8); err == nil {
		t.Fatal("se esperaba error al cerrar un mes ya cerrado")
	}

	// Reabrir funciona
	if err := s.Reopen(ctx, 2026, 8); err != nil {
		t.Fatalf("reabrir mes: %v", err)
	}
	closed, err = s.IsClosed(ctx, 2026, 8)
	if err != nil {
		t.Fatalf("is closed tras reabrir: %v", err)
	}
	if closed {
		t.Fatal("el mes debería estar abierto tras reabrir")
	}

	// Reabrir un mes no cerrado falla
	if err := s.Reopen(ctx, 2026, 8); err == nil {
		t.Fatal("se esperaba error al reabrir un mes no cerrado")
	}
}

func TestMonthClosingService_InvalidPeriod(t *testing.T) {
	s := newMonthClosingService(t)
	ctx := context.Background()

	if _, err := s.Close(ctx, 2026, 0); err == nil {
		t.Fatal("se esperaba error con mes 0")
	}
	if _, err := s.Close(ctx, 2026, 13); err == nil {
		t.Fatal("se esperaba error con mes 13")
	}
	if _, err := s.Close(ctx, 1999, 8); err == nil {
		t.Fatal("se esperaba error con año inválido")
	}
}
