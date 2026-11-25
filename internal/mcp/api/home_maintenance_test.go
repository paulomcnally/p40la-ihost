package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

func TestHomeMaintenanceClientListBuildsFilters(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HomeMaintenanceList{
			Items: []HomeMaintenance{{ID: 10, HomeID: 3, PerformedAt: "2026-03-10", Area: "plumbing"}},
		})
	}))
	defer server.Close()

	client, err := New(config.Config{BaseURL: server.URL, APIKey: "demo-token", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}
	list, err := client.ListHomeMaintenances(context.Background(), 3, HomeMaintenanceFilter{
		Date: "2026-03-10", ItemType: "labor", Query: "fuga", IncludeItems: true,
	})
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != 10 {
		t.Fatalf("listado inesperado: %+v", list)
	}
	for _, fragment := range []string{"date=2026-03-10", "item_type=labor", "q=fuga", "include_items=1"} {
		if !strings.Contains(gotQuery, fragment) {
			t.Fatalf("query %q no contiene %q", gotQuery, fragment)
		}
	}
}

func TestHomeMaintenanceClientCreateSendsSourceAndItem(t *testing.T) {
	var gotSource, gotPath string
	var gotBody HomeMaintenanceRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSource = r.Header.Get("X-P40LA-Source")
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decodificar body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(HomeMaintenance{
			ID: 10, HomeID: 3, PerformedAt: "2026-03-10", Area: "plumbing",
			Items: []HomeMaintenanceItem{{ID: 101, MaintenanceID: 10, ItemType: "labor", Description: "Mano de obra demo", Amount: 500, CurrencyID: 1}},
		})
	}))
	defer server.Close()

	client, err := New(config.Config{BaseURL: server.URL, APIKey: "demo-token", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}
	created, err := client.CreateHomeMaintenance(context.Background(), 3, HomeMaintenanceRequest{
		PerformedAt: "2026-03-10",
		Kind:        "corrective",
		Area:        "plumbing",
		Description: "Reparación demo",
		Item: &HomeMaintenanceItemRequest{
			ItemType: "labor", Description: "Mano de obra demo", Amount: 500, CurrencyID: 1,
		},
	})
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	if gotSource != "mcp" {
		t.Fatalf("source = %q", gotSource)
	}
	if gotPath != "/api/homes/3/maintenances" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody.Item == nil || gotBody.Item.ItemType != "labor" || gotBody.Item.Amount != 500 {
		t.Fatalf("body inesperado: %+v", gotBody)
	}
	if created.ID != 10 || len(created.Items) != 1 {
		t.Fatalf("respuesta inesperada: %+v", created)
	}
}

func TestHomeMaintenanceClientMoveSendsSourceAndTarget(t *testing.T) {
	var gotSource, gotPath string
	var gotBody HomeMaintenanceMoveRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSource = r.Header.Get("X-P40LA-Source")
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decodificar body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HomeMaintenanceMoveResult{
			SourceMaintenanceID: 10,
			TargetMaintenanceID: 11,
			MovedCount:          1,
			MovedItems: []HomeMaintenanceItem{
				{ID: 101, MaintenanceID: 11, ItemType: "labor", Description: "Mano de obra demo", Amount: 500, CurrencyID: 1},
			},
			SourceItemCount: 0,
		})
	}))
	defer server.Close()

	client, err := New(config.Config{BaseURL: server.URL, APIKey: "demo-token", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}
	target := int64(11)
	result, err := client.MoveHomeMaintenanceItems(context.Background(), 3, 10, HomeMaintenanceMoveRequest{
		ItemIDs:             []int64{101},
		ItemType:            "labor",
		TargetMaintenanceID: &target,
		DeleteSourceIfEmpty: true,
	})
	if err != nil {
		t.Fatalf("mover: %v", err)
	}
	if gotSource != "mcp" {
		t.Fatalf("source = %q", gotSource)
	}
	if gotPath != "/api/homes/3/maintenances/10/items/move" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(gotBody.ItemIDs) != 1 || gotBody.ItemIDs[0] != 101 || gotBody.ItemType != "labor" {
		t.Fatalf("body inesperado: %+v", gotBody)
	}
	if gotBody.TargetMaintenanceID == nil || *gotBody.TargetMaintenanceID != 11 || !gotBody.DeleteSourceIfEmpty {
		t.Fatalf("destino inesperado: %+v", gotBody)
	}
	if result.MovedCount != 1 || result.TargetMaintenanceID != 11 || len(result.MovedItems) != 1 {
		t.Fatalf("respuesta inesperada: %+v", result)
	}
}

