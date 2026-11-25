package homemaint

import (
	"context"
	"encoding/json"
	"errors"
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

	maintenances     map[int64]*api.HomeMaintenance
	getErr           error
	getErrAfterCalls int
	getErrAfter      error

	createMaintenanceResult *api.HomeMaintenance
	createMaintenanceErr    error
	createItemResult        *api.HomeMaintenanceItem
	createItemErr           error
	updateMaintenanceResult *api.HomeMaintenance
	updateMaintenanceErr    error
	updateItemResult        *api.HomeMaintenanceItem
	updateItemErr           error
	deleteItemErr           error
	moveResult              *api.HomeMaintenanceMoveResult
	moveErr                 error

	getSequence []*api.HomeMaintenance
	getIndex    int

	listCalls              int
	getCalls               int
	createMaintenanceCalls int
	createItemCalls        int
	updateMaintenanceCalls int
	updateItemCalls        int
	deleteItemCalls        int
	moveCalls              int
	lastFilter             api.HomeMaintenanceFilter
	lastCreateMaintenance  api.HomeMaintenanceRequest
	lastCreateItem         api.HomeMaintenanceItemRequest
	lastUpdateMaintenance  api.HomeMaintenanceUpdateRequest
	lastUpdateItem         api.HomeMaintenanceItemRequest
	lastDeleteMaintenance  int64
	lastDeleteItem         int64
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
	if f.getErrAfterCalls > 0 && f.getCalls > f.getErrAfterCalls && f.getErrAfter != nil {
		return nil, f.getErrAfter
	}
	if f.getErr != nil {
		return nil, f.getErr
	}
	if len(f.getSequence) > 0 {
		index := f.getIndex
		if index >= len(f.getSequence) {
			index = len(f.getSequence) - 1
		}
		f.getIndex++
		return f.getSequence[index], nil
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

func (f *fakeAPI) UpdateHomeMaintenance(_ context.Context, _, maintenanceID int64, req api.HomeMaintenanceUpdateRequest) (*api.HomeMaintenance, error) {
	f.updateMaintenanceCalls++
	f.lastUpdateMaintenance = req
	if f.updateMaintenanceErr != nil {
		return nil, f.updateMaintenanceErr
	}
	if f.updateMaintenanceResult != nil {
		return f.updateMaintenanceResult, nil
	}
	return &api.HomeMaintenance{ID: maintenanceID, PerformedAt: req.PerformedAt, Kind: req.Kind, Area: req.Area, InstitutionID: req.InstitutionID, Description: req.Description}, nil
}

func (f *fakeAPI) UpdateHomeMaintenanceItem(_ context.Context, _, maintenanceID, itemID int64, req api.HomeMaintenanceItemRequest) (*api.HomeMaintenanceItem, error) {
	f.updateItemCalls++
	f.lastUpdateItem = req
	if f.updateItemErr != nil {
		return nil, f.updateItemErr
	}
	if f.updateItemResult != nil {
		return f.updateItemResult, nil
	}
	return &api.HomeMaintenanceItem{ID: itemID, MaintenanceID: maintenanceID, ItemType: req.ItemType, Description: req.Description, Quantity: req.Quantity, Unit: req.Unit, Amount: req.Amount, CurrencyID: req.CurrencyID, InstitutionID: req.InstitutionID}, nil
}

func (f *fakeAPI) DeleteHomeMaintenanceItem(_ context.Context, _, maintenanceID, itemID int64) error {
	f.deleteItemCalls++
	f.lastDeleteMaintenance = maintenanceID
	f.lastDeleteItem = itemID
	return f.deleteItemErr
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
		"home_maintenance_get",
		"home_maintenance_create",
		"home_maintenance_labor_register",
		"home_maintenance_supplies_register",
		"home_maintenance_update",
		"home_maintenance_item_update",
		"home_maintenance_item_delete",
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

func ptrInt64(value int64) *int64 { return &value }

func demoRepair() *api.HomeMaintenance {
	labor := demoItem(101, 12, "labor", "Mano de obra demo", 500, 1, "unit")
	labor.InstitutionID = ptrInt64(7)
	supplies := demoItem(102, 12, "supplies", "Insumo demo", 300, 2, "unit")
	supplies.InstitutionID = ptrInt64(7)
	repair := demoMaintenance(12, "other", "2026-03-12", labor, supplies)
	repair.InstitutionID = ptrInt64(7)
	repair.TotalsByCurrency = []api.HomeMaintenanceCurrencyTotal{
		{CurrencyID: 1, CurrencyCode: "NIO", LaborCost: 500, SuppliesCost: 300, Total: 800},
	}
	return &repair
}

func TestHomeMaintenanceGetDetail(t *testing.T) {
	session := newServer(t, &fakeAPI{getSequence: []*api.HomeMaintenance{demoRepair()}}, config.Config{})
	out := callJSON(t, session, "home_maintenance_get", map[string]any{"home_id": 3, "maintenance_id": 12})
	if out["item_count"] != float64(2) {
		t.Fatalf("item_count inesperado: %v", out)
	}
	maintenance, ok := out["maintenance"].(map[string]any)
	if !ok || maintenance["id"] != float64(12) {
		t.Fatalf("mantenimiento inesperado: %v", out["maintenance"])
	}
	if maintenance["description"] != "Reparación demo" {
		t.Fatalf("descripción inesperada: %v", maintenance["description"])
	}
}

func TestHomeMaintenanceGetRejectsInvalidIDs(t *testing.T) {
	session := newServer(t, &fakeAPI{}, config.Config{})
	out := callJSON(t, session, "home_maintenance_get", map[string]any{"home_id": 0, "maintenance_id": 12})
	if out["error"] != "invalid_request" {
		t.Fatalf("esperaba invalid_request por home_id: %v", out)
	}
	out = callJSON(t, session, "home_maintenance_get", map[string]any{"home_id": 3, "maintenance_id": 0})
	if out["error"] != "invalid_request" {
		t.Fatalf("esperaba invalid_request por maintenance_id: %v", out)
	}
}

func TestHomeMaintenanceUpdateDryRunMerges(t *testing.T) {
	fake := &fakeAPI{getSequence: []*api.HomeMaintenance{demoRepair()}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "area": "paint",
	})
	if out["status"] != "planned" || out["dry_run"] != true {
		t.Fatalf("esperaba planned: %v", out)
	}
	changed, ok := out["changed_fields"].([]any)
	if !ok || len(changed) != 1 || changed[0] != "area" {
		t.Fatalf("changed_fields inesperado: %v", out["changed_fields"])
	}
	proposed, ok := out["proposed"].(map[string]any)
	if !ok {
		t.Fatalf("falta proposed: %v", out)
	}
	if proposed["performed_at"] != "2026-03-12" || proposed["kind"] != "corrective" ||
		proposed["institution_id"] != float64(7) || proposed["description"] != "Reparación demo" {
		t.Fatalf("el merge perdió campos actuales: %v", proposed)
	}
	if fake.updateMaintenanceCalls != 0 {
		t.Fatalf("dry_run no debía escribir")
	}
}

func TestHomeMaintenanceUpdateWritesMergedHeader(t *testing.T) {
	updated := demoRepair()
	updated.Area = "paint"
	updated.Description = "Reparación demo de pintura"
	fake := &fakeAPI{getSequence: []*api.HomeMaintenance{demoRepair()}, updateMaintenanceResult: updated}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "area": "paint",
		"description": "Reparación demo de pintura", "dry_run": false,
	})
	if out["status"] != "updated" {
		t.Fatalf("esperaba updated: %v", out)
	}
	if fake.updateMaintenanceCalls != 1 {
		t.Fatalf("esperaba un PUT: %d", fake.updateMaintenanceCalls)
	}
	request := fake.lastUpdateMaintenance
	if request.PerformedAt != "2026-03-12" || request.Kind != "corrective" || request.Area != "paint" ||
		request.InstitutionID == nil || *request.InstitutionID != 7 || request.Description != "Reparación demo de pintura" {
		t.Fatalf("PUT incompleto: %+v", request)
	}
}

