package services

import (
	"context"
	"testing"
)

func TestEmailAlerts_MasterToggle(t *testing.T) {
	s := newVMSettingsService(t)
	ctx := context.Background()

	enabled, err := s.GetEmailAlertsEnabled(ctx)
	if err != nil {
		t.Fatalf("get default: %v", err)
	}
	if enabled {
		t.Errorf("default debía ser false, got %v", enabled)
	}

	if err := s.SetEmailAlertsEnabled(ctx, true); err != nil {
		t.Fatalf("set master: %v", err)
	}
	enabled, err = s.GetEmailAlertsEnabled(ctx)
	if err != nil || !enabled {
		t.Errorf("master debía quedar on, got %v err=%v", enabled, err)
	}

	if err := s.SetEmailAlertsEnabled(ctx, false); err != nil {
		t.Fatalf("set master off: %v", err)
	}
	enabled, err = s.GetEmailAlertsEnabled(ctx)
	if err != nil || enabled {
		t.Errorf("master debía quedar off, got %v err=%v", enabled, err)
	}
}

func TestEmailAlerts_MasterToggleIndependentOfVoiceMonkey(t *testing.T) {
	s := newVMSettingsService(t)
	ctx := context.Background()

	if err := s.SetEmailAlertsEnabled(ctx, true); err != nil {
		t.Fatalf("set email master: %v", err)
	}

	vmEnabled, err := s.IsVoiceMonkeyEnabled(ctx)
	if err != nil {
		t.Fatalf("get vm: %v", err)
	}
	if vmEnabled {
		t.Errorf("voice monkey master no debía verse afectado, got %v", vmEnabled)
	}
}
