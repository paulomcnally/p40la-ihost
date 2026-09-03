package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newAlertService(t *testing.T) (*AlertService, *SystemSettingsService, *VoiceMonkeyService) {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	alertService := NewAlertService(storage.NewAlertStorage(database))
	settingsService := NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
	voiceMonkeyService := NewVoiceMonkeyService(settingsService)

	if err := alertService.Seed(context.Background()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return alertService, settingsService, voiceMonkeyService
}

func TestAlertService_Seed_IsIdempotent(t *testing.T) {
	alertService, _, _ := newAlertService(t)
	ctx := context.Background()

	if err := alertService.Seed(ctx); err != nil {
		t.Fatalf("segundo seed: %v", err)
	}

	alerts, err := alertService.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(alerts) != len(catalog()) {
		t.Fatalf("se esperaban %d alertas (catálogo), got %d", len(catalog()), len(alerts))
	}

	// Todas las claves del catálogo están presentes.
	expectedKeys := map[string]bool{}
	for _, c := range catalog() {
		expectedKeys[c.Key] = true
	}
	for _, a := range alerts {
		if !expectedKeys[a.Key] {
			t.Errorf("alerta %s no está en el catálogo", a.Key)
		}
		delete(expectedKeys, a.Key)
	}
	if len(expectedKeys) > 0 {
		t.Errorf("faltan alertas del catálogo: %v", expectedKeys)
	}

	// Todos los toggles inician en OFF (opt-in).
	for _, a := range alerts {
		if a.MailEnabled || a.VoiceEnabled {
			t.Errorf("alerta %s debía arrancar apagada, got mail=%v voice=%v", a.Key, a.MailEnabled, a.VoiceEnabled)
		}
	}
}

func TestAlertCatalog_IncludesPension(t *testing.T) {
	keys := map[string]bool{}
	for _, c := range catalog() {
		keys[c.Key] = true
	}
	for _, key := range []string{
		models.AlertKeyPensionRecordsCreated,
		models.AlertKeyPensionRecordPaid,
		models.AlertKeyPensionSalaryReceived,
		models.AlertKeyPensionRecordRejected,
		models.AlertKeyPensionMonthClosing,
	} {
		if !keys[key] {
			t.Errorf("la alerta %s debería estar en el catálogo (SPEC-051)", key)
		}
	}
}

func TestAlertService_SetFlags_Persists(t *testing.T) {
	alertService, _, _ := newAlertService(t)
	ctx := context.Background()

	mail := true
	voice := true
	if err := alertService.SetFlags(ctx, models.AlertKeyInsurance, &mail, &voice); err != nil {
		t.Fatalf("set flags: %v", err)
	}

	a, err := alertService.GetByKey(ctx, models.AlertKeyInsurance)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if a == nil {
		t.Fatal("alerta insurance no encontrada")
	}
	if !a.MailEnabled || !a.VoiceEnabled {
		t.Errorf("flags esperados true/true, got %v/%v", a.MailEnabled, a.VoiceEnabled)
	}
}

func TestAlertService_IsEnabled(t *testing.T) {
	alertService, _, _ := newAlertService(t)
	ctx := context.Background()

	// Desconocida → false.
	enabled, err := alertService.IsEnabled(ctx, "unknown", models.AlertChannelMail)
	if err != nil || enabled {
		t.Errorf("alerta desconocida debe ser false, got %v err=%v", enabled, err)
	}

	mail := true
	if err := alertService.SetFlags(ctx, models.AlertKeyBillSummary, &mail, nil); err != nil {
		t.Fatalf("set flags: %v", err)
	}

	mailEnabled, _ := alertService.IsEnabled(ctx, models.AlertKeyBillSummary, models.AlertChannelMail)
	if !mailEnabled {
		t.Error("mail debía estar habilitado")
	}
	voiceEnabled, _ := alertService.IsEnabled(ctx, models.AlertKeyBillSummary, models.AlertChannelVoice)
	if voiceEnabled {
		t.Error("voz debía seguir apagada")
	}
}

func TestAlertService_Speech(t *testing.T) {
	alertService, _, _ := newAlertService(t)
	ctx := context.Background()

	speech, err := alertService.Speech(ctx, models.AlertKeyInsurance)
	if err != nil {
		t.Fatalf("speech: %v", err)
	}
	if !contains(speech, "seguros") {
		t.Errorf("speech de insurance no menciona seguros: %q", speech)
	}
}

func TestSummarySpeech(t *testing.T) {
	if got := summarySpeech("Paulo, tienes {n} facturas pendientes.", 3); got != "Paulo, tienes 3 facturas pendientes." {
		t.Errorf("plural incorrecto: %q", got)
	}
	if got := summarySpeech("Paulo, tienes {n} facturas pendientes.", 1); got != "Paulo, tienes una factura pendiente." {
		t.Errorf("singular incorrecto: %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
