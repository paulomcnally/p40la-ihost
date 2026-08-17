package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// newVoiceMonkeyService crea un VoiceMonkeyService con config token/device set.
func newVoiceMonkeyService(t *testing.T) *VoiceMonkeyService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	settingsService := NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
	cfg := &models.VoiceMonkeyConfig{Enabled: true, SendAlerts: true, Token: "test-token", Device: "echo-show-test"}
	if err := settingsService.SetVoiceMonkeyConfig(context.Background(), cfg); err != nil {
		t.Fatalf("set config VM: %v", err)
	}
	return NewVoiceMonkeyService(settingsService)
}

func TestVoiceMonkey_Announce_Success(t *testing.T) {
	vm := newVoiceMonkeyService(t)

	var got announceRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("método esperado POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type esperado application/json, got %s", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decodificar payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"data":"ok"}`))
	}))
	defer server.Close()
	vm.url = server.URL + "/announce"

	if err := vm.Announce(context.Background(), "Paulo, prueba."); err != nil {
		t.Fatalf("Announce devolvió error: %v", err)
	}

	if got.Token != "test-token" {
		t.Errorf("token esperado test-token, got %q", got.Token)
	}
	if got.Device != "echo-show-test" {
		t.Errorf("device esperado echo-show-test, got %q", got.Device)
	}
	if got.Speech != "Paulo, prueba." {
		t.Errorf("speech incorrecto: %q", got.Speech)
	}
	if got.Voice != "Lucia" {
		t.Errorf("voz esperada Lucia, got %q", got.Voice)
	}
}

func TestVoiceMonkey_Announce_APIFailure(t *testing.T) {
	vm := newVoiceMonkeyService(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":false,"error":"device not found"}`))
	}))
	defer server.Close()
	vm.url = server.URL + "/announce"

	err := vm.Announce(context.Background(), "Prueba.")
	if err == nil {
		t.Fatal("se esperaba error cuando success=false")
	}
	if strings.Contains(err.Error(), "test-token") {
		t.Error("el error no debe contener el token")
	}
}

func TestVoiceMonkey_Announce_NotConfigured(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	defer database.Close()

	settingsService := NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
	vm := NewVoiceMonkeyService(settingsService) // sin token/device

	err = vm.Announce(context.Background(), "Prueba.")
	if err == nil {
		t.Fatal("se esperaba error de config incompleta")
	}
}