func TestHomeMaintenanceUpdateNoChanges(t *testing.T) {
	fake := &fakeAPI{getSequence: []*api.HomeMaintenance{demoRepair()}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "area": "other",
	})
	if out["status"] != "unchanged" {
		t.Fatalf("esperaba unchanged: %v", out)
	}
	if fake.updateMaintenanceCalls != 0 {
		t.Fatalf("no debía escribir")
	}
}

func TestHomeMaintenanceUpdateClearsInstitutionAndDescription(t *testing.T) {
	updated := demoRepair()
	updated.InstitutionID = nil
	updated.Description = ""
	fake := &fakeAPI{getSequence: []*api.HomeMaintenance{demoRepair()}, updateMaintenanceResult: updated}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "institution_id": 0, "description": "", "dry_run": false,
	})
	if out["status"] != "updated" {
		t.Fatalf("esperaba updated: %v", out)
	}
	changed, _ := out["changed_fields"].([]any)
	if len(changed) != 2 {
		t.Fatalf("esperaba dos cambios: %v", out["changed_fields"])
	}
	if fake.lastUpdateMaintenance.InstitutionID != nil || fake.lastUpdateMaintenance.Description != "" {
		t.Fatalf("no limpió institución/descripción: %+v", fake.lastUpdateMaintenance)
	}
}

