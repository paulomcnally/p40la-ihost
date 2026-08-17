package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newSMTPSettingsService(t *testing.T) *SystemSettingsService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
}

func TestSMTP_ClearRemovesCredentials(t *testing.T) {
	s := newSMTPSettingsService(t)
	ctx := context.Background()

	cfg := &models.SMTPConfig{
		Host:      "smtp.example.com",
		Port:      587,
		User:      "user",
		Password:  "secret",
		FromEmail: "a@x.com",
		FromName:  "P40LA",
	}
	if err := s.SetSMTPConfig(ctx, cfg); err != nil {
		t.Fatalf("set smtp: %v", err)
	}
	if err := s.SetAlertEmails(ctx, []string{"alert@x.com"}); err != nil {
		t.Fatalf("set alert emails: %v", err)
	}
	if err := s.SetBillingGenerationHour(ctx, 9); err != nil {
		t.Fatalf("set billing hour: %v", err)
	}
	if err := s.SetVoiceMonkeyEnabled(ctx, true); err != nil {
		t.Fatalf("set vm enabled: %v", err)
	}

	configured, err := s.IsSMTPConfigured(ctx)
	if err != nil || !configured {
		t.Fatalf("SMTP debía estar configurado, got %v err=%v", configured, err)
	}

	if err := s.ClearSMTP(ctx); err != nil {
		t.Fatalf("clear smtp: %v", err)
	}

	configured, err = s.IsSMTPConfigured(ctx)
	if err != nil || configured {
		t.Errorf("tras clear SMTP no debe estar configurado, got %v err=%v", configured, err)
	}

	got, err := s.GetSMTPConfig(ctx)
	if err != nil {
		t.Fatalf("get smtp: %v", err)
	}
	if got.Host != "" || got.User != "" || got.Password != "" || got.FromEmail != "" || got.FromName != "" {
		t.Errorf("campos SMTP no limpiados: %+v", got)
	}
}

func TestSMTP_ClearDoesNotAffectOtherSettings(t *testing.T) {
	s := newSMTPSettingsService(t)
	ctx := context.Background()

	if err := s.SetSMTPConfig(ctx, &models.SMTPConfig{Host: "h", Port: 587, User: "u", Password: "p"}); err != nil {
		t.Fatalf("set smtp: %v", err)
	}
	if err := s.SetAlertEmails(ctx, []string{"a@x.com"}); err != nil {
		t.Fatalf("set alert emails: %v", err)
	}
	if err := s.SetBillingGenerationHour(ctx, 9); err != nil {
		t.Fatalf("set billing hour: %v", err)
	}
	if err := s.SetVoiceMonkeyEnabled(ctx, true); err != nil {
		t.Fatalf("set vm enabled: %v", err)
	}
	if err := s.SetVoiceMonkeyConfig(ctx, &models.VoiceMonkeyConfig{Token: "tok", Device: "dev"}); err != nil {
		t.Fatalf("set vm config: %v", err)
	}

	if err := s.ClearSMTP(ctx); err != nil {
		t.Fatalf("clear smtp: %v", err)
	}

	emails, err := s.GetAlertEmails(ctx)
	if err != nil || len(emails) != 1 || emails[0] != "a@x.com" {
		t.Errorf("alert emails afectados: %v err=%v", emails, err)
	}
	hour, err := s.GetBillingGenerationHour(ctx)
	if err != nil || hour != 9 {
		t.Errorf("billing hour afectado: %v err=%v", hour, err)
	}
	vmEnabled, err := s.IsVoiceMonkeyEnabled(ctx)
	if err != nil || !vmEnabled {
		t.Errorf("vm enabled afectado: %v err=%v", vmEnabled, err)
	}
	vmCfg, err := s.GetVoiceMonkeyConfig(ctx)
	if err != nil || vmCfg.Token != "tok" || vmCfg.Device != "dev" {
		t.Errorf("vm config afectada: %+v err=%v", vmCfg, err)
	}
}
