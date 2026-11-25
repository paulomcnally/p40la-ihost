package homemaint

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

type fakeAPI struct {
	list    *api.HomeMaintenanceList
	listErr error

	maintenances map[int64]*api.HomeMaintenance
	getErr       error

	createMaintenanceResult *api.HomeMaintenance
	createMaintenanceErr    error
	createItemResult        *api.HomeMaintenanceItem
	createItemErr           error
	moveResult              *api.HomeMaintenanceMoveResult
	moveErr                 error

	listCalls              int
	getCalls               int
	createMaintenanceCalls int
	createItemCalls        int
	moveCalls              int
	lastFilter             api.HomeMaintenanceFilter
	lastCreateMaintenance  api.HomeMaintenanceRequest
	lastCreateItem         api.HomeMaintenanceItemRequest
	lastMove               api.HomeMaintenanceMoveRequest
	lastMoveSource         int64
}

func (f *fakeAPI) ListHomeMaintenances(_ context.Context, _ int64, filter api.HomeMaintenanceFilter) (*api.HomeMaintenanceList, error) {
	f.listCalls++
	f.lastFilter = filter
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.list == nil {
		return &api.HomeMaintenanceList{}, nil
	}
	return f.list, nil
}

func (f *fakeAPI) GetHomeMaintenance(_ context.Context, _, maintenanceID int64) (*api.HomeMaintenance, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	if maintenance, ok := f.maintenances[maintenanceID]; ok {
		return maintenance, nil
	}
	return nil, &api.APIError{Status: 404, Code: "maintenance_not_found", Message: "Mantenimiento no encontrado"}
}

func (f *fakeAPI) CreateHomeMaintenance(_ context.Context, _ int64, req api.HomeMaintenanceRequest) (*api.HomeMaintenance, error) {
	f.createMaintenanceCalls++
	f.lastCreateMaintenance = req
	if f.createMaintenanceErr != nil {
		return nil, f.createMaintenanceErr
	}
	if f.createMaintenanceResult != nil {
		return f.createMaintenanceResult, nil
	}
	return &api.HomeMaintenance{ID: 10, HomeID: 3, PerformedAt: req.PerformedAt, Area: req.Area}, nil
}

func (f *fakeAPI) CreateHomeMaintenanceItem(_ context.Context, _, maintenanceID int64, req api.HomeMaintenanceItemRequest) (*api.HomeMaintenanceItem, error) {
	f.createItemCalls++
	f.lastCreateItem = req
	if f.createItemErr != nil {
		return nil, f.createItemErr
	}
	if f.createItemResult != nil {
		return f.createItemResult, nil
	}
	return &api.HomeMaintenanceItem{ID: 99, MaintenanceID: maintenanceID, ItemType: req.ItemType, Description: req.Description, Amount: req.Amount, CurrencyID: req.CurrencyID, Quantity: req.Quantity, Unit: req.Unit}, nil
}

func (f *fakeAPI) MoveHomeMaintenanceItems(_ context.Context, _, sourceID int64, req api.HomeMaintenanceMoveRequest) (*api.HomeMaintenanceMoveResult, error) {
	f.moveCalls++
	f.lastMove = req
	f.lastMoveSource = sourceID
	if f.moveErr != nil {
		return nil, f.moveErr
	}
	if f.moveResult != nil {
		return f.moveResult, nil
	}
	targetID := int64(11)
	if req.TargetMaintenanceID != nil {
		targetID = *req.TargetMaintenanceID
	}
	moved := make([]api.HomeMaintenanceItem, 0, len(req.ItemIDs))
	for _, id := range req.ItemIDs {
		moved = append(moved, api.HomeMaintenanceItem{ID: id, MaintenanceID: targetID, ItemType: req.ItemType})
	}
	return &api.HomeMaintenanceMoveResult{
		SourceMaintenanceID: sourceID,
		TargetMaintenanceID: targetID,
		TargetCreated:       req.Target != nil,
		MovedCount:          len(moved),
		MovedItems:          moved,
		SourceItemCount:     0,
	}, nil
}

func newServer(t *testing.T, fake *fakeAPI, cfg config.Config) *mcp.ClientSession {
	t.Helper()
	if cfg.Location == nil {
		cfg.Location = time.UTC
	}
	return toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, cfg)
	})
}