func TestHomeMaintenanceUpdateWritesDisabledAndValidation(t *testing.T) {
	session := newServer(t, &fakeAPI{}, config.Config{})
	out := callJSON(t, session, "home_maintenance_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "area": "paint", "dry_run": false,
	})
	if out["error"] != "writes_disabled" {
		t.Fatalf("esperaba writes_disabled: %v", out)
	}

	session = newServer(t, &fakeAPI{getSequence: []*api.HomeMaintenance{demoRepair()}}, config.Config{AllowWrites: true})
	out = callJSON(t, session, "home_maintenance_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "kind": "otro",
	})
	if out["error"] != "invalid_kind" {
		t.Fatalf("esperaba invalid_kind: %v", out)
	}
	out = callJSON(t, session, "home_maintenance_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "area": "otra",
	})
	if out["error"] != "invalid_area" {
		t.Fatalf("esperaba invalid_area: %v", out)
	}
	out = callJSON(t, session, "home_maintenance_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "performed_at": "2999-01-01",
	})
	if out["error"] != "invalid_date" {
		t.Fatalf("esperaba invalid_date: %v", out)
	}
	out = callJSON(t, session, "home_maintenance_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "institution_id": -1,
	})
	if out["error"] != "invalid_request" {
		t.Fatalf("esperaba invalid_request por institución: %v", out)
	}
}

func TestHomeMaintenanceItemUpdateDryRunMerges(t *testing.T) {
	fake := &fakeAPI{getSequence: []*api.HomeMaintenance{demoRepair()}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_item_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "item_id": 101, "amount": 650, "currency_id": 2,
	})
	if out["status"] != "planned" || out["dry_run"] != true {
		t.Fatalf("esperaba planned: %v", out)
	}
	changed, _ := out["changed_fields"].([]any)
	if len(changed) != 2 || changed[0] != "amount" || changed[1] != "currency_id" {
		t.Fatalf("changed_fields inesperado: %v", out["changed_fields"])
	}
	proposed, _ := out["proposed"].(map[string]any)
	if proposed["item_type"] != "labor" || proposed["description"] != "Mano de obra demo" ||
		proposed["quantity"] != float64(1) || proposed["unit"] != "unit" ||
		proposed["institution_id"] != float64(7) {
		t.Fatalf("el merge perdió campos: %v", proposed)
	}
	if fake.updateItemCalls != 0 {
		t.Fatalf("dry_run no debía escribir")
	}
}

