package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	client, err := New(config.Config{
		BaseURL:  serverURL,
		Email:    "demo@example.com",
		Password: "demo-password",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("crear cliente: %v", err)
	}
	return client
}

func TestClientLoginAndCalls(t *testing.T) {
	var logins atomic.Int32
	var generateHeader, payHeader string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		logins.Add(1)
		var payload struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decodificar login: %v", err)
		}
		if payload.Email != "demo@example.com" || payload.Password != "demo-password" {
			t.Errorf("credenciales inesperadas")
		}
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "demo-session", Path: "/"})
		writeJSON(w, map[string]string{"email": payload.Email})
	})
	mux.HandleFunc("GET /api/services/12", func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("session"); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeJSON(w, Service{ID: 12, Name: "Servicio Demo", Frequency: "monthly", Active: true})
	})
	mux.HandleFunc("GET /api/services/12/bills", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []Bill{{ID: 1, ServiceID: 12, Year: 2026, Month: 1, Amount: 100, Status: "pending"}})
	})
	mux.HandleFunc("POST /api/services/12/bills/generate", func(w http.ResponseWriter, r *http.Request) {
		generateHeader = r.Header.Get(sourceHeader)
		writeJSON(w, GenerateResult{ServiceID: 12, From: "2026-01", To: "2026-02", Created: 2})
	})
	mux.HandleFunc("POST /api/services/12/bills/pay", func(w http.ResponseWriter, r *http.Request) {
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
	if logins.Load() != 1 {
		t.Fatalf("logins = %d, quiero 1", logins.Load())
	}
	if generateHeader != sourceMCP || payHeader != sourceMCP {
		t.Fatalf("headers = generate:%q pay:%q, quiero mcp", generateHeader, payHeader)
	}
}

func TestClientReloginAfter401(t *testing.T) {
	var logins atomic.Int32
	var billsCalls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		logins.Add(1)
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "demo-session", Path: "/"})
		writeJSON(w, map[string]string{"email": "demo@example.com"})
	})
	mux.HandleFunc("GET /api/services/1/bills", func(w http.ResponseWriter, r *http.Request) {
		if billsCalls.Add(1) == 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeJSON(w, []Bill{})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(t, server.URL)
	if _, err := client.ListBills(context.Background(), 1); err != nil {
		t.Fatalf("list bills con relogin: %v", err)
	}
	if logins.Load() != 2 {
		t.Fatalf("logins = %d, quiero 2", logins.Load())
	}
}

func TestClientAPIErrorMapping(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "demo-session", Path: "/"})
		writeJSON(w, map[string]string{"email": "demo@example.com"})
	})
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

func TestClientSessionCookieSkipsLogin(t *testing.T) {
	var logins atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		logins.Add(1)
		writeJSON(w, map[string]string{"email": "demo@example.com"})
	})
	mux.HandleFunc("GET /api/services/1/bills", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil || cookie.Value != "demo-cookie" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeJSON(w, []Bill{})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := New(config.Config{
		BaseURL:       server.URL,
		SessionCookie: "demo-cookie",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("crear cliente: %v", err)
	}
	if _, err := client.ListBills(context.Background(), 1); err != nil {
		t.Fatalf("list bills con cookie: %v", err)
	}
	if logins.Load() != 0 {
		t.Fatalf("logins = %d, quiero 0", logins.Load())
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