func callJSON(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) map[string]any {
	t.Helper()
	text := toolkitest.Call(t, session, name, args)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("%s devolvió JSON inválido: %v (%s)", name, err, text)
	}
	return out
}

func demoMaintenance(id int64, area, date string, items ...api.HomeMaintenanceItem) api.HomeMaintenance {
	return api.HomeMaintenance{
		ID:          id,
		HomeID:      3,
		PerformedAt: date,
		Kind:        "corrective",
		Area:        area,
		Description: "Reparación demo",
		Items:       items,
	}
}

func demoItem(id, maintenanceID int64, itemType, description string, amount float64, quantity float64, unit string) api.HomeMaintenanceItem {
	return api.HomeMaintenanceItem{
		ID:            id,
		MaintenanceID: maintenanceID,
		ItemType:      itemType,
		Description:   description,
		Quantity:      quantity,
		Unit:          unit,
		Amount:        amount,
		CurrencyID:    1,
		CurrencyCode:  "NIO",
	}
}

func TestHomeMaintenanceToolsRegistered(t *testing.T) {
	session := newServer(t, &fakeAPI{}, config.Config{})
	names := toolkitest.ToolNames(t, session)
	for _, name := range []string{
		"home_maintenance_list",
		"home_maintenance_create",
		"home_maintenance_labor_register",
		"home_maintenance_supplies_register",
		"home_maintenance_item_move",
	} {
		if !names[name] {
			t.Fatalf("falta la tool %s", name)
		}
	}
}

func TestHomeMaintenanceListFilters(t *testing.T) {
	fake := &fakeAPI{list: &api.HomeMaintenanceList{
		Items: []api.HomeMaintenance{demoMaintenance(10, "plumbing", "2026-03-10",
			demoItem(101, 10, "labor", "Mano de obra demo", 500, 1, "unit"))},
		TotalsByCurrency: []api.HomeMaintenanceCurrencyTotal{
			{CurrencyID: 1, CurrencyCode: "NIO", LaborCost: 500, SuppliesCost: 200, Total: 700},
		},
	}}
	session := newServer(t, fake, config.Config{})
	out := callJSON(t, session, "home_maintenance_list", map[string]any{
		"home_id": 3, "date": "2026-03-10", "item_type": "labor", "query": "fuga", "include_items": true,
	})
	if out["count"] != float64(1) {
		t.Fatalf("count inesperado: %v", out["count"])
	}
	if out["truncated"] != false {
		t.Fatalf("truncated inesperado: %v", out["truncated"])
	}
	totals, ok := out["totals_by_currency"].([]any)
	if !ok || len(totals) != 1 || totals[0].(map[string]any)["total"] != float64(700) {
		t.Fatalf("totales inesperados: %v", out["totals_by_currency"])
	}
	if fake.lastFilter.Date != "2026-03-10" || fake.lastFilter.ItemType != "labor" || fake.lastFilter.Query != "fuga" || !fake.lastFilter.IncludeItems {
		t.Fatalf("filtros inesperados: %+v", fake.lastFilter)
	}
}

func TestHomeMaintenanceListRejectsBadFilters(t *testing.T) {
	session := newServer(t, &fakeAPI{}, config.Config{})
	cases := []map[string]any{
		{"home_id": 3, "date": "2026-03-10", "year": 2026},
		{"home_id": 3, "item_type": "otro"},
		{"home_id": 3, "query": strings.Repeat("x", 101)},
		{"home_id": 3, "date": "10/03/2026"},
	}
	for _, args := range cases {
		out := callJSON(t, session, "home_maintenance_list", args)
		if out["error"] != "invalid_request" {
			t.Fatalf("args %v: esperaba invalid_request, obtuve %v", args, out)
		}
	}
}

func TestLaborRegisterDryRunByDefault(t *testing.T) {
	fake := &fakeAPI{list: &api.HomeMaintenanceList{}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_labor_register", map[string]any{
		"home_id": 3, "performed_at": "2026-03-10", "description": "Mano de obra demo", "amount": 500, "currency_id": 1,
	})
	if out["status"] != "planned" || out["dry_run"] != true {
		t.Fatalf("esperaba planned/dry_run: %v", out)
	}
	if fake.createMaintenanceCalls != 0 || fake.createItemCalls != 0 {
		t.Fatalf("no debía escribir: create=%d item=%d", fake.createMaintenanceCalls, fake.createItemCalls)
	}
	if _, ok := out["existing_today"]; !ok {
		t.Fatalf("falta existing_today: %v", out)
	}
}