func TestHomeMaintenanceItemUpdateWritesAndReturnsTotals(t *testing.T) {
	after := demoRepair()
	after.Items[0].Amount = 650
	after.Items[0].CurrencyID = 2
	after.Items[0].CurrencyCode = "USD"
	after.TotalsByCurrency = []api.HomeMaintenanceCurrencyTotal{
		{CurrencyID: 2, CurrencyCode: "USD", LaborCost: 650, SuppliesCost: 0, Total: 650},
	}
	fake := &fakeAPI{
		getSequence:      []*api.HomeMaintenance{demoRepair(), after},
		updateItemResult: &after.Items[0],
	}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_item_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "item_id": 101,
		"amount": 650, "currency_id": 2, "dry_run": false,
	})
	if out["status"] != "updated" {
		t.Fatalf("esperaba updated: %v", out)
	}
	if fake.updateItemCalls != 1 {
		t.Fatalf("esperaba un PUT de línea: %d", fake.updateItemCalls)
	}
	request := fake.lastUpdateItem
	if request.ItemType != "labor" || request.Description != "Mano de obra demo" ||
		request.Quantity != 1 || request.Unit != "unit" || request.Amount != 650 ||
		request.CurrencyID != 2 || request.InstitutionID == nil || *request.InstitutionID != 7 {
		t.Fatalf("PUT de línea incompleto: %+v", request)
	}
	totals, ok := out["totals_by_currency"].([]any)
	if !ok || len(totals) != 1 || totals[0].(map[string]any)["total"] != float64(650) {
		t.Fatalf("totales inesperados: %v", out["totals_by_currency"])
	}
}

func TestHomeMaintenanceItemUpdateRejectsInvalidValues(t *testing.T) {
	session := newServer(t, &fakeAPI{getSequence: []*api.HomeMaintenance{demoRepair()}}, config.Config{AllowWrites: true})
	cases := []struct {
		args map[string]any
		code string
	}{
		{map[string]any{"home_id": 3, "maintenance_id": 12, "item_id": 101, "amount": -1}, "invalid_amount"},
		{map[string]any{"home_id": 3, "maintenance_id": 12, "item_id": 101, "description": ""}, "item_description_required"},
		{map[string]any{"home_id": 3, "maintenance_id": 12, "item_id": 101, "unit": "xx"}, "invalid_unit"},
		{map[string]any{"home_id": 3, "maintenance_id": 12, "item_id": 101, "quantity": 0}, "invalid_quantity"},
	}
	for _, tc := range cases {
		out := callJSON(t, session, "home_maintenance_item_update", tc.args)
		if out["error"] != tc.code {
			t.Fatalf("esperaba %s para %v: %v", tc.code, tc.args, out)
		}
	}
	out := callJSON(t, session, "home_maintenance_item_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "item_id": 999, "amount": 10,
	})
	if out["error"] != "item_not_found" {
		t.Fatalf("esperaba item_not_found: %v", out)
	}
}

func TestHomeMaintenanceItemUpdateWritesDisabled(t *testing.T) {
	session := newServer(t, &fakeAPI{}, config.Config{})
	out := callJSON(t, session, "home_maintenance_item_update", map[string]any{
		"home_id": 3, "maintenance_id": 12, "item_id": 101, "amount": 650, "dry_run": false,
	})
	if out["error"] != "writes_disabled" {
		t.Fatalf("esperaba writes_disabled: %v", out)
	}
}

func TestHomeMaintenanceItemDeleteDryRun(t *testing.T) {
	fake := &fakeAPI{getSequence: []*api.HomeMaintenance{demoRepair()}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_item_delete", map[string]any{
		"home_id": 3, "maintenance_id": 12, "item_id": 101,
	})
	if out["status"] != "planned" || out["dry_run"] != true {
		t.Fatalf("esperaba planned: %v", out)
	}
	if out["remaining_item_count"] != float64(1) || out["maintenance_empty"] != false {
		t.Fatalf("conteo inesperado: %v", out)
	}
	wouldDelete, _ := out["would_delete"].(map[string]any)
	if wouldDelete["id"] != float64(101) {
		t.Fatalf("would_delete inesperado: %v", out["would_delete"])
	}
	if fake.deleteItemCalls != 0 {
		t.Fatalf("dry_run no debía borrar")
	}
}

