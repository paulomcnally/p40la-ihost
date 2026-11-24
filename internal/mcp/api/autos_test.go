package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientAutosCalls(t *testing.T) {
	var sourceHeaders []string
	check := func(w http.ResponseWriter, r *http.Request) bool {
		if r.Header.Get("Authorization") != "Bearer "+testAPIKey {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return false
		}
		if r.Method != http.MethodGet {
			sourceHeaders = append(sourceHeaders, r.Header.Get(sourceHeader))
		}
		return true
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/autos", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r) {
			return
		}
		writeJSON(w, []Auto{{ID: 1, Brand: "Marca Demo", Model: "Modelo Demo", Color: "white"}})
	})
	mux.HandleFunc("GET /api/autos/1", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r) {
			return
		}
		writeJSON(w, Auto{ID: 1, Brand: "Marca Demo", Model: "Modelo Demo", Color: "white"})
	})
	mux.HandleFunc("POST /api/autos", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r) {
			return
		}
		var req AutoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, Auto{ID: 2, Year: req.Year, Brand: req.Brand, Model: req.Model, Color: req.Color})
	})
	mux.HandleFunc("PUT /api/autos/1", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r) {
			return
		}
		writeJSON(w, Auto{ID: 1, Brand: "Marca Demo", Model: "Modelo Demo", Color: "black"})
	})
	mux.HandleFunc("GET /api/autos/1/available-services", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r) {
			return
		}
		writeJSON(w, []AvailableService{{ID: 12, Name: "Seguro Demo"}})
	})
	mux.HandleFunc("POST /api/autos/1/maintenances", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r) {
			return
		}
		writeJSON(w, Maintenance{ID: 7, AutoID: 1, PerformedAt: "2026-11-20", Mileage: 45000})
	})
	mux.HandleFunc("GET /api/autos/1/maintenances/7", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r) {
			return
		}
		writeJSON(w, Maintenance{ID: 7, AutoID: 1, TotalCost: 1700})
	})
	mux.HandleFunc("POST /api/autos/1/maintenances/7/items", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r) {
			return
		}
		writeJSON(w, MaintenanceItem{ID: 101, MaintenanceID: 7, ItemType: "labor", Amount: 500})
	})
	mux.HandleFunc("POST /api/autos/1/inspections", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r) {
			return
		}
		writeJSON(w, Inspection{ID: 11, AutoID: 1, InspectionYear: 2026, Result: "approved"})
	})
	mux.HandleFunc("GET /api/autos/1/inspections", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r) {
			return
		}
		writeJSON(w, InspectionList{Schedule: InspectionSchedule{NextDueYear: 2028}})
	})
	mux.HandleFunc("GET /api/autos/1/services", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r) {
			return
		}
		writeJSON(w, []InsuranceDetail{{Insurance: Insurance{ID: 21, ServiceID: 12}}})
	})
	mux.HandleFunc("POST /api/autos/1/services", func(w http.ResponseWriter, r *http.Request) {
		if !check(w, r) {
			return
		}
		writeJSON(w, Insurance{ID: 21, AutoID: 1, ServiceID: 12, CoverageType: "full_cover"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := newTestClient(t, server.URL)
	ctx := context.Background()

	if autos, err := client.ListAutos(ctx); err != nil || len(autos) != 1 {
		t.Fatalf("list autos: %v (%d)", err, len(autos))
	}
	if auto, err := client.GetAuto(ctx, 1); err != nil || auto.ID != 1 {
		t.Fatalf("get auto: %v", err)
	}
	if auto, err := client.CreateAuto(ctx, AutoRequest{Year: 2024, Brand: "Marca Demo", Model: "Modelo Demo", Color: "white"}); err != nil || auto.ID != 2 {
		t.Fatalf("create auto: %v", err)
	}
	if auto, err := client.UpdateAuto(ctx, 1, AutoRequest{Year: 2024, Brand: "Marca Demo", Model: "Modelo Demo", Color: "black"}); err != nil || auto.Color != "black" {
		t.Fatalf("update auto: %v", err)
	}
	if services, err := client.ListAvailableServices(ctx, 1); err != nil || len(services) != 1 {
		t.Fatalf("available services: %v", err)
	}
	maintenance, err := client.CreateMaintenance(ctx, 1, MaintenanceRequest{PerformedAt: "2026-11-20", Mileage: 45000})
	if err != nil || maintenance.ID != 7 {
		t.Fatalf("create maintenance: %v", err)
	}
	if detail, err := client.GetMaintenance(ctx, 1, 7); err != nil || detail.TotalCost != 1700 {
		t.Fatalf("get maintenance: %v", err)
	}
	if item, err := client.CreateMaintenanceItem(ctx, 1, 7, MaintenanceItemRequest{ItemType: "labor", Description: "demo", Amount: 500}); err != nil || item.ID != 101 {
		t.Fatalf("create item: %v", err)
	}
	if inspection, err := client.CreateInspection(ctx, 1, InspectionRequest{InspectionYear: 2026, Result: "approved"}); err != nil || inspection.ID != 11 {
		t.Fatalf("create inspection: %v", err)
	}
	if list, err := client.ListInspections(ctx, 1); err != nil || list.Schedule.NextDueYear != 2028 {
		t.Fatalf("list inspections: %v", err)
	}
	if insurances, err := client.ListInsurances(ctx, 1); err != nil || len(insurances) != 1 {
		t.Fatalf("list insurances: %v", err)
	}
	if insurance, err := client.CreateInsurance(ctx, 1, InsuranceRequest{ServiceID: 12, CoverageType: "full_cover"}); err != nil || insurance.ID != 21 {
		t.Fatalf("create insurance: %v", err)
	}

	for _, header := range sourceHeaders {
		if header != sourceMCP {
			t.Fatalf("header de origen = %q, quiero %q", header, sourceMCP)
		}
	}
	if len(sourceHeaders) != 6 {
		t.Fatalf("peticiones de escritura con header = %d, quiero 6", len(sourceHeaders))
	}
}