func TestLaborRegisterCreatesMaintenanceWithItem(t *testing.T) {
	fake := &fakeAPI{
		list: &api.HomeMaintenanceList{},
		createMaintenanceResult: &api.HomeMaintenance{
			ID: 10, HomeID: 3, PerformedAt: "2026-03-10", Area: "plumbing", Description: "Mano de obra demo",
			Items: []api.HomeMaintenanceItem{demoItem(101, 10, "labor", "Mano de obra demo", 500, 1, "unit")},
		},
	}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_labor_register", map[string]any{
		"home_id": 3, "performed_at": "2026-03-10", "area": "plumbing",
		"description": "Mano de obra demo", "amount": 500, "currency_id": 1, "dry_run": false,
	})
	if out["status"] != "created" || out["reused_maintenance"] != false {
		t.Fatalf("esperaba created: %v", out)
	}
	if fake.createMaintenanceCalls != 1 || fake.createItemCalls != 0 {
		t.Fatalf("llamadas inesperadas: create=%d item=%d", fake.createMaintenanceCalls, fake.createItemCalls)
	}
	if fake.lastCreateMaintenance.Item == nil {
		t.Fatal("el POST debía incluir la línea")
	}
	if fake.lastCreateMaintenance.Item.ItemType != "labor" || fake.lastCreateMaintenance.Item.Quantity != 1 || fake.lastCreateMaintenance.Item.Unit != "unit" {
		t.Fatalf("línea inesperada: %+v", fake.lastCreateMaintenance.Item)
	}
	if fake.lastCreateMaintenance.Area != "plumbing" {
		t.Fatalf("área inesperada: %q", fake.lastCreateMaintenance.Area)
	}
}

func TestLaborRegisterReusesMaintenanceAndAppends(t *testing.T) {
	fake := &fakeAPI{
		list: &api.HomeMaintenanceList{Items: []api.HomeMaintenance{
			demoMaintenance(10, "plumbing", "2026-03-10", demoItem(101, 10, "labor", "Otra tarea demo", 100, 1, "unit")),
		}},
		createItemResult: &api.HomeMaintenanceItem{ID: 102, MaintenanceID: 10, ItemType: "labor", Description: "Mano de obra demo", Amount: 500, CurrencyID: 1, Quantity: 1, Unit: "unit"},
	}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_labor_register", map[string]any{
		"home_id": 3, "performed_at": "2026-03-10", "area": "plumbing",
		"description": "Mano de obra demo", "amount": 500, "currency_id": 1, "dry_run": false,
	})
	if out["status"] != "created" || out["reused_maintenance"] != true {
		t.Fatalf("esperaba reutilización: %v", out)
	}
	if fake.createItemCalls != 1 || fake.createMaintenanceCalls != 0 {
		t.Fatalf("llamadas inesperadas: create=%d item=%d", fake.createMaintenanceCalls, fake.createItemCalls)
	}
	if fake.lastCreateItem.ItemType != "labor" {
		t.Fatalf("línea inesperada: %+v", fake.lastCreateItem)
	}
}

func TestLaborRegisterDetectsDuplicate(t *testing.T) {
	duplicate := demoItem(101, 10, "labor", "Mano de obra demo", 500, 1, "unit")
	fake := &fakeAPI{list: &api.HomeMaintenanceList{Items: []api.HomeMaintenance{
		demoMaintenance(10, "plumbing", "2026-03-10", duplicate),
	}}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_labor_register", map[string]any{
		"home_id": 3, "performed_at": "2026-03-10", "area": "plumbing",
		"description": "  Mano de obra   demo ", "amount": 500, "currency_id": 1, "dry_run": false,
	})
	if out["status"] != "duplicate" || out["dry_run"] != false {
		t.Fatalf("esperaba duplicate: %v", out)
	}
	if fake.createItemCalls != 0 || fake.createMaintenanceCalls != 0 {
		t.Fatalf("no debía escribir: create=%d item=%d", fake.createMaintenanceCalls, fake.createItemCalls)
	}
	existing, ok := out["existing"].(map[string]any)
	if !ok || existing["maintenance_id"] != float64(10) {
		t.Fatalf("existing inesperado: %v", out["existing"])
	}
}

