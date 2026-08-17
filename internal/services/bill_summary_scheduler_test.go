package services

import (
	"context"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// newSummaryTestEnv crea una DB en memoria con homes/services de prueba y
// devuelve los storages y servicios necesarios para el scheduler.
// Si withPendingBill=true inserta una factura pendiente.
// Siembra las alertas y habilita el canal mail del resumen diario por defecto.
func newSummaryTestEnv(t *testing.T, withPendingBill bool) (*storage.BillStorage, *SystemSettingsService, *AlertService, *VoiceMonkeyService) {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("insertar dato de prueba (%q): %v", query, err)
		}
	}

	mustExec("INSERT INTO homes (name) VALUES ('Casa Test')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Internet', 'Claro', 1, 'monthly', 100, 1, 'internet', 'fixed', 1)`)
	if withPendingBill {
		mustExec("INSERT INTO bills (service_id, year, month, amount, status) VALUES (1, 2026, 8, 1500, 'pending')")
	}

	systemSettingsStorage := storage.NewSystemSettingsStorage(database)
	alertStorage := storage.NewAlertStorage(database)
	settingsService := NewSystemSettingsService(systemSettingsStorage)
	alertService := NewAlertService(alertStorage)
	voiceMonkeyService := NewVoiceMonkeyService(settingsService)

	if err := alertService.Seed(ctx); err != nil {
		t.Fatalf("seed alertas: %v", err)
	}
	mail := true
	if err := alertService.SetFlags(ctx, "bill_summary", &mail, nil); err != nil {
		t.Fatalf("habilitar mail del resumen: %v", err)
	}

	return storage.NewBillStorage(database), settingsService, alertService, voiceMonkeyService
}

func TestBillSummaryScheduler_NoPending_NoEmail(t *testing.T) {
	billStorage, settingsService, alertService, voiceMonkeyService := newSummaryTestEnv(t, false)
	scheduler := NewBillSummaryScheduler(billStorage, NewEmailService(settingsService), settingsService, alertService, voiceMonkeyService)

	// Forzar la hora actual para que el check se ejecute.
	now := time.Now()
	if err := settingsService.SetAlertCheckHour(context.Background(), now.Hour()); err != nil {
		t.Fatalf("set hora: %v", err)
	}

	scheduler.CheckNow()

	// Sin pendientes debe marcar el check del día (deduplicación).
	last, err := settingsService.GetSetting(context.Background(), "last_bill_summary_check")
	if err != nil {
		t.Fatalf("obtener último check: %v", err)
	}
	if last == nil || last.Value != now.Format("2006-01-02") {
		t.Fatalf("se esperaba check del día %s, got %+v", now.Format("2006-01-02"), last)
	}
}

func TestBillSummaryScheduler_WithPending_NoRecipients(t *testing.T) {
	billStorage, settingsService, alertService, voiceMonkeyService := newSummaryTestEnv(t, true)
	scheduler := NewBillSummaryScheduler(billStorage, NewEmailService(settingsService), settingsService, alertService, voiceMonkeyService)

	now := time.Now()
	if err := settingsService.SetAlertCheckHour(context.Background(), now.Hour()); err != nil {
		t.Fatalf("set hora: %v", err)
	}

	// Sin destinatarios configurados: el envío mail falla (no bloqueante) pero
	// el check del día se marca igualmente (deduplicación, reintento al día siguiente).
	scheduler.CheckNow()

	last, err := settingsService.GetSetting(context.Background(), "last_bill_summary_check")
	if err != nil {
		t.Fatalf("obtener último check: %v", err)
	}
	if last == nil || last.Value != now.Format("2006-01-02") {
		t.Fatalf("se esperaba check marcado pese al fallo de mail, got %+v", last)
	}
}

func TestBillSummaryScheduler_DedupSameDay(t *testing.T) {
	billStorage, settingsService, alertService, voiceMonkeyService := newSummaryTestEnv(t, false)
	scheduler := NewBillSummaryScheduler(billStorage, NewEmailService(settingsService), settingsService, alertService, voiceMonkeyService)

	now := time.Now()
	if err := settingsService.SetAlertCheckHour(context.Background(), now.Hour()); err != nil {
		t.Fatalf("set hora: %v", err)
	}

	// Primer check sin pendientes → marca el día.
	scheduler.CheckNow()

	// Segundo check el mismo día → se detecta last_bill_summary_check = hoy y no re-evalúa.
	// Se simula re-ejecutando CheckNow; el resultado debe ser el mismo (sin error).
	scheduler.CheckNow()

	last, err := settingsService.GetSetting(context.Background(), "last_bill_summary_check")
	if err != nil {
		t.Fatalf("obtener último check: %v", err)
	}
	if last == nil || last.Value != now.Format("2006-01-02") {
		t.Fatalf("deduplicación rota: %+v", last)
	}
}