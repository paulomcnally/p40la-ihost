package services

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func TestAutomationAccountOptions(t *testing.T) {
	database, err := db.OpenDB(filepath.Join(t.TempDir(), "app.db"), "../../migrations")
	if err != nil {
		t.Fatalf("abrir DB: %v", err)
	}
	defer database.Close()

	if _, err := database.Exec(`
		INSERT INTO institutions (id, name) VALUES (1, 'Tigo');
		INSERT INTO automation_plugins (id, name, version, source, status)
			VALUES (1, 'tigo.nicaragua', '1.0.0', 'builtin', 'active');
		INSERT INTO automation_accounts
			(id, institution_id, identifier, credentials, plugin_name, schedule_frequency)
			VALUES (7, 1, 'demo-account-004', '{}', 'tigo.nicaragua', 'daily');
	`); err != nil {
		t.Fatalf("insertar fixture: %v", err)
	}

	svc := NewAutomationService(storage.NewAutomationStorage(database))
	options, err := svc.ListAccountOptions(context.Background())
	if err != nil {
		t.Fatalf("ListAccountOptions: %v", err)
	}
	if len(options) != 1 {
		t.Fatalf("opciones = %d, esperado 1", len(options))
	}
	want := "Tigo · demo-account-004 (tigo.nicaragua)"
	if options[0].Label != want {
		t.Errorf("label = %q, esperado %q", options[0].Label, want)
	}
}
