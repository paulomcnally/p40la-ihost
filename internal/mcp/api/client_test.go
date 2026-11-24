package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

const testAPIKey = "p40la_demo-demo-demo-demo-demo-demo-demo"

func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	client, err := New(config.Config{
		BaseURL: serverURL,
		APIKey:  testAPIKey,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("crear cliente: %v", err)
	}
	return client
}

func TestClientSendsAuthorizationAndCalls(t *testing.T) {
	var generateHeader, payHeader string
	checkAuth := func(w http.ResponseWriter, r *http.Request) bool {
		if r.Header.Get("Authorization") != "Bearer "+testAPIKey {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return false
		}
		return true
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/services/12", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		writeJSON(w, Service{ID: 12, Name: "Servicio Demo", Frequency: "monthly", Active: true})
	})
	mux.HandleFunc("GET /api/services/12/bills", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		writeJSON(w, []Bill{{ID: 1, ServiceID: 12, Year: 2026, Month: 1, Amount: 100, Status: "pending"}})
	})
	mux.HandleFunc("POST /api/services/12/bills/generate", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		generateHeader = r.Header.Get(sourceHeader)
		writeJSON(w, GenerateResult{ServiceID: 12, From: "2026-01", To: "2026-02", Created: 2})
	})
	mux.HandleFunc("POST /api/services/12/bills/pay", func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(w, r) {
			return
		}
		payHeader = r.Header.Get(sourceHeader)
		writeJSON(w, PayResult{ServiceID: 12, Paid: 1})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(t, server.URL)
	ctx := context.Background()

	service, err := client.GetService(ctx, 12)
	if err != nil {
		t.Fatalf("get service: %v", err)
	}
	if service.ID != 12 {
		t.Fatalf("service ID = %d", service.ID)
	}
	bills, err := client.ListBills(ctx, 12)
	if err != nil {
		t.Fatalf("list bills: %v", err)
	}
	if len(bills) != 1 {
		t.Fatalf("bills = %d", len(bills))
	}
	if _, err := client.GenerateBills(ctx, 12, GenerateRequest{From: "2026-01", To: "2026-02"}); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if _, err := client.PayBills(ctx, 12, PayRequest{Month: "2026-01"}); err != nil {
		t.Fatalf("pay: %v", err)
	}
	if generateHeader != sourceMCP || payHeader != sourceMCP {
		t.Fatalf("headers = generate:%q pay:%q, quiero mcp", generateHeader, payHeader)
	}
}

func TestClientAPIErrorMapping(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/services/99", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{"error": "service_not_found", "message": "el servicio no existe"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(t, server.URL)
	_, err := client.GetService(context.Background(), 99)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error inesperado: %v", err)
	}
	if apiErr.Status != http.StatusNotFound || apiErr.Code != "service_not_found" {
		t.Fatalf("error = %+v", apiErr)
	}
}

func TestClientUnauthorized(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/services/1/bills", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(t, server.URL)
	_, err := client.ListBills(context.Background(), 1)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error inesperado: %v", err)
	}
	if apiErr.Status != http.StatusUnauthorized || apiErr.Code != "auth_failed" {
		t.Fatalf("error = %+v", apiErr)
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
