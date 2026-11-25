package maintenance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) (*httptest.Server, *Service, int64, int64, int64) {
	t.Helper()
	_, database, homeID, institutionID := newTestService(t)
	module := New(database)
	mux := http.NewServeMux()
	module.RegisterRoutes(mux, func(next http.Handler) http.Handler { return next })
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, module.Service(), homeID, institutionID, currencyID(t, database, "USD")
}

func TestHandlersCreateWithItemAndListFilters(t *testing.T) {
	server, _, homeID, institutionID, usd := newTestServer(t)

	body := fmt.Sprintf(`{
		"performed_at": "2026-03-10",
		"kind": "corrective",
		"area": "plumbing",
		"institution_id": %d,
		"description": "Reparación demo",
		"item": {
			"item_type": "labor",
			"description": "Mano de obra demo",
			"amount": 50,
			"currency_id": %d,
			"institution_id": %d
		}
	}`, institutionID, usd, institutionID)
	resp, err := http.Post(fmt.Sprintf("%s/api/homes/%d/maintenances", server.URL, homeID), "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST status = %d (%s)", resp.StatusCode, payload)
	}
	var created Maintenance
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decodificar creado: %v", err)
	}
	if len(created.Items) != 1 || created.Items[0].ItemType != ItemLabor {
		t.Fatalf("líneas = %+v", created.Items)
	}
	if created.Items[0].Quantity != 1 || created.Items[0].Unit != UnitUnit {
		t.Fatalf("mano de obra no normalizada: %+v", created.Items[0])
	}

	listURL := fmt.Sprintf("%s/api/homes/%d/maintenances?date=2026-03-10&item_type=labor&q=demo&include_items=1", server.URL, homeID)
	resp, err = http.Get(listURL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET status = %d (%s)", resp.StatusCode, payload)
	}
	var list List
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decodificar listado: %v", err)
	}
	if len(list.Items) != 1 || len(list.Items[0].Items) != 1 {
		t.Fatalf("listado inesperado: %+v", list)
	}
	if list.Truncated {
		t.Fatal("no debía truncar")
	}
	total, ok := totalFor(list.TotalsByCurrency, "USD")
	if !ok || total.LaborCost != 50 {
		t.Fatalf("totales inesperados: %+v", list.TotalsByCurrency)
	}
}

func TestHandlersListRejectsBadFilters(t *testing.T) {
	server, _, homeID, _, _ := newTestServer(t)
	cases := []string{
		"date=2026-03-10&year=2026",
		"item_type=otro",
		"include_items=quizas",
		"date=10/03/2026",
	}
	for _, query := range cases {
		resp, err := http.Get(fmt.Sprintf("%s/api/homes/%d/maintenances?%s", server.URL, homeID, query))
		if err != nil {
			t.Fatalf("GET %s: %v", query, err)
		}
		payload, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s: status = %d (%s)", query, resp.StatusCode, payload)
		}
	}
}

func TestHandlersCreateWithInvalidItemDoesNotPersist(t *testing.T) {
	server, service, homeID, _, _ := newTestServer(t)
	body := `{
		"performed_at": "2026-03-10",
		"kind": "corrective",
		"area": "plumbing",
		"description": "Reparación demo",
		"item": {
			"item_type": "supplies",
			"description": "Insumo demo",
			"quantity": 1,
			"unit": "unit",
			"amount": 10,
			"currency_id": 999999
		}
	}`
	resp, err := http.Post(fmt.Sprintf("%s/api/homes/%d/maintenances", server.URL, homeID), "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d (%s)", resp.StatusCode, payload)
	}
	list, err := service.List(context.Background(), homeID, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 0 {
		t.Fatalf("quedó un mantenimiento vacío: %+v", list.Items)
	}
}

func TestHandlersCreateRejectsFutureDate(t *testing.T) {
	server, _, homeID, _, usd := newTestServer(t)
	body := fmt.Sprintf(`{
		"performed_at": "2999-01-01",
		"kind": "corrective",
		"area": "other",
		"description": "Reparación demo",
		"item": {"item_type": "labor", "description": "Mano de obra demo", "amount": 10, "currency_id": %d}
	}`, usd)
	resp, err := http.Post(fmt.Sprintf("%s/api/homes/%d/maintenances", server.URL, homeID), "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(payload), "future_date") {
		t.Fatalf("status = %d (%s)", resp.StatusCode, payload)
	}
}

