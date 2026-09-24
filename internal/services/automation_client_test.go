package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newAutomationClientForTest(t *testing.T) (*AutomationClient, *SystemSettingsService) {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	settings := NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
	return NewAutomationClient(settings), settings
}

func TestAutomationClientNotConfigured(t *testing.T) {
	client, _ := newAutomationClientForTest(t)
	_, _, err := client.SyncAccount(context.Background(), 7)
	if !errors.Is(err, ErrAutomationNotConfigured) {
		t.Fatalf("err = %v, want ErrAutomationNotConfigured", err)
	}
}

func TestAutomationClientSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Webhook-Key") != "key-test-1234567890" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/api/accounts/7/webhook:run" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(map[string]int{"delivered": 3, "failed": 2})
	}))
	defer server.Close()

	client, settings := newAutomationClientForTest(t)
	ctx := context.Background()
	if err := settings.SetAutomationConfig(ctx, server.URL, "key-test-1234567890"); err != nil {
		t.Fatalf("config: %v", err)
	}

	delivered, failed, err := client.SyncAccount(ctx, 7)
	if err != nil {
		t.Fatalf("SyncAccount: %v", err)
	}
	if delivered != 3 || failed != 2 {
		t.Errorf("delivered/failed = %d/%d, want 3/2", delivered, failed)
	}
}

func TestAutomationClientUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized", "message": "api_key de webhook inválida"})
	}))
	defer server.Close()

	client, settings := newAutomationClientForTest(t)
	ctx := context.Background()
	if err := settings.SetAutomationConfig(ctx, server.URL, "key-incorrecta"); err != nil {
		t.Fatalf("config: %v", err)
	}

	_, _, err := client.SyncAccount(ctx, 7)
	if !errors.Is(err, ErrAutomationUnauthorized) {
		t.Fatalf("err = %v, want ErrAutomationUnauthorized", err)
	}
}

func TestAutomationClientUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close() // cortar conexión

	client, settings := newAutomationClientForTest(t)
	ctx := context.Background()
	if err := settings.SetAutomationConfig(ctx, url, "cualquier-key"); err != nil {
		t.Fatalf("config: %v", err)
	}

	_, _, err := client.SyncAccount(ctx, 7)
	if !errors.Is(err, ErrAutomationUnreachable) {
		t.Fatalf("err = %v, want ErrAutomationUnreachable", err)
	}
}

func TestSystemSettingsAutomationConfig(t *testing.T) {
	_, settings := newAutomationClientForTest(t)
	ctx := context.Background()

	// Defaults: base URL por defecto, sin api_key.
	cfg, err := settings.GetAutomationConfig(ctx)
	if err != nil {
		t.Fatalf("GetAutomationConfig: %v", err)
	}
	if cfg.BaseURL != DefaultAutomationBaseURL {
		t.Errorf("base URL default = %q, want %q", cfg.BaseURL, DefaultAutomationBaseURL)
	}
	if cfg.APIKey != "" {
		t.Errorf("api_key default = %q, want vacía", cfg.APIKey)
	}

	// Guardar base URL + api_key.
	if err := settings.SetAutomationConfig(ctx, "http://localhost:9000/", "clave-secreta"); err != nil {
		t.Fatalf("SetAutomationConfig: %v", err)
	}
	cfg, err = settings.GetAutomationConfig(ctx)
	if err != nil {
		t.Fatalf("GetAutomationConfig: %v", err)
	}
	if cfg.BaseURL != "http://localhost:9000" {
		t.Errorf("base URL = %q, want http://localhost:9000 (sin slash final)", cfg.BaseURL)
	}
	if cfg.APIKey != "clave-secreta" {
		t.Errorf("api_key = %q", cfg.APIKey)
	}

	// IsAutomationConfigured.
	ok, err := settings.IsAutomationConfigured(ctx)
	if err != nil || !ok {
		t.Errorf("IsAutomationConfigured = %v/%v, want true", ok, err)
	}

	// API key vacía no sobrescribe la existente.
	if err := settings.SetAutomationConfig(ctx, "", ""); err != nil {
		t.Fatalf("SetAutomationConfig vacío: %v", err)
	}
	cfg, _ = settings.GetAutomationConfig(ctx)
	if cfg.APIKey != "clave-secreta" {
		t.Errorf("api_key tras update vacío = %q, want clave-secreta", cfg.APIKey)
	}
}