func TestLaborRegisterAllowDuplicate(t *testing.T) {
	duplicate := demoItem(101, 10, "labor", "Mano de obra demo", 500, 1, "unit")
	fake := &fakeAPI{
		list: &api.HomeMaintenanceList{Items: []api.HomeMaintenance{
			demoMaintenance(10, "plumbing", "2026-03-10", duplicate),
		}},
		createItemResult: &api.HomeMaintenanceItem{ID: 102, MaintenanceID: 10, ItemType: "labor", Description: "Mano de obra demo", Amount: 500, CurrencyID: 1, Quantity: 1, Unit: "unit"},
	}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_labor_register", map[string]any{
		"home_id": 3, "performed_at": "2026-03-10", "area": "plumbing",
		"description": "Mano de obra demo", "amount": 500, "currency_id": 1, "dry_run": false, "allow_duplicate": true,
	})
	if out["status"] != "created" {
		t.Fatalf("esperaba created: %v", out)
	}
	if fake.createItemCalls != 1 {
		t.Fatalf("debía crear la línea: %d", fake.createItemCalls)
	}
	warnings, ok := out["warnings"].([]any)
	if !ok || len(warnings) == 0 {
		t.Fatalf("faltan warnings: %v", out)
	}
}

func TestLaborRegisterAmbiguousMaintenance(t *testing.T) {
	fake := &fakeAPI{list: &api.HomeMaintenanceList{Items: []api.HomeMaintenance{
		demoMaintenance(10, "plumbing", "2026-03-10"),
		demoMaintenance(11, "plumbing", "2026-03-10"),
	}}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_labor_register", map[string]any{
		"home_id": 3, "performed_at": "2026-03-10", "area": "plumbing",
		"description": "Mano de obra demo", "amount": 500, "currency_id": 1, "dry_run": false,
	})
	if out["status"] != "ambiguous_maintenance" {
		t.Fatalf("esperaba ambiguous_maintenance: %v", out)
	}
	candidates, ok := out["candidates"].([]any)
	if !ok || len(candidates) != 2 {
		t.Fatalf("candidatos inesperados: %v", out["candidates"])
	}
	if fake.createItemCalls != 0 || fake.createMaintenanceCalls != 0 {
		t.Fatalf("no debía escribir")
	}
}

func TestLaborRegisterWritesDisabled(t *testing.T) {
	session := newServer(t, &fakeAPI{list: &api.HomeMaintenanceList{}}, config.Config{})
	out := callJSON(t, session, "home_maintenance_labor_register", map[string]any{
		"home_id": 3, "performed_at": "2026-03-10", "description": "Mano de obra demo", "amount": 500, "currency_id": 1, "dry_run": false,
	})
	if out["error"] != "writes_disabled" {
		t.Fatalf("esperaba writes_disabled: %v", out)
	}
}

func TestLaborRegisterRejectsFutureDateAndMissingDate(t *testing.T) {
	session := newServer(t, &fakeAPI{list: &api.HomeMaintenanceList{}}, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_labor_register", map[string]any{
		"home_id": 3, "performed_at": "2999-01-01", "description": "Mano de obra demo", "amount": 500, "currency_id": 1, "dry_run": false,
	})
	if out["error"] != "invalid_date" {
		t.Fatalf("esperaba invalid_date por fecha futura: %v", out)
	}
	out = callJSON(t, session, "home_maintenance_labor_register", map[string]any{
		"home_id": 3, "description": "Mano de obra demo", "amount": 500, "currency_id": 1, "dry_run": false,
	})
	if out["error"] != "invalid_request" {
		t.Fatalf("esperaba invalid_request por fecha faltante: %v", out)
	}
}

func TestLaborRegisterMaintenanceIDDateMismatch(t *testing.T) {
	fake := &fakeAPI{maintenances: map[int64]*api.HomeMaintenance{
		10: {ID: 10, HomeID: 3, PerformedAt: "2026-03-10", Area: "plumbing"},
	}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_labor_register", map[string]any{
		"home_id": 3, "maintenance_id": 10, "performed_at": "2026-03-11",
		"description": "Mano de obra demo", "amount": 500, "currency_id": 1, "dry_run": false,
	})
	if out["error"] != "invalid_request" {
		t.Fatalf("esperaba invalid_request por fecha distinta: %v", out)
	}
}

