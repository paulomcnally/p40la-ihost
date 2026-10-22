package services

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

const demoPlugin = "demo.provider"

func newAutomationTestService(t *testing.T) (*AutomationService, *sql.DB) {
	t.Helper()
	database, err := db.OpenDB(filepath.Join(t.TempDir(), "app.db"), "../../migrations")
	if err != nil {
		t.Fatalf("abrir DB: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if _, err := database.Exec(`
		INSERT INTO institutions (id, name) VALUES (1, 'Proveedor Demo');
		INSERT INTO automation_plugins (id, name, version, source, status)
			VALUES (1, 'demo.provider', '1.0.0', 'builtin', 'active');
		INSERT INTO homes (id, name, sort_order) VALUES (1, 'Hogar Demo', 0);
	`); err != nil {
		t.Fatalf("insertar fixture: %v", err)
	}
	return NewAutomationService(storage.NewAutomationStorage(database)), database
}

func TestAutomationAccountOptions(t *testing.T) {
	svc, database := newAutomationTestService(t)
	if _, err := database.Exec(`
		INSERT INTO automation_accounts
			(id, institution_id, identifier, credentials, plugin_name, schedule_frequency)
			VALUES (7, 1, 'demo-account-001', '{}', 'demo.provider', 'daily');
	`); err != nil {
		t.Fatalf("insertar cuenta: %v", err)
	}

	options, err := svc.ListAccountOptions(context.Background())
	if err != nil {
		t.Fatalf("ListAccountOptions: %v", err)
	}
	if len(options) != 1 {
		t.Fatalf("opciones = %d, esperado 1", len(options))
	}
	want := "Proveedor Demo · demo-account-001 (demo.provider)"
	if options[0].Label != want {
		t.Errorf("label = %q, esperado %q", options[0].Label, want)
	}
}

func TestAutomationCreateAccountValidation(t *testing.T) {
	svc, _ := newAutomationTestService(t)
	ctx := context.Background()
	plugin := demoPlugin

	if _, err := svc.CreateAccount(ctx, AutomationAccountInput{
		InstitutionID: 999, Identifier: "demo-account-002", PluginName: &plugin,
	}); err == nil {
		t.Error("se esperaba error por institución inexistente")
	}

	missingPlugin := "no.existe"
	if _, err := svc.CreateAccount(ctx, AutomationAccountInput{
		InstitutionID: 1, Identifier: "demo-account-002", PluginName: &missingPlugin,
	}); err == nil {
		t.Error("se esperaba error por plugin inexistente")
	}

	created, err := svc.CreateAccount(ctx, AutomationAccountInput{
		InstitutionID: 1, Identifier: "demo-account-002", PluginName: &plugin,
		ScheduleFrequency: "daily",
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if created.InstitutionName != "Proveedor Demo" {
		t.Errorf("institución = %q", created.InstitutionName)
	}

	if _, err := svc.CreateAccount(ctx, AutomationAccountInput{
		InstitutionID: 1, Identifier: "demo-account-002", PluginName: &plugin,
	}); err == nil {
		t.Error("se esperaba error por identificador duplicado")
	}
}

func TestAutomationDeleteAccountInUse(t *testing.T) {
	svc, database := newAutomationTestService(t)
	ctx := context.Background()
	plugin := demoPlugin

	account, err := svc.CreateAccount(ctx, AutomationAccountInput{
		InstitutionID: 1, Identifier: "demo-account-003", PluginName: &plugin,
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO services (home_id, name, currency_id, frequency, suggested_amount, icon_key, automation_account_id)
		VALUES (1, 'Servicio Demo', 1, 'monthly', 10, 'bolt', ?)`, account.ID); err != nil {
		t.Fatalf("insertar servicio: %v", err)
	}

	if err := svc.DeleteAccount(ctx, account.ID); !errors.Is(err, ErrAutomationAccountInUse) {
		t.Fatalf("DeleteAccount = %v, esperado ErrAutomationAccountInUse", err)
	}

	if _, err := database.Exec("DELETE FROM services WHERE automation_account_id = ?", account.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteAccount(ctx, account.ID); err != nil {
		t.Fatalf("DeleteAccount tras desvincular: %v", err)
	}
}

func TestAutomationUpdateWebhookInvalidURL(t *testing.T) {
	svc, _ := newAutomationTestService(t)
	ctx := context.Background()
	plugin := demoPlugin
	account, err := svc.CreateAccount(ctx, AutomationAccountInput{
		InstitutionID: 1, Identifier: "demo-account-004", PluginName: &plugin,
	})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}

	invalid := "ftp://no-es-http"
	if _, err := svc.UpdateAccountWebhook(ctx, account.ID, AutomationWebhookInput{
		WebhookURL: &invalid, ScheduleFrequency: "daily",
	}); err == nil {
		t.Error("se esperaba error por URL de webhook inválida")
	}

	valid := "http://demo.local/webhooks/demo"
	updated, err := svc.UpdateAccountWebhook(ctx, account.ID, AutomationWebhookInput{
		WebhookURL: &valid, ScheduleEnabled: true, ScheduleFrequency: "daily",
	})
	if err != nil {
		t.Fatalf("UpdateAccountWebhook: %v", err)
	}
	if updated.WebhookURL == nil || *updated.WebhookURL != valid {
		t.Errorf("webhook_url = %v", updated.WebhookURL)
	}
}
