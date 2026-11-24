package autos

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

type fakeAPI struct {
	autos  []api.Auto
	get    *api.Auto
	getErr error

	createReq    *api.AutoRequest
	createResult *api.Auto
	createErr    error

	updateReq    *api.AutoRequest
	updateResult *api.Auto
	updateErr    error

	available    []api.AvailableService
	availableErr error

	maintReq    *api.MaintenanceRequest
	maintResult *api.Maintenance
	maintErr    error

	maintDetail *api.Maintenance
	maintGetErr error

	itemCalls int
	itemErrAt int
	itemErr   error

	inspectionReq     *api.InspectionRequest
	inspectionResult  *api.Inspection
	inspectionErr     error
	inspectionList    *api.InspectionList
	inspectionListErr error

	insurances    []api.InsuranceDetail
	insurancesErr error
	insureReq     *api.InsuranceRequest
	insureResult  *api.Insurance
	insureErr     error
}

func (f *fakeAPI) ListAutos(context.Context) ([]api.Auto, error) { return f.autos, nil }

func (f *fakeAPI) GetAuto(_ context.Context, id int64) (*api.Auto, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.get != nil {
		return f.get, nil
	}
	return &api.Auto{ID: id, Brand: "Marca Demo", Model: "Modelo Demo"}, nil
}

func (f *fakeAPI) CreateAuto(_ context.Context, req api.AutoRequest) (*api.Auto, error) {
	f.createReq = &req
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResult != nil {
		return f.createResult, nil
	}
	return &api.Auto{ID: 5, Year: req.Year, Brand: req.Brand, Model: req.Model, Color: req.Color, Icon: req.Icon, Placa: req.Placa}, nil
}

func (f *fakeAPI) UpdateAuto(_ context.Context, _ int64, req api.AutoRequest) (*api.Auto, error) {
	f.updateReq = &req
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if f.updateResult != nil {
		return f.updateResult, nil
	}
	return &api.Auto{ID: 1, Year: req.Year, Brand: req.Brand, Model: req.Model, Color: req.Color}, nil
}

func (f *fakeAPI) ListAvailableServices(context.Context, int64) ([]api.AvailableService, error) {
	return f.available, f.availableErr
}

func (f *fakeAPI) CreateMaintenance(_ context.Context, _ int64, req api.MaintenanceRequest) (*api.Maintenance, error) {
	f.maintReq = &req
	if f.maintErr != nil {
		return nil, f.maintErr
	}
	if f.maintResult != nil {
		return f.maintResult, nil
	}
	return &api.Maintenance{ID: 7, AutoID: 1, PerformedAt: req.PerformedAt, Mileage: req.Mileage, Description: req.Description}, nil
}

func (f *fakeAPI) GetMaintenance(context.Context, int64, int64) (*api.Maintenance, error) {
	if f.maintGetErr != nil {
		return nil, f.maintGetErr
	}
	if f.maintDetail != nil {
		return f.maintDetail, nil
	}
	return &api.Maintenance{ID: 7, AutoID: 1, TotalCost: 0, Items: []api.MaintenanceItem{}}, nil
}

func (f *fakeAPI) CreateMaintenanceItem(_ context.Context, _, maintenanceID int64, req api.MaintenanceItemRequest) (*api.MaintenanceItem, error) {
	f.itemCalls++
	if f.itemErr != nil && f.itemCalls == f.itemErrAt {
		return nil, f.itemErr
	}
	return &api.MaintenanceItem{ID: int64(100 + f.itemCalls), MaintenanceID: maintenanceID, ItemType: req.ItemType, Description: req.Description, Amount: req.Amount}, nil
}

func (f *fakeAPI) CreateInspection(_ context.Context, _ int64, req api.InspectionRequest) (*api.Inspection, error) {
	f.inspectionReq = &req
	if f.inspectionErr != nil {
		return nil, f.inspectionErr
	}
	if f.inspectionResult != nil {
		return f.inspectionResult, nil
	}
	return &api.Inspection{ID: 11, AutoID: 1, PerformedAt: req.PerformedAt, InspectionYear: req.InspectionYear, Result: req.Result}, nil
}

func (f *fakeAPI) ListInspections(context.Context, int64) (*api.InspectionList, error) {
	return f.inspectionList, f.inspectionListErr
}

func (f *fakeAPI) ListInsurances(context.Context, int64) ([]api.InsuranceDetail, error) {
	return f.insurances, f.insurancesErr
}

func (f *fakeAPI) CreateInsurance(_ context.Context, _ int64, req api.InsuranceRequest) (*api.Insurance, error) {
	f.insureReq = &req
	if f.insureErr != nil {
		return nil, f.insureErr
	}
	if f.insureResult != nil {
		return f.insureResult, nil
	}
	return &api.Insurance{ID: 21, AutoID: 1, ServiceID: req.ServiceID, CoverageType: req.CoverageType, PolicyNumber: req.PolicyNumber}, nil
}

func fixedNow() time.Time {
	return time.Date(2026, time.November, 24, 15, 0, 0, 0, time.UTC)
}

