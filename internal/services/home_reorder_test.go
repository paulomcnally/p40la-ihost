package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func TestHomeServiceReorderValidation(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	svc := NewHomeService(storage.NewHomeStorage(database))
	ctx := context.Background()

	// Lista vacía → error.
	if err := svc.Reorder(ctx, nil); err == nil {
		t.Fatal("se esperaba error con lista vacía")
	}

	// IDs duplicados → error.
	if err := svc.Reorder(ctx, []int64{1, 1}); err == nil {
		t.Fatal("se esperaba error con IDs duplicados")
	}
}