func TestSuppliesRegisterCreatesWithQuantityAndUnit(t *testing.T) {
	fake := &fakeAPI{list: &api.HomeMaintenanceList{}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_supplies_register", map[string]any{
		"home_id": 3, "performed_at": "2026-03-10", "area": "plumbing",
		"description": "Insumos demo", "quantity": 2, "unit": "kg", "amount": 300, "currency_id": 1, "dry_run": false,
	})
	if out["status"] != "created" {
		t.Fatalf("esperaba created: %v", out)
	}
	if fake.lastCreateMaintenance.Item == nil || fake.lastCreateMaintenance.Item.ItemType != "supplies" ||
		fake.lastCreateMaintenance.Item.Quantity != 2 || fake.lastCreateMaintenance.Item.Unit != "kg" {
		t.Fatalf("línea de insumos inesperada: %+v", fake.lastCreateMaintenance.Item)
	}
}

func TestSuppliesRegisterDetectsDuplicate(t *testing.T) {
	duplicate := demoItem(101, 10, "supplies", "Insumos demo", 300, 2, "kg")
	fake := &fakeAPI{list: &api.HomeMaintenanceList{Items: []api.HomeMaintenance{
		demoMaintenance(10, "plumbing", "2026-03-10", duplicate),
	}}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_supplies_register", map[string]any{
		"home_id": 3, "performed_at": "2026-03-10", "area": "plumbing",
		"description": "Insumos demo", "quantity": 2, "unit": "kg", "amount": 300, "currency_id": 1, "dry_run": false,
	})
	if out["status"] != "duplicate" {
		t.Fatalf("esperaba duplicate: %v", out)
	}
	if fake.createMaintenanceCalls != 0 || fake.createItemCalls != 0 {
		t.Fatalf("no debía escribir")
	}
}

func TestSuppliesRegisterRejectsInvalidUnit(t *testing.T) {
	session := newServer(t, &fakeAPI{list: &api.HomeMaintenanceList{}}, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_supplies_register", map[string]any{
		"home_id": 3, "performed_at": "2026-03-10", "description": "Insumos demo",
		"quantity": 2, "unit": "caja", "amount": 300, "currency_id": 1, "dry_run": false,
	})
	if out["error"] != "invalid_request" {
		t.Fatalf("esperaba invalid_request: %v", out)
	}
}

func TestLaborRegisterAPIError(t *testing.T) {
	fake := &fakeAPI{listErr: &api.APIError{Status: 404, Code: "home_not_found", Message: "Vivienda no encontrada"}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_labor_register", map[string]any{
		"home_id": 404, "performed_at": "2026-03-10", "description": "Mano de obra demo", "amount": 500, "currency_id": 1, "dry_run": false,
	})
	if out["error"] != "home_not_found" {
		t.Fatalf("esperaba home_not_found: %v", out)
	}
}

func TestHomeMaintenanceCreateDryRunAndWrite(t *testing.T) {
	fake := &fakeAPI{createMaintenanceResult: &api.HomeMaintenance{
		ID: 12, HomeID: 3, PerformedAt: "2026-03-12", Kind: "preventive", Area: "roof",
		Description: "Revisión demo de techo", Items: []api.HomeMaintenanceItem{},
	}}
	session := newServer(t, fake, config.Config{AllowWrites: true})

	out := callJSON(t, session, "home_maintenance_create", map[string]any{
		"home_id": 3, "performed_at": "2026-03-12", "kind": "preventive", "area": "roof", "description": "Revisión demo de techo",
	})
	if out["status"] != "planned" || out["dry_run"] != true {
		t.Fatalf("esperaba planned: %v", out)
	}
	if fake.createMaintenanceCalls != 0 {
		t.Fatalf("dry_run no debía escribir")
	}

	out = callJSON(t, session, "home_maintenance_create", map[string]any{
		"home_id": 3, "performed_at": "2026-03-12", "kind": "preventive", "area": "roof", "description": "Revisión demo de techo", "dry_run": false,
	})
	if out["status"] != "created" || out["item_count"] != float64(0) {
		t.Fatalf("esperaba created con 0 líneas: %v", out)
	}
	if fake.createMaintenanceCalls != 1 || fake.lastCreateMaintenance.Item != nil {
		t.Fatalf("alta inesperada: %+v", fake.lastCreateMaintenance)
	}
}

