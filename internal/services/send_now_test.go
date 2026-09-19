package services

import (
	"context"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// TestAlertScheduler_SendNow_NoLastKey verifica que el envío manual (SPEC-090)
// corre aunque la hora no coincida, respeta los canales deshabilitados y NO
// escribe last_alert_check.
func TestAlertScheduler_SendNow_NoLastKey(t *testing.T) {
	autoStorage := newAlertTestDB(t)
	ctx := context.Background()

	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	systemSettingsStorage := storage.NewSystemSettingsStorage(database)
	alertStorage := storage.NewAlertStorage(database)
	settingsService := NewSystemSettingsService(systemSettingsStorage)
	alertService := NewAlertService(alertStorage)
	if err := alertService.Seed(ctx); err != nil {
		t.Fatalf("seed alertas: %v", err)
	}

	// Hora de check distinta de la actual para probar que SendNow la salta.
	if err := settingsService.SetAlertCheckHour(ctx, otherHour()); err != nil {
		t.Fatalf("set hora: %v", err)
	}

	autoServiceStorage := storage.NewAutoServiceStorage(database)
	scheduler := NewAlertScheduler(
		autoStorage, autoServiceStorage,
		NewEmailService(settingsService), settingsService,
		alertService, NewVoiceMonkeyService(settingsService), nil,
	)

	res := scheduler.SendNow()

	// Sin canales habilitados → nada enviado, sin error, items recolectados.
	if len(res.SentChannels) != 0 {
		t.Fatalf("esperaba 0 canales enviados (ninguno habilitado), got %v", res.SentChannels)
	}
	if res.Items == 0 {
		t.Fatalf("esperaba autos en alerta recolectados (hay 2 en newAlertTestDB), got %d", res.Items)
	}

	last, err := settingsService.GetSetting(ctx, "last_alert_check")
	if err != nil {
		t.Fatalf("obtener último check: %v", err)
	}
	if last != nil {
		t.Fatalf("SendNow no debe escribir last_alert_check, got %+v", last)
	}
}

// TestAlertScheduler_SendNow_MailEnabled sin SMTP no bloquea y no escribe last.
func TestAlertScheduler_SendNow_MailEnabledNoSMTP(t *testing.T) {
	autoStorage := newAlertTestDB(t)
	ctx := context.Background()

	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	systemSettingsStorage := storage.NewSystemSettingsStorage(database)
	alertStorage := storage.NewAlertStorage(database)
	settingsService := NewSystemSettingsService(systemSettingsStorage)
	alertService := NewAlertService(alertStorage)
	if err := alertService.Seed(ctx); err != nil {
		t.Fatalf("seed alertas: %v", err)
	}

	mail := true
	if err := alertService.SetFlags(ctx, models.AlertKeyInsurance, &mail, nil, nil); err != nil {
		t.Fatalf("habilitar mail de seguros: %v", err)
	}

	autoServiceStorage := storage.NewAutoServiceStorage(database)
	scheduler := NewAlertScheduler(
		autoStorage, autoServiceStorage,
		NewEmailService(settingsService), settingsService,
		alertService, NewVoiceMonkeyService(settingsService), nil,
	)

	// Sin SMTP/destinatarios: el mail falla (no bloqueante) y el resumen lo refleja.
	res := scheduler.SendNow()
	if res.Detail == "" {
		t.Fatalf("esperaba detail con el error de mail, got vacío")
	}

	last, err := settingsService.GetSetting(ctx, "last_alert_check")
	if err != nil {
		t.Fatalf("obtener último check: %v", err)
	}
	if last != nil {
		t.Fatalf("SendNow no debe escribir last_alert_check, got %+v", last)
	}
}

// TestBillSummaryScheduler_SendNow_NoLastKey verifica que SendNow del resumen
// salta la hora, respeta canales y no escribe last_bill_summary_check.
func TestBillSummaryScheduler_SendNow_NoLastKey(t *testing.T) {
	billStorage, settingsService, alertService, voiceMonkeyService := newSummaryTestEnv(t, true)
	ctx := context.Background()

	if err := settingsService.SetAlertCheckHour(ctx, otherHour()); err != nil {
		t.Fatalf("set hora: %v", err)
	}

	scheduler := NewBillSummaryScheduler(billStorage, NewEmailService(settingsService), settingsService, alertService, voiceMonkeyService, nil)
	res := scheduler.SendNow()

	if res.Items != 1 {
		t.Fatalf("esperaba 1 factura pendiente, got %d", res.Items)
	}
	// Mail habilitado por defecto pero sin SMTP/destinatarios → falla, no bloquea.
	if len(res.SentChannels) != 0 {
		t.Fatalf("sin SMTP no debe haber canales enviados, got %v", res.SentChannels)
	}

	last, err := settingsService.GetSetting(ctx, "last_bill_summary_check")
	if err != nil {
		t.Fatalf("obtener último check: %v", err)
	}
	if last != nil {
		t.Fatalf("SendNow no debe escribir last_bill_summary_check, got %+v", last)
	}
}

// TestBillingScheduler_SendNowBillCreated_NoBillCreated verifica que el aviso
// de prueba no crea facturas ni toca last_billing_generation.
func TestBillingScheduler_SendNowBillCreated_NoBillCreated(t *testing.T) {
	ctx := context.Background()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("insertar dato (%q): %v", query, err)
		}
	}
	mustExec("INSERT INTO homes (name) VALUES ('Casa')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, auto_generate, is_recurring)
		VALUES (1, 'Internet', 'Claro', 1, 'monthly', 100, 1, 'internet', 'fixed', 1, 1)`)
	mustExec("INSERT OR IGNORE INTO currencies (code, name, symbol) VALUES ('NIO', 'Córdoba', 'C$')")

	serviceStorage := storage.NewServiceStorage(database)
	billStorage := storage.NewBillStorage(database)
	systemSettingsStorage := storage.NewSystemSettingsStorage(database)
	alertStorage := storage.NewAlertStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)

	settingsService := NewSystemSettingsService(systemSettingsStorage)
	alertService := NewAlertService(alertStorage)
	if err := alertService.Seed(ctx); err != nil {
		t.Fatalf("seed alertas: %v", err)
	}

	scheduler := NewBillingScheduler(
		serviceStorage, billStorage, settingsService,
		NewEmailService(settingsService), currencyStorage,
		alertService, NewVoiceMonkeyService(settingsService), nil,
	)

	res := scheduler.SendNowBillCreated()

	if res.Key != models.AlertKeyBillCreated {
		t.Fatalf("key esperada bill_created, got %s", res.Key)
	}
	if res.Items != 1 {
		t.Fatalf("aviso de prueba debe contar 1 item, got %d", res.Items)
	}

	// Sin SMTP/destinatarios el mail falla, pero igual hay items.
	last, err := settingsService.GetSetting(ctx, "last_billing_generation")
	if err != nil {
		t.Fatalf("obtener última generación: %v", err)
	}
	if last != nil {
		t.Fatalf("SendNowBillCreated no debe escribir last_billing_generation, got %+v", last)
	}

	bills, err := billStorage.ListByService(ctx, 1)
	if err != nil {
		t.Fatalf("listar facturas: %v", err)
	}
	if len(bills) != 0 {
		t.Fatalf("el aviso de prueba no debe crear facturas, got %d", len(bills))
	}
}

// otherHour devuelve una hora distinta a la actual (para probar que SendNow
// salta el gate de hora).
func otherHour() int {
	h := time.Now().Hour()
	if h == 0 {
		return 1
	}
	return h - 1
}
