package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newAPIKeyTestService(t *testing.T) *services.APIKeyService {
	t.Helper()
	database, err := db.OpenDB(filepath.Join(t.TempDir(), "test.db"), "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return services.NewAPIKeyService(storage.NewSystemSettingsStorage(database))
}

func TestAPIKeyHandlers(t *testing.T) {
	service := newAPIKeyTestService(t)
	handlers := NewAPIKeyHandlers(service)

	rec := httptest.NewRecorder()
	handlers.GetAPIKey(rec, httptest.NewRequest(http.MethodGet, "/api/api-key", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status inicial = %d", rec.Code)
	}
	var status services.APIKeyStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decodificar status: %v", err)
	}
	if status.Configured {
		t.Fatal("no debería estar configurada")
	}

	rec = httptest.NewRecorder()
	handlers.RegenerateAPIKey(rec, httptest.NewRequest(http.MethodPost, "/api/api-key/regenerate", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status regenerate = %d (%s)", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("quiero Cache-Control: no-store")
	}
	var generated struct {
		APIKey string `json:"api_key"`
		Prefix string `json:"prefix"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &generated); err != nil {
		t.Fatalf("decodificar key: %v", err)
	}
	if generated.APIKey == "" || generated.Prefix == "" {
		t.Fatalf("respuesta incompleta: %+v", generated)
	}

	rec = httptest.NewRecorder()
	handlers.GetAPIKey(rec, httptest.NewRequest(http.MethodGet, "/api/api-key", nil))
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("decodificar status configurado: %v", err)
	}
	if !status.Configured || status.Prefix != generated.Prefix {
		t.Fatalf("status configurado inesperado: %+v", status)
	}
	if rec.Body.String() == generated.APIKey {
		t.Fatal("el estado no debe exponer la key")
	}

	rec = httptest.NewRecorder()
	handlers.RevokeAPIKey(rec, httptest.NewRequest(http.MethodDelete, "/api/api-key", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status revoke = %d", rec.Code)
	}
	if valid, _ := service.Validate(context.Background(), generated.APIKey); valid {
		t.Fatal("la key revocada no debe validar")
	}
}

func TestSessionOrAPIKeyMiddleware(t *testing.T) {
	service := newAPIKeyTestService(t)
	key, _, err := service.Regenerate(context.Background())
	if err != nil {
		t.Fatalf("generar key: %v", err)
	}

	var source string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		source = requestBillSource(r)
		w.WriteHeader(http.StatusNoContent)
	})
	handler := SessionOrAPIKeyMiddleware(nil, service)(next)

	cases := []struct {
		name       string
		header     string
		value      string
		wantStatus int
		wantSource string
	}{
		{name: "bearer válido", header: "Authorization", value: "Bearer " + key, wantStatus: http.StatusNoContent, wantSource: models.BillSourceMCP},
		{name: "x-api-key válido", header: "X-API-Key", value: key, wantStatus: http.StatusNoContent, wantSource: models.BillSourceMCP},
		{name: "key inválida", header: "Authorization", value: "Bearer p40la_incorrecta", wantStatus: http.StatusUnauthorized},
		{name: "sin key", header: "", value: "", wantStatus: http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			if tc.header != "" {
				req.Header.Set(tc.header, tc.value)
			}
			req.Header.Set(bulkSourceHeader, "dashboard")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, quiero %d (%s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantSource != "" && source != tc.wantSource {
				t.Fatalf("source = %q, quiero %q", source, tc.wantSource)
			}
		})
	}

	if err := service.Revoke(context.Background()); err != nil {
		t.Fatalf("revocar: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+key)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("key revocada: status = %d, quiero 401", rec.Code)
	}
}
