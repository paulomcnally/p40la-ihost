package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newVMSettingsService(t *testing.T) *SystemSettingsService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
}

func TestVoiceMonkey_TogglesPersistIndependently(t *testing.T) {
	s := newVMSettingsService(t)
	ctx := context.Background()

	if err := s.SetVoiceMonkeyEnabled(ctx, true); err != nil {
		t.Fatalf("set master: %v", err)
	}
	if err := s.SetVoiceMonkeySendAlerts(ctx, false); err != nil {
		t.Fatalf("set send alerts: %v", err)
	}

	// Solo cambiar un toggle no debe afectar al otro.
	if err := s.SetVoiceMonkeySendAlerts(ctx, true); err != nil {
		t.Fatalf("set send alerts (2): %v", err)
	}
	enabled, err := s.IsVoiceMonkeyEnabled(ctx)
	if err != nil || !enabled {
		t.Errorf("master debía seguir on, got %v err=%v", enabled, err)
	}
	sending, err := s.IsVoiceMonkeySendingAlerts(ctx)
	if err != nil || !sending {
		t.Errorf("send alerts debía quedar on, got %v err=%v", sending, err)
	}
}

func TestVoiceMonkey_ClearResetsAll(t *testing.T) {
	s := newVMSettingsService(t)
	ctx := context.Background()

	if err := s.SetVoiceMonkeyEnabled(ctx, true); err != nil {
		t.Fatalf("set master: %v", err)
	}
	if err := s.SetVoiceMonkeySendAlerts(ctx, true); err != nil {
		t.Fatalf("set send alerts: %v", err)
	}
	if err := s.SetVoiceMonkeyConfig(ctx, &models.VoiceMonkeyConfig{Token: "tok", Device: "dev"}); err != nil {
		t.Fatalf("set config: %v", err)
	}

	if err := s.ClearVoiceMonkey(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}

	cfg, err := s.GetVoiceMonkeyConfigPublic(ctx)
	if err != nil {
		t.Fatalf("get public: %v", err)
	}
	if cfg.Enabled || cfg.SendAlerts || cfg.Configured {
		t.Errorf("tras clear todo debe estar en false, got %+v", cfg)
	}

	internal, err := s.GetVoiceMonkeyConfig(ctx)
	if err != nil {
		t.Fatalf("get internal: %v", err)
	}
	if internal.Token != "" || internal.Device != "" {
		t.Errorf("credenciales no limpiadas: %+v", internal)
	}
}
