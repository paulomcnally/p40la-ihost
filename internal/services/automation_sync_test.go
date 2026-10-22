package services

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost-automation-plugins/plugins"
	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type fakePlugin struct {
	bills []plugins.Bill
	err   error
}

func (f *fakePlugin) Name() string        { return "demo.provider" }
func (f *fakePlugin) Version() string     { return "1.0.0" }
func (f *fakePlugin) Source() string      { return "demo" }
func (f *fakePlugin) Description() string { return "Plugin demo" }
func (f *fakePlugin) CredentialSchema() []plugins.CredentialField {
	return []plugins.CredentialField{{Key: "user", Label: "Usuario", Type: "text", Required: true}}
}
func (f *fakePlugin) FetchBills(context.Context, map[string]string, string) ([]plugins.Bill, error) {
	return f.bills, f.err
}

type fakeRegistry struct{ plugin plugins.Plugin }

func (r fakeRegistry) Get(name string) (plugins.Plugin, bool) {
	if r.plugin != nil && name == r.plugin.Name() {
		return r.plugin, true
	}
	return nil, false
}
func (r fakeRegistry) List() []plugins.Plugin {
	if r.plugin == nil {
		return nil
	}
	return []plugins.Plugin{r.plugin}
}

const demoWebhookUUID = "00000000-0000-4000-8000-000000000000"