func TestHomeMaintenanceItemDeleteWritesLastLine(t *testing.T) {
	single := demoRepair()
	single.Items = []api.HomeMaintenanceItem{single.Items[0]}
	empty := demoRepair()
	empty.Items = []api.HomeMaintenanceItem{}
	empty.TotalsByCurrency = []api.HomeMaintenanceCurrencyTotal{}
	fake := &fakeAPI{getSequence: []*api.HomeMaintenance{single, empty}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_item_delete", map[string]any{
		"home_id": 3, "maintenance_id": 12, "item_id": 101, "dry_run": false,
	})
	if out["status"] != "deleted" || out["maintenance_empty"] != true || out["remaining_item_count"] != float64(0) {
		t.Fatalf("esperaba deleted sin líneas: %v", out)
	}
	warnings, _ := out["warnings"].([]any)
	if len(warnings) != 1 {
		t.Fatalf("esperaba warning de cabecera: %v", out["warnings"])
	}
	if fake.deleteItemCalls != 1 || fake.lastDeleteItem != 101 || fake.lastDeleteMaintenance != 12 {
		t.Fatalf("borrado inesperado: %+v", fake)
	}
	totals, _ := out["totals_by_currency"].([]any)
	if len(totals) != 0 {
		t.Fatalf("totales inesperados: %v", totals)
	}
}

func TestHomeMaintenanceItemDeleteWritesRemaining(t *testing.T) {
	after := demoRepair()
	after.Items = []api.HomeMaintenanceItem{after.Items[1]}
	after.TotalsByCurrency = []api.HomeMaintenanceCurrencyTotal{
		{CurrencyID: 1, CurrencyCode: "NIO", LaborCost: 0, SuppliesCost: 300, Total: 300},
	}
	fake := &fakeAPI{getSequence: []*api.HomeMaintenance{demoRepair(), after}}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_item_delete", map[string]any{
		"home_id": 3, "maintenance_id": 12, "item_id": 101, "dry_run": false,
	})
	if out["status"] != "deleted" || out["remaining_item_count"] != float64(1) || out["maintenance_empty"] != false {
		t.Fatalf("esperaba una línea restante: %v", out)
	}
	totals, _ := out["totals_by_currency"].([]any)
	if len(totals) != 1 || totals[0].(map[string]any)["total"] != float64(300) {
		t.Fatalf("totales inesperados: %v", out["totals_by_currency"])
	}
}

func TestHomeMaintenanceItemDeleteKeepsDeletedWhenTotalsFail(t *testing.T) {
	fake := &fakeAPI{
		getSequence:      []*api.HomeMaintenance{demoRepair(), demoRepair()},
		getErrAfterCalls: 1,
		getErrAfter:      errors.New("upstream caído"),
	}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_maintenance_item_delete", map[string]any{
		"home_id": 3, "maintenance_id": 12, "item_id": 101, "dry_run": false,
	})
	if out["status"] != "deleted" {
		t.Fatalf("el borrado exitoso no debía fallar: %v", out)
	}
	warnings, _ := out["warnings"].([]any)
	if len(warnings) != 1 || !strings.Contains(warnings[0].(string), "no se pudieron recalcular") {
		t.Fatalf("esperaba warning de totales: %v", out["warnings"])
	}
}

func TestHomeMaintenanceItemDeleteWritesDisabledAndNotFound(t *testing.T) {
	session := newServer(t, &fakeAPI{}, config.Config{})
	out := callJSON(t, session, "home_maintenance_item_delete", map[string]any{
		"home_id": 3, "maintenance_id": 12, "item_id": 101, "dry_run": false,
	})
	if out["error"] != "writes_disabled" {
		t.Fatalf("esperaba writes_disabled: %v", out)
	}
	session = newServer(t, &fakeAPI{getSequence: []*api.HomeMaintenance{demoRepair()}}, config.Config{AllowWrites: true})
	out = callJSON(t, session, "home_maintenance_item_delete", map[string]any{
		"home_id": 3, "maintenance_id": 12, "item_id": 999,
	})
	if out["error"] != "item_not_found" {
		t.Fatalf("esperaba item_not_found: %v", out)
	}
}