func TestHomeMaintenanceClientUpdateSendsFullHeaderAndSource(t *testing.T) {
	var gotSource, gotPath, gotMethod string
	var gotBody HomeMaintenanceUpdateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSource = r.Header.Get("X-P40LA-Source")
		gotPath = r.URL.Path
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decodificar body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HomeMaintenance{
			ID: 12, HomeID: 3, PerformedAt: "2026-03-12", Kind: "corrective", Area: "paint",
			Description: "Reparación demo de pintura",
		})
	}))
	defer server.Close()

	client, err := New(config.Config{BaseURL: server.URL, APIKey: "demo-token", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}
	institutionID := int64(7)
	updated, err := client.UpdateHomeMaintenance(context.Background(), 3, 12, HomeMaintenanceUpdateRequest{
		PerformedAt:   "2026-03-12",
		Kind:          "corrective",
		Area:          "paint",
		InstitutionID: &institutionID,
		Description:   "Reparación demo de pintura",
	})
	if err != nil {
		t.Fatalf("actualizar: %v", err)
	}
	if gotSource != "mcp" || gotMethod != "PUT" || gotPath != "/api/homes/3/maintenances/12" {
		t.Fatalf("request inesperado: source=%q method=%q path=%q", gotSource, gotMethod, gotPath)
	}
	if gotBody.PerformedAt != "2026-03-12" || gotBody.Kind != "corrective" || gotBody.Area != "paint" ||
		gotBody.InstitutionID == nil || *gotBody.InstitutionID != 7 || gotBody.Description != "Reparación demo de pintura" {
		t.Fatalf("body inesperado: %+v", gotBody)
	}
	if updated.ID != 12 || updated.Area != "paint" {
		t.Fatalf("respuesta inesperada: %+v", updated)
	}
}

func TestHomeMaintenanceClientUpdateItemSendsFullItemAndSource(t *testing.T) {
	var gotSource, gotPath, gotMethod string
	var gotBody HomeMaintenanceItemRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSource = r.Header.Get("X-P40LA-Source")
		gotPath = r.URL.Path
		gotMethod = r.Method
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decodificar body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HomeMaintenanceItem{
			ID: 101, MaintenanceID: 12, ItemType: "labor", Description: "Mano de obra demo",
			Quantity: 1, Unit: "unit", Amount: 650, CurrencyID: 2, CurrencyCode: "USD",
		})
	}))
	defer server.Close()

	client, err := New(config.Config{BaseURL: server.URL, APIKey: "demo-token", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}
	institutionID := int64(7)
	updated, err := client.UpdateHomeMaintenanceItem(context.Background(), 3, 12, 101, HomeMaintenanceItemRequest{
		ItemType:      "labor",
		Description:   "Mano de obra demo",
		Quantity:      1,
		Unit:          "unit",
		Amount:        650,
		CurrencyID:    2,
		InstitutionID: &institutionID,
	})
	if err != nil {
		t.Fatalf("actualizar línea: %v", err)
	}
	if gotSource != "mcp" || gotMethod != "PUT" || gotPath != "/api/homes/3/maintenances/12/items/101" {
		t.Fatalf("request inesperado: source=%q method=%q path=%q", gotSource, gotMethod, gotPath)
	}
	if gotBody.ItemType != "labor" || gotBody.Description != "Mano de obra demo" || gotBody.Quantity != 1 ||
		gotBody.Unit != "unit" || gotBody.Amount != 650 || gotBody.CurrencyID != 2 ||
		gotBody.InstitutionID == nil || *gotBody.InstitutionID != 7 {
		t.Fatalf("body inesperado: %+v", gotBody)
	}
	if updated.ID != 101 || updated.Amount != 650 || updated.CurrencyCode != "USD" {
		t.Fatalf("respuesta inesperada: %+v", updated)
	}
}

func TestHomeMaintenanceClientDeleteItemSendsSourceAndHandlesNoContent(t *testing.T) {
	var gotSource, gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSource = r.Header.Get("X-P40LA-Source")
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(config.Config{BaseURL: server.URL, APIKey: "demo-token", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}
	if err := client.DeleteHomeMaintenanceItem(context.Background(), 3, 12, 101); err != nil {
		t.Fatalf("borrar línea: %v", err)
	}
	if gotSource != "mcp" || gotMethod != "DELETE" || gotPath != "/api/homes/3/maintenances/12/items/101" {
		t.Fatalf("request inesperado: source=%q method=%q path=%q", gotSource, gotMethod, gotPath)
	}
}
