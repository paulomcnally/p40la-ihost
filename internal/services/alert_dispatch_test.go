package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func TestDispatchVoice_Disabled(t *testing.T) {
	alertService, _, vm := newAlertService(t)
	ctx := context.Background()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()
	vm.url = server.URL + "/announce"

	// Voz deshabilitada para la alerta → no se anuncia.
	mail := true
	voice := false
	if err := alertService.SetFlags(ctx, models.AlertKeyBillSummary, &mail, &voice); err != nil {
		t.Fatalf("set flags: %v", err)
	}
	dispatchVoice(ctx, alertService, vm, models.AlertKeyBillSummary, "Prueba")
	if called {
		t.Error("no debería anunciarse con voz deshabilitada")
	}
}

func TestDispatchVoice_VMMasterOff(t *testing.T) {
	alertService, _, vm := newAlertService(t)
	ctx := context.Background()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()
	vm.url = server.URL + "/announce"

	// Voz habilitada para la alerta, pero Voice Monkey desactivado.
	mail := true
	voice := true
	if err := alertService.SetFlags(ctx, models.AlertKeyInsurance, &mail, &voice); err != nil {
		t.Fatalf("set flags: %v", err)
	}
	dispatchVoice(ctx, alertService, vm, models.AlertKeyInsurance, "Prueba")
	if called {
		t.Error("no debería anunciarse con Voice Monkey desactivado")
	}
}

func TestDispatchVoice_Enabled(t *testing.T) {
	alertService, settingsService, vm := newAlertService(t)
	ctx := context.Background()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()
	vm.url = server.URL + "/announce"

	// Activar Voice Monkey: master on + enviar alertas on + token/device.
	if err := settingsService.SetVoiceMonkeyEnabled(ctx, true); err != nil {
		t.Fatalf("set master: %v", err)
	}
	if err := settingsService.SetVoiceMonkeySendAlerts(ctx, true); err != nil {
		t.Fatalf("set send alerts: %v", err)
	}
	if err := settingsService.SetVoiceMonkeyConfig(ctx, &models.VoiceMonkeyConfig{Token: "tok", Device: "dev"}); err != nil {
		t.Fatalf("set config VM: %v", err)
	}

	mail := true
	voice := true
	if err := alertService.SetFlags(ctx, models.AlertKeyInsurance, &mail, &voice); err != nil {
		t.Fatalf("set flags: %v", err)
	}

	dispatchVoice(ctx, alertService, vm, models.AlertKeyInsurance, "Prueba")
	if !called {
		t.Error("debería haberse anunciado por voz")
	}
}