func newToolset(fake *fakeAPI, allowWrites bool) *Toolset {
	toolset := New(fake, config.Config{AllowWrites: allowWrites, Timezone: "UTC", Location: time.UTC})
	toolset.now = fixedNow
	return toolset
}

func boolPtr(value bool) *bool    { return &value }
func strPtr(value string) *string { return &value }

func TestListFiltersByQueryAndID(t *testing.T) {
	fake := &fakeAPI{autos: []api.Auto{
		{ID: 1, Brand: "Marca Demo", Model: "Camioneta", Placa: "DEMO-001"},
		{ID: 2, Brand: "Otra Marca", Model: "Sedán", Placa: "DEMO-002"},
	}}
	toolset := newToolset(fake, false)

	_, output, err := toolset.list(context.Background(), nil, ListInput{Query: strPtr("sedan")})
	if err != nil {
		t.Fatalf("list devolvió error: %v", err)
	}
	list := output.(ListOutput)
	if list.Count != 1 || list.Autos[0].ID != 2 {
		t.Fatalf("filtro por texto inesperado: %+v", list)
	}

	_, output, err = toolset.list(context.Background(), nil, ListInput{AutoID: int64Ptr(1)})
	if err != nil {
		t.Fatalf("list por ID devolvió error: %v", err)
	}
	if got := output.(ListOutput); got.Count != 1 || got.Autos[0].ID != 1 {
		t.Fatalf("filtro por ID inesperado: %+v", got)
	}
}

func int64Ptr(value int64) *int64 { return &value }

func TestCreateDryRunDoesNotWrite(t *testing.T) {
	fake := &fakeAPI{}
	toolset := newToolset(fake, true)

	_, output, err := toolset.create(context.Background(), nil, CreateInput{
		Year: 2024, Brand: "Marca Demo", Model: "Modelo Demo", Color: "white",
	})
	if err != nil {
		t.Fatalf("create devolvió error: %v", err)
	}
	result := output.(WriteOutput)
	if result.Status != "planned" || !result.DryRun {
		t.Fatalf("esperaba planned: %+v", result)
	}
	if fake.createReq != nil {
		t.Fatal("dry_run no debe llamar CreateAuto")
	}
}

