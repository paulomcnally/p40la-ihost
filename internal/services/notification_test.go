package services

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func setupNotificationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`
		CREATE TABLE notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestNotificationServiceCRUD(t *testing.T) {
	db := setupNotificationTestDB(t)
	svc := NewNotificationService(storage.NewNotificationStorage(db))
	ctx := context.Background()

	// Create
	n, err := svc.Create(ctx, "María Pérez", "maria@example.com", true)
	if err != nil {
		t.Fatalf("create falló: %v", err)
	}
	if !n.Active || n.Name != "María Pérez" {
		t.Fatalf("valores incorrectos: %+v", n)
	}

	// List
	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("list falló: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("se esperaba 1 registro, got=%d", len(list))
	}

	// GetByID
	got, err := svc.GetByID(ctx, n.ID)
	if err != nil || got == nil {
		t.Fatalf("get falló: %v", err)
	}

	// Update
	upd, err := svc.Update(ctx, n.ID, "María Pérez", "maria@example.com", false)
	if err != nil {
		t.Fatalf("update falló: %v", err)
	}
	if upd.Active {
		t.Fatal("active debería ser false tras update")
	}

	// Delete
	if err := svc.Delete(ctx, n.ID); err != nil {
		t.Fatalf("delete falló: %v", err)
	}
	list, _ = svc.List(ctx)
	if len(list) != 0 {
		t.Fatalf("se esperaba 0 registros tras delete, got=%d", len(list))
	}
}

func TestNotificationServiceValidation(t *testing.T) {
	db := setupNotificationTestDB(t)
	svc := NewNotificationService(storage.NewNotificationStorage(db))
	ctx := context.Background()

	if _, err := svc.Create(ctx, "", "maria@example.com", true); err == nil {
		t.Fatal("nombre vacío debería fallar")
	}
	if _, err := svc.Create(ctx, "María", "", true); err == nil {
		t.Fatal("email vacío debería fallar")
	}
	if _, err := svc.Create(ctx, "María", "email-invalido", true); err == nil {
		t.Fatal("email inválido debería fallar")
	}
}

func TestNotificationServiceDuplicateEmail(t *testing.T) {
	db := setupNotificationTestDB(t)
	svc := NewNotificationService(storage.NewNotificationStorage(db))
	ctx := context.Background()

	if _, err := svc.Create(ctx, "María", "maria@example.com", true); err != nil {
		t.Fatalf("primer create falló: %v", err)
	}
	_, err := svc.Create(ctx, "Otra", "maria@example.com", true)
	if err != storage.ErrDuplicateEmail {
		t.Fatalf("se esperaba ErrDuplicateEmail, got=%v", err)
	}

	// Update a un email ya existente en otro registro.
	second, err := svc.Create(ctx, "Juan", "juan@example.com", true)
	if err != nil {
		t.Fatalf("segundo create falló: %v", err)
	}
	_, err = svc.Update(ctx, second.ID, "Juan", "maria@example.com", true)
	if err != storage.ErrDuplicateEmail {
		t.Fatalf("update duplicado debería dar ErrDuplicateEmail, got=%v", err)
	}
}
