package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientEntityCalls(t *testing.T) {
	mux := http.NewServeMux()
	check := func(w http.ResponseWriter, r *http.Request, method, path string) bool {
		if r.Method != method || r.URL.Path != path {
			t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, method, path)
		}
		if r.Header.Get("Authorization") != "Bearer "+testAPIKey {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return false
		}
		return true
	}
	mux.HandleFunc("GET /api/homes", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r, http.MethodGet, "/api/homes") {
			return
		}
		writeJSON(w, []Home{{ID: 7, Name: "Hogar Demo", Usage: "rental"}})
	})
	mux.HandleFunc("POST /api/homes", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r, http.MethodPost, "/api/homes") {
			return
		}
		var req HomeCreateRequest
		if err := decodeJSON(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, Home{ID: 8, Name: req.Name, Address: req.Address, Usage: req.Usage})
	})
	mux.HandleFunc("GET /api/institutions", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r, http.MethodGet, "/api/institutions") {
			return
		}
		writeJSON(w, []Institution{{ID: 9, Name: "Proveedor Demo"}})
	})
	mux.HandleFunc("GET /api/banks", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r, http.MethodGet, "/api/banks") {
			return
		}
		writeJSON(w, []Bank{{ID: 5, Name: "Banco Demo", Active: true}})
	})
	mux.HandleFunc("GET /api/currencies", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r, http.MethodGet, "/api/currencies") {
			return
		}
		writeJSON(w, []Currency{{ID: 1, Code: "NIO", Name: "Córdoba"}})
	})
	mux.HandleFunc("GET /api/institution-categories", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r, http.MethodGet, "/api/institution-categories") {
			return
		}
		writeJSON(w, []InstitutionCategory{{ID: 3, Key: "servicios", Name: "Servicios"}})
	})
	mux.HandleFunc("GET /api/services", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r, http.MethodGet, "/api/services") {
			return
		}
		if got := r.URL.Query().Get("home_id"); got != "7" {
			t.Errorf("home_id = %q, want 7", got)
		}
		writeJSON(w, []Service{{ID: 12, HomeID: 7, Name: "Internet Demo"}})
	})
	mux.HandleFunc("POST /api/services", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r, http.MethodPost, "/api/services") {
			return
		}
		var req ServiceCreateRequest
		if err := decodeJSON(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, Service{ID: 13, HomeID: req.HomeID, Name: req.Name, CurrencyID: req.CurrencyID, Active: req.Active})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(t, server.URL)
	ctx := context.Background()

	homes, err := client.ListHomes(ctx)
	if err != nil || len(homes) != 1 || homes[0].ID != 7 {
		t.Fatalf("list homes: %v %#v", err, homes)
	}
	home, err := client.CreateHome(ctx, HomeCreateRequest{Name: "Hogar Demo", Address: "Calle Demo 123", Usage: "rental"})
	if err != nil || home.ID != 8 || home.Usage != "rental" {
		t.Fatalf("create home: %v %#v", err, home)
	}
	if _, err := client.ListInstitutions(ctx); err != nil {
		t.Fatalf("list institutions: %v", err)
	}
	if _, err := client.ListBanks(ctx); err != nil {
		t.Fatalf("list banks: %v", err)
	}
	if _, err := client.ListCurrencies(ctx); err != nil {
		t.Fatalf("list currencies: %v", err)
	}
	if _, err := client.ListInstitutionCategories(ctx); err != nil {
		t.Fatalf("list categories: %v", err)
	}
	homeID := int64(7)
	services, err := client.ListServices(ctx, &homeID)
	if err != nil || len(services) != 1 || services[0].HomeID != 7 {
		t.Fatalf("list services: %v %#v", err, services)
	}
	service, err := client.CreateService(ctx, ServiceCreateRequest{HomeID: 7, Name: "Internet Demo", CurrencyID: 1, Active: true})
	if err != nil || service.ID != 13 || !service.Active {
		t.Fatalf("create service: %v %#v", err, service)
	}
}

func decodeJSON(r *http.Request, out any) error {
	return json.NewDecoder(r.Body).Decode(out)
}
