package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func newAutomationTestDB(t *testing.T) *AutomationStorage {
	t.Helper()
	database, err := db.OpenDB(filepath.Join(t.TempDir(), "app.db"), "../../migrations")
	if err != nil {
		t.Fatalf("abrir DB: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if _, err := database.Exec(`
		INSERT INTO institutions (id, name) VALUES (1, 'Tigo');
		INSERT INTO automation_plugins (id, name, version, source, status)
			VALUES (1, 'tigo.nicaragua', '1.0.0', 'builtin', 'active');
		INSERT INTO automation_accounts
			(id, institution_id, identifier, label, credentials, plugin_name,
			 webhook_url, schedule_time, schedule_enabled, schedule_frequency)
			VALUES (7, 1, 'demo-account-004', 'Casa', '{"user":"x"}', 'tigo.nicaragua',
			        'http://ihost.local:8088/webhooks/abc', '07:00', 1, 'daily');
		INSERT INTO automation_bills
			(account_id, plugin_name, plugin_version, source, status, raw)
			VALUES (7, 'tigo.nicaragua', '1.0.0', 'plugin', 'ok', '{"amount":100}');
		INSERT INTO automation_webhook_logs
			(account_id, webhook_url, status, payload, response, http_status)
			VALUES (7, 'http://ihost.local:8088/webhooks/abc', 'ok', '{}', 'ok', 200);
	`); err != nil {
		t.Fatalf("insertar fixture: %v", err)
	}
	return NewAutomationStorage(database)
}

func TestAutomationStorageAccounts(t *testing.T) {
	st := newAutomationTestDB(t)
	accounts, err := st.ListAccounts(context.Background())
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("cuentas = %d, esperado 1", len(accounts))
	}
	account := accounts[0]
	if account.InstitutionName != "Tigo" {
		t.Errorf("institución = %q, esperado Tigo", account.InstitutionName)
	}
	if account.Identifier != "demo-account-004" {
		t.Errorf("identificador = %q", account.Identifier)
	}
	if account.PluginName == nil || *account.PluginName != "tigo.nicaragua" {
		t.Errorf("plugin = %v", account.PluginName)
	}
	if !account.ScheduleEnabled || account.ScheduleTime == nil || *account.ScheduleTime != "07:00" {
		t.Errorf("schedule = enabled:%v time:%v", account.ScheduleEnabled, account.ScheduleTime)
	}
	if account.Credentials != "" {
		t.Error("credentials no debe seleccionarse en el listado")
	}
}

func TestAutomationStorageBillsAndLogs(t *testing.T) {
	st := newAutomationTestDB(t)
	bills, err := st.ListBills(context.Background(), 7)
	if err != nil {
		t.Fatalf("ListBills: %v", err)
	}
	if len(bills) != 1 || bills[0].AccountID != 7 {
		t.Errorf("facturas = %+v", bills)
	}

	logs, err := st.ListWebhookLogs(context.Background(), 7, 20, 0)
	if err != nil {
		t.Fatalf("ListWebhookLogs: %v", err)
	}
	if len(logs) != 1 || logs[0].HTTPStatus == nil || *logs[0].HTTPStatus != 200 {
		t.Errorf("logs = %+v", logs)
	}
}

func TestAutomationStoragePlugins(t *testing.T) {
	st := newAutomationTestDB(t)
	plugins, err := st.ListPlugins(context.Background())
	if err != nil {
		t.Fatalf("ListPlugins: %v", err)
	}
	if len(plugins) != 1 || plugins[0].Name != "tigo.nicaragua" {
		t.Errorf("plugins = %+v", plugins)
	}
}

func TestAutomationStorageAccountCRUD(t *testing.T) {
	st := newAutomationTestDB(t)
	ctx := context.Background()

	plugin := "tigo.nicaragua"
	webhook := "http://demo.local/webhooks/demo"
	schedule := "07:30"
	account := &models.AutomationAccount{
		InstitutionID:     1,
		Identifier:        "demo-account-010",
		Label:             "Demo",
		PluginName:        &plugin,
		WebhookURL:        &webhook,
		ScheduleTime:      &schedule,
		ScheduleEnabled:   true,
		ScheduleFrequency: "daily",
	}
	created, err := st.CreateAccount(ctx, account, `{"user":"demo"}`)
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if created.ID == 0 || created.InstitutionName != "Tigo" {
		t.Fatalf("cuenta creada inesperada: %+v", created)
	}

	created.Label = "Demo actualizada"
	updated, err := st.UpdateAccount(ctx, created, "")
	if err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	if updated.Label != "Demo actualizada" {
		t.Errorf("label = %q", updated.Label)
	}
	credentials, err := st.CredentialsForAccount(ctx, created.ID)
	if err != nil || credentials != `{"user":"demo"}` {
		t.Errorf("credentials = %q, err = %v", credentials, err)
	}

	if _, err := st.CreateAccount(ctx, account, "{}"); err == nil {
		t.Error("se esperaba error por identificador duplicado")
	}

	if err := st.DeleteAccount(ctx, created.ID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
}