func TestCreateRequiresWriteGate(t *testing.T) {
	fake := &fakeAPI{}
	toolset := newToolset(fake, false)

	result, output, err := toolset.create(context.Background(), nil, CreateInput{
		Year: 2024, Brand: "Marca Demo", Model: "Modelo Demo", Color: "white", DryRun: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("create devolvió error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("esperaba resultado de error")
	}
	payload := output.(map[string]any)
	if payload["error"] != "writes_disabled" {
		t.Fatalf("error inesperado: %+v", payload)
	}
	if fake.createReq != nil {
		t.Fatal("no debe escribir sin compuerta")
	}
}

func TestCreateWrites(t *testing.T) {
	fake := &fakeAPI{}
	toolset := newToolset(fake, true)

	_, output, err := toolset.create(context.Background(), nil, CreateInput{
		Year: 2024, Brand: "Marca Demo", Model: "Modelo Demo", Color: "white", DryRun: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("create devolvió error: %v", err)
	}
	result := output.(WriteOutput)
	if result.Status != "created" || result.Auto == nil || result.Auto.ID != 5 {
		t.Fatalf("alta inesperada: %+v", result)
	}
}

func TestUpdateDryRunChecksAuto(t *testing.T) {
	fake := &fakeAPI{getErr: &api.APIError{Status: 404, Code: "not_found", Message: "Auto no encontrado"}}
	toolset := newToolset(fake, true)

	result, output, err := toolset.update(context.Background(), nil, UpdateInput{
		AutoID: 99, Year: 2024, Brand: "Marca Demo", Model: "Modelo Demo", Color: "black",
	})
	if err != nil {
		t.Fatalf("update devolvió error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("esperaba error not_found")
	}
	if output.(map[string]any)["error"] != "not_found" {
		t.Fatalf("error inesperado: %+v", output)
	}
}

func TestMaintenanceDryRunValidatesDate(t *testing.T) {
	toolset := newToolset(&fakeAPI{}, true)

	result, output, err := toolset.maintenanceRegister(context.Background(), nil, MaintenanceInput{
		AutoID: 1, PerformedAt: "2026-12-01", Mileage: 1000, InstitutionID: 2, CurrencyID: 1,
	})
	if err != nil {
		t.Fatalf("maintenance devolvió error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("esperaba error de fecha futura")
	}
	if output.(map[string]any)["error"] != "invalid_date" {
		t.Fatalf("error inesperado: %+v", output)
	}
}

func TestMaintenanceCreatesItems(t *testing.T) {
	fake := &fakeAPI{
		maintDetail: &api.Maintenance{ID: 7, TotalCost: 1700, Items: []api.MaintenanceItem{{ID: 101}, {ID: 102}}},
	}
	toolset := newToolset(fake, true)

	_, output, err := toolset.maintenanceRegister(context.Background(), nil, MaintenanceInput{
		AutoID: 1, PerformedAt: "2026-11-20", Mileage: 45000, InstitutionID: 7, CurrencyID: 1,
		Items: []MaintenanceItemInput{
			{ItemType: "labor", Description: "Mano de obra demo", Amount: 500},
			{ItemType: "supplies", Description: "Insumos demo", Amount: 1200},
		},
		DryRun: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("maintenance devolvió error: %v", err)
	}
	result := output.(WriteOutput)
	if result.Status != "created" || result.Partial {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if fake.itemCalls != 2 {
		t.Fatalf("esperaba 2 ítems, obtuve %d", fake.itemCalls)
	}
	if result.Maintenance == nil || result.Maintenance.TotalCost != 1700 {
		t.Fatalf("esperaba el detalle recalculado: %+v", result.Maintenance)
	}
}

func TestMaintenancePartialWhenItemFails(t *testing.T) {
	fake := &fakeAPI{itemErrAt: 2, itemErr: errors.New("falló la línea demo")}
	toolset := newToolset(fake, true)

	_, output, err := toolset.maintenanceRegister(context.Background(), nil, MaintenanceInput{
		AutoID: 1, PerformedAt: "2026-11-20", Mileage: 45000, InstitutionID: 7, CurrencyID: 1,
		Items: []MaintenanceItemInput{
			{ItemType: "labor", Description: "Mano de obra demo", Amount: 500},
			{ItemType: "supplies", Description: "Insumos demo", Amount: 1200},
		},
		DryRun: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("maintenance devolvió error: %v", err)
	}
	result := output.(WriteOutput)
	if !result.Partial || result.Status != "created_partial" {
		t.Fatalf("esperaba parcial: %+v", result)
	}
	if len(result.Items) != 1 || len(result.Warnings) != 1 {
		t.Fatalf("esperaba un ítem y una advertencia: %+v", result)
	}
}

func TestInspectionReturnsSchedule(t *testing.T) {
	fake := &fakeAPI{
		inspectionList: &api.InspectionList{Schedule: api.InspectionSchedule{NextDueYear: 2028, Status: "ok"}},
	}
	toolset := newToolset(fake, true)

	_, output, err := toolset.inspectionRegister(context.Background(), nil, InspectionInput{
		AutoID: 1, PerformedAt: "2026-11-20", InspectionYear: 2026, InstitutionID: 9, Result: "approved",
		DryRun: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("inspection devolvió error: %v", err)
	}
	result := output.(WriteOutput)
	if result.Status != "created" || result.Schedule == nil || result.Schedule.NextDueYear != 2028 {
		t.Fatalf("esperaba cronograma: %+v", result)
	}
}

func TestInsurancePrerequisiteMissing(t *testing.T) {
	fake := &fakeAPI{available: []api.AvailableService{{ID: 12, Name: "Seguro Demo"}}}
	toolset := newToolset(fake, true)

	result, output, err := toolset.insuranceRegister(context.Background(), nil, InsuranceInput{
		AutoID: 1, ServiceID: 99, CoverageType: "full_cover", PolicyNumber: "POL-DEMO-1", InsurerNumber: "ASEG-DEMO-1",
	})
	if err != nil {
		t.Fatalf("insurance devolvió error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("esperaba error de prerrequisito")
	}
	if output.(map[string]any)["error"] != "prerequisite_missing" {
		t.Fatalf("error inesperado: %+v", output)
	}
}

func TestInsuranceAlreadyExists(t *testing.T) {
	fake := &fakeAPI{
		available:  []api.AvailableService{{ID: 12, Name: "Seguro Demo"}},
		insurances: []api.InsuranceDetail{{Insurance: api.Insurance{ID: 3, ServiceID: 12, PolicyNumber: "POL-DEMO-1"}}},
	}
	toolset := newToolset(fake, true)

	result, output, err := toolset.insuranceRegister(context.Background(), nil, InsuranceInput{
		AutoID: 1, ServiceID: 12, CoverageType: "full_cover", PolicyNumber: "POL-DEMO-2", InsurerNumber: "ASEG-DEMO-2",
	})
	if err != nil {
		t.Fatalf("insurance devolvió error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("esperaba already_exists")
	}
	if output.(map[string]any)["error"] != "already_exists" {
		t.Fatalf("error inesperado: %+v", output)
	}
}

func TestInsuranceCreates(t *testing.T) {
	fake := &fakeAPI{available: []api.AvailableService{{ID: 12, Name: "Seguro Demo"}}}
	toolset := newToolset(fake, true)

	_, output, err := toolset.insuranceRegister(context.Background(), nil, InsuranceInput{
		AutoID: 1, ServiceID: 12, CoverageType: "daños_a_terceros", PolicyNumber: "POL-DEMO-1", InsurerNumber: "ASEG-DEMO-1",
		DryRun: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("insurance devolvió error: %v", err)
	}
	result := output.(WriteOutput)
	if result.Status != "created" || result.Insurance == nil || fake.insureReq == nil {
		t.Fatalf("alta inesperada: %+v", result)
	}
}