func TestHandlersMoveItems(t *testing.T) {
	server, service, homeID, _, usd := newTestServer(t)
	ctx := context.Background()

	source, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-10", Kind: KindCorrective, Area: AreaOther, Description: "Reparación demo origen",
	})
	if err != nil {
		t.Fatal(err)
	}
	labor, err := service.CreateItem(ctx, homeID, source.ID, ItemInput{
		ItemType: ItemLabor, Description: "Mano de obra demo", Amount: 500, CurrencyID: usd,
	})
	if err != nil {
		t.Fatal(err)
	}
	target, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-11", Kind: KindCorrective, Area: AreaOther, Description: "Reparación demo destino",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := fmt.Sprintf(`{"item_ids":[%d],"target_maintenance_id":%d}`, labor.ID, target.ID)
	resp, err := http.Post(
		fmt.Sprintf("%s/api/homes/%d/maintenances/%d/items/move", server.URL, homeID, source.ID),
		"application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST move: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d (%s)", resp.StatusCode, payload)
	}
	var moved MoveResult
	if err := json.NewDecoder(resp.Body).Decode(&moved); err != nil {
		t.Fatalf("decodificar movimiento: %v", err)
	}
	if moved.MovedCount != 1 || moved.TargetMaintenanceID != target.ID || moved.SourceItemCount != 0 {
		t.Fatalf("movimiento inesperado: %+v", moved)
	}
	if len(moved.MovedItems) != 1 || moved.MovedItems[0].MaintenanceID != target.ID {
		t.Fatalf("líneas movidas inesperadas: %+v", moved.MovedItems)
	}
	if moved.Target == nil || moved.Target.ID != target.ID {
		t.Fatalf("destino inesperado: %+v", moved.Target)
	}

	second, err := service.CreateItem(ctx, homeID, source.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Insumo demo", Quantity: 1, Unit: UnitUnit, Amount: 20, CurrencyID: usd,
	})
	if err != nil {
		t.Fatal(err)
	}
	createTarget := fmt.Sprintf(`{
		"item_ids": [%d],
		"target": {
			"performed_at": "2026-03-12",
			"kind": "preventive",
			"area": "roof",
			"description": "Reparación demo creada al mover"
		},
		"delete_source_if_empty": true
	}`, second.ID)
	resp, err = http.Post(
		fmt.Sprintf("%s/api/homes/%d/maintenances/%d/items/move", server.URL, homeID, source.ID),
		"application/json", strings.NewReader(createTarget))
	if err != nil {
		t.Fatalf("POST move create target: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d (%s)", resp.StatusCode, payload)
	}
	var created MoveResult
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decodificar movimiento con destino nuevo: %v", err)
	}
	if !created.TargetCreated || !created.SourceDeleted || created.Target == nil || created.Target.PerformedAt != "2026-03-12" {
		t.Fatalf("movimiento con destino nuevo inesperado: %+v", created)
	}

	resp, err = http.Post(
		fmt.Sprintf("%s/api/homes/%d/maintenances/%d/items/move", server.URL, homeID, created.TargetMaintenanceID),
		"application/json", strings.NewReader(fmt.Sprintf(`{"item_ids":[%d],"target_maintenance_id":%d}`, labor.ID, created.TargetMaintenanceID)))
	if err != nil {
		t.Fatalf("POST same maintenance: %v", err)
	}
	payload, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(payload), "same_maintenance") {
		t.Fatalf("esperaba same_maintenance: %d (%s)", resp.StatusCode, payload)
	}

	current, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-10", Kind: KindCorrective, Area: AreaOther, Description: "Reparación demo actual",
	})
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-10", Kind: KindCorrective, Area: AreaOther, Description: "Reparación demo ajena",
	})
	if err != nil {
		t.Fatal(err)
	}
	foreignItem, err := service.CreateItem(ctx, homeID, foreign.ID, ItemInput{
		ItemType: ItemLabor, Description: "Mano de obra ajena", Amount: 10, CurrencyID: usd,
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.Post(
		fmt.Sprintf("%s/api/homes/%d/maintenances/%d/items/move", server.URL, homeID, current.ID),
		"application/json", strings.NewReader(fmt.Sprintf(`{"item_ids":[%d],"target_maintenance_id":%d}`, foreignItem.ID, target.ID)))
	if err != nil {
		t.Fatalf("POST foreign item: %v", err)
	}
	payload, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(payload), "item_not_in_source") {
		t.Fatalf("esperaba item_not_in_source: %d (%s)", resp.StatusCode, payload)
	}
}

func TestRespondModuleErrorMapsMoveConflict(t *testing.T) {
	recorder := httptest.NewRecorder()
	respondModuleError(recorder, fmt.Errorf("mover líneas: %w", ErrMoveConflict))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, quiero 409", recorder.Code)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "move_conflict") {
		t.Fatalf("cuerpo inesperado: %s", body)
	}
}