func TestHomeMaintenanceCreateWritesDisabledAndInvalidDate(t *testing.T) {
	session := newServer(t, &fakeAPI{}, config.Config{})
	out := callJSON(t, session, "home_maintenance_create", map[string]any{
		"home_id": 3, "performed_at": "2026-03-12", "dry_run": false,
	})
	if out["error"] != "writes_disabled" {
		t.Fatalf("esperaba writes_disabled: %v", out)
	}
	out = callJSON(t, session, "home_maintenance_create", map[string]any{
		"home_id": 3, "performed_at": "2999-01-01",
	})
	if out["error"] != "invalid_date" {
		t.Fatalf("esperaba invalid_date: %v", out)
	}
	out = callJSON(t, session, "home_maintenance_create", map[string]any{
		"home_id": 3, "performed_at": "2026-03-12", "area": "otra",
	})
	if out["error"] != "invalid_request" {
		t.Fatalf("esperaba invalid_request por área: %v", out)
	}
	out = callJSON(t, session, "home_maintenance_create", map[string]any{
		"home_id": 3, "performed_at": "2026-03-12", "description": strings.Repeat("x", 501),
	})
	if out["error"] != "invalid_request" {
		t.Fatalf("esperaba invalid_request por descripción: %v", out)
	}
}

func demoMaintenancePtr(id int64, area, date string, items ...api.HomeMaintenanceItem) *api.HomeMaintenance {
	m := demoMaintenance(id, area, date, items...)
	return &m
}

func TestHomeMaintenanceMoveDryRun(t *testing.T) {
	fake := &fakeAPI{maintenances: map[int64]*api.HomeMaintenance{
		10: demoMaintenancePtr(10, "plumbing", "2026-03-10",
			demoItem(101, 10, "labor", "Mano de obra demo", 500, 1, "unit"),
			demoItem(102, 10, "supplies", "Insumos demo", 300, 2, "unit")),
		11: demoMaintenancePtr(11, "plumbing", "2026-03-11"),
	}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_item_move", map[string]any{
		"home_id": 3, "source_maintenance_id": 10, "item_ids": []any{101},
		"target_maintenance_id": 11, "delete_source_if_empty": true,
	})
	if out["status"] != "planned" || out["dry_run"] != true {
		t.Fatalf("esperaba planned: %v", out)
	}
	if fake.moveCalls != 0 {
		t.Fatalf("dry_run no debía mover")
	}
	moved, ok := out["moved_items"].([]any)
	if !ok || len(moved) != 1 {
		t.Fatalf("moved_items inesperados: %v", out["moved_items"])
	}
	remaining, ok := out["remaining_items"].([]any)
	if !ok || len(remaining) != 1 {
		t.Fatalf("remaining_items inesperados: %v", out["remaining_items"])
	}
	if out["source_item_count_after"] != float64(1) || out["would_delete_source"] != false {
		t.Fatalf("estado del origen inesperado: %v", out)
	}
	if warnings, ok := out["warnings"].([]any); !ok || len(warnings) != 1 {
		t.Fatalf("faltan warnings: %v", out)
	}
}

func TestHomeMaintenanceMoveWritesToExistingTarget(t *testing.T) {
	fake := &fakeAPI{maintenances: map[int64]*api.HomeMaintenance{
		10: demoMaintenancePtr(10, "plumbing", "2026-03-10",
			demoItem(101, 10, "labor", "Mano de obra demo", 500, 1, "unit"),
			demoItem(102, 10, "supplies", "Insumos demo", 300, 2, "unit")),
		11: demoMaintenancePtr(11, "plumbing", "2026-03-11"),
	}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_item_move", map[string]any{
		"home_id": 3, "source_maintenance_id": 10, "item_ids": []any{101, 102},
		"target_maintenance_id": 11, "dry_run": false,
	})
	if out["status"] != "moved" || out["moved_count"] != float64(2) {
		t.Fatalf("esperaba moved: %v", out)
	}
	if fake.moveCalls != 1 || fake.lastMoveSource != 10 {
		t.Fatalf("movimiento no ejecutado: %+v", fake.lastMove)
	}
	if fake.lastMove.TargetMaintenanceID == nil || *fake.lastMove.TargetMaintenanceID != 11 {
		t.Fatalf("destino inesperado: %+v", fake.lastMove)
	}
}