func newAutomationSyncEnv(t *testing.T, plugin *fakePlugin) *AutomationService {
	t.Helper()
	database, err := db.OpenDB(filepath.Join(t.TempDir(), "app.db"), "../../migrations")
	if err != nil {
		t.Fatalf("abrir DB: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if _, err := database.Exec(`
		INSERT INTO institutions (id, name) VALUES (1, 'Proveedor Demo');
		INSERT INTO automation_plugins (id, name, version, source, status)
			VALUES (1, 'demo.provider', '1.0.0', 'demo', 'active');
		INSERT INTO automation_accounts
			(id, institution_id, identifier, credentials, plugin_name, webhook_url, schedule_frequency)
			VALUES (1, 1, 'demo-account-001', '{"user":"demo"}', 'demo.provider',
			        'http://demo.local/webhooks/` + demoWebhookUUID + `', 'daily');
		INSERT INTO homes (id, name, sort_order) VALUES (1, 'Hogar Demo', 0);
		INSERT INTO services
			(id, home_id, name, currency_id, frequency, suggested_amount, icon_key, webhook_uuid, automation_account_id)
			VALUES (1, 1, 'Servicio Demo', 1, 'monthly', 10, 'bolt', '` + demoWebhookUUID + `', 1);
	`); err != nil {
		t.Fatalf("insertar fixture: %v", err)
	}

	settingsStorage := storage.NewSystemSettingsStorage(database)
	settings := NewSystemSettingsService(settingsStorage)
	if err := settings.SetWebhookEnabled(context.Background(), true); err != nil {
		t.Fatalf("habilitar webhooks: %v", err)
	}
	serviceStorage := storage.NewServiceStorage(database)
	webhook := NewWebhookService(settingsStorage, settings, serviceStorage, storage.NewBillStorage(database))

	automation := NewAutomationService(storage.NewAutomationStorage(database))
	automation.SetSyncDependencies(serviceStorage, webhook, settings, fakeRegistry{plugin: plugin})

	return automation
}

func TestAutomationSyncAccount(t *testing.T) {
	plugin := &fakePlugin{bills: []plugins.Bill{{
		Period:    "2026-09",
		Amount:    "123.45",
		Status:    "pending",
		IssueDate: "2026-09-10",
		DueDate:   "2026-10-05",
		Raw:       map[string]any{"numFactura": "F000000000000001"},
	}}}
	automation := newAutomationSyncEnv(t, plugin)

	delivered, failed, err := automation.SyncAccount(context.Background(), 1)
	if err != nil {
		t.Fatalf("SyncAccount: %v", err)
	}
	if delivered != 1 || failed != 0 {
		t.Errorf("delivered=%d failed=%d, esperado 1/0", delivered, failed)
	}

	bills, err := automation.ListBills(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListBills: %v", err)
	}
	if len(bills) != 1 || bills[0].Status != "ok" {
		t.Errorf("facturas consultadas = %+v", bills)
	}

	logs, err := automation.ListWebhookLogs(context.Background(), 1, 10, 0)
	if err != nil {
		t.Fatalf("ListWebhookLogs: %v", err)
	}
	if len(logs) != 1 || logs[0].Status != "ok" {
		t.Errorf("logs = %+v", logs)
	}
}

func TestAutomationSyncAccountPluginError(t *testing.T) {
	plugin := &fakePlugin{err: errors.New("proveedor caído")}
	automation := newAutomationSyncEnv(t, plugin)

	delivered, failed, err := automation.SyncAccount(context.Background(), 1)
	if err == nil {
		t.Fatal("se esperaba error del plugin")
	}
	if delivered != 0 || failed != 1 {
		t.Errorf("delivered=%d failed=%d, esperado 0/1", delivered, failed)
	}

	bills, err := automation.ListBills(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListBills: %v", err)
	}
	if len(bills) != 1 || bills[0].Status != "error" {
		t.Errorf("factura de error = %+v", bills)
	}
	logs, err := automation.ListWebhookLogs(context.Background(), 1, 10, 0)
	if err != nil {
		t.Fatalf("ListWebhookLogs: %v", err)
	}
	if len(logs) != 1 || logs[0].Status != "error" {
		t.Errorf("logs = %+v", logs)
	}
}

func TestAutomationSyncRequiresLinkedService(t *testing.T) {
	plugin := &fakePlugin{}
	automation := newAutomationSyncEnv(t, plugin)
	if err := automation.DeleteAccount(context.Background(), 1); !errors.Is(err, ErrAutomationAccountInUse) {
		t.Fatalf("DeleteAccount = %v, esperado ErrAutomationAccountInUse", err)
	}
}

func TestIsAutomationDue(t *testing.T) {
	monday := time.Date(2026, 10, 19, 7, 0, 0, 0, time.UTC) // lunes
	hour := "07:00"
	interval := 2
	weekday := "1"
	day15 := "15"

	daily := models.AutomationAccount{ScheduleEnabled: true, ScheduleTime: &hour, ScheduleFrequency: "daily"}
	if !isAutomationDue(daily, monday, time.Time{}) {
		t.Error("daily debería ejecutarse sin last_run")
	}
	last := monday.Add(-10 * time.Second)
	if isAutomationDue(daily, monday, last) {
		t.Error("daily no debería reintentar dentro del minuto")
	}
	daily.LastRunAt = &monday
	if isAutomationDue(daily, monday, time.Time{}) {
		t.Error("daily no debería ejecutarse dos veces el mismo día")
	}

	every := models.AutomationAccount{ScheduleEnabled: true, ScheduleTime: &hour,
		ScheduleFrequency: "every_n_days", ScheduleInterval: &interval}
	old := monday.Add(-3 * 24 * time.Hour)
	every.LastRunAt = &old
	if !isAutomationDue(every, monday, time.Time{}) {
		t.Error("every_n_days debería ejecutarse al cumplirse el intervalo")
	}

	weekly := models.AutomationAccount{ScheduleEnabled: true, ScheduleTime: &hour,
		ScheduleFrequency: "weekly", ScheduleDays: &weekday}
	if !isAutomationDue(weekly, monday, time.Time{}) {
		t.Error("weekly debería ejecutarse el lunes")
	}
	if isAutomationDue(weekly, monday.Add(24*time.Hour), time.Time{}) {
		t.Error("weekly no debería ejecutarse el martes")
	}

	monthly := models.AutomationAccount{ScheduleEnabled: true, ScheduleTime: &hour,
		ScheduleFrequency: "monthly", ScheduleDays: &day15}
	if !isAutomationDue(monthly, time.Date(2026, 10, 15, 7, 0, 0, 0, time.UTC), time.Time{}) {
		t.Error("monthly debería ejecutarse el día 15")
	}
}