func TestHomeMaintenanceMoveCreatesTarget(t *testing.T) {
	fake := &fakeAPI{maintenances: map[int64]*api.HomeMaintenance{
		10: demoMaintenancePtr(10, "plumbing", "2026-03-10",
			demoItem(101, 10, "labor", "Mano de obra demo", 500, 1, "unit")),
	}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_item_move", map[string]any{
		"home_id": 3, "source_maintenance_id": 10, "item_ids": []any{101},
		"target":  map[string]any{"performed_at": "2026-03-11", "area": "plumbing", "description": "Reparación demo destino"},
		"dry_run": false,
	})
	if out["status"] != "moved" || out["target_created"] != true {
		t.Fatalf("esperaba destino creado: %v", out)
	}
	if fake.lastMove.Target == nil || fake.lastMove.Target.PerformedAt != "2026-03-11" || fake.lastMove.Target.Area != "plumbing" {
		t.Fatalf("target inesperado: %+v", fake.lastMove.Target)
	}
}

func TestHomeMaintenanceMoveRejectsInvalidSelection(t *testing.T) {
	fake := &fakeAPI{maintenances: map[int64]*api.HomeMaintenance{
		10: demoMaintenancePtr(10, "plumbing", "2026-03-10",
			demoItem(101, 10, "labor", "Mano de obra demo", 500, 1, "unit")),
		11: demoMaintenancePtr(11, "plumbing", "2026-03-11"),
	}}
	session := newServer(t, fake, config.Config{AllowWrites: true})

	out := callJSON(t, session, "home_maintenance_item_move", map[string]any{
		"home_id": 3, "source_maintenance_id": 10, "item_ids": []any{999}, "target_maintenance_id": 11, "dry_run": false,
	})
	if out["error"] != "item_not_found" {
		t.Fatalf("esperaba item_not_found: %v", out)
	}
	out = callJSON(t, session, "home_maintenance_item_move", map[string]any{
		"home_id": 3, "source_maintenance_id": 10, "item_ids": []any{101}, "item_type": "supplies", "target_maintenance_id": 11, "dry_run": false,
	})
	if out["error"] != "invalid_request" {
		t.Fatalf("esperaba invalid_request por tipo: %v", out)
	}
	out = callJSON(t, session, "home_maintenance_item_move", map[string]any{
		"home_id": 3, "source_maintenance_id": 10, "item_ids": []any{101}, "dry_run": false,
	})
	if out["error"] != "target_required" {
		t.Fatalf("esperaba target_required: %v", out)
	}
	out = callJSON(t, session, "home_maintenance_item_move", map[string]any{
		"home_id": 3, "source_maintenance_id": 10, "item_ids": []any{101},
		"target_maintenance_id": 11, "target": map[string]any{"performed_at": "2026-03-11"}, "dry_run": false,
	})
	if out["error"] != "target_conflict" {
		t.Fatalf("esperaba target_conflict: %v", out)
	}
	out = callJSON(t, session, "home_maintenance_item_move", map[string]any{
		"home_id": 3, "source_maintenance_id": 10, "item_ids": []any{101}, "target_maintenance_id": 10, "dry_run": false,
	})
	if out["error"] != "same_maintenance" {
		t.Fatalf("esperaba same_maintenance: %v", out)
	}
	if fake.moveCalls != 0 {
		t.Fatalf("no debía mover: %d", fake.moveCalls)
	}
}

func TestHomeMaintenanceMoveWritesDisabled(t *testing.T) {
	session := newServer(t, &fakeAPI{}, config.Config{})
	out := callJSON(t, session, "home_maintenance_item_move", map[string]any{
		"home_id": 3, "source_maintenance_id": 10, "item_ids": []any{101}, "target_maintenance_id": 11, "dry_run": false,
	})
	if out["error"] != "writes_disabled" {
		t.Fatalf("esperaba writes_disabled: %v", out)
	}
}
