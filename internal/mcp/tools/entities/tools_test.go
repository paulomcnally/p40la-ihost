package entities

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

type fakeAPI struct {
	homes        []api.Home
	institutions []api.Institution
	banks        []api.Bank
	currencies   []api.Currency
	categories   []api.InstitutionCategory
	services     []api.Service
	nextID       int64
	createdHome  *api.HomeCreateRequest
	createdSvc   *api.ServiceCreateRequest
	posts        int
}

func (f *fakeAPI) GetService(context.Context, int64) (*api.Service, error) { return nil, nil }
func (f *fakeAPI) ListBills(context.Context, int64) ([]api.Bill, error)    { return nil, nil }
func (f *fakeAPI) GenerateBills(context.Context, int64, api.GenerateRequest) (*api.GenerateResult, error) {
	return nil, nil
}
func (f *fakeAPI) PayBills(context.Context, int64, api.PayRequest) (*api.PayResult, error) {
	return nil, nil
}

func (f *fakeAPI) ListHomes(context.Context) ([]api.Home, error) { return f.homes, nil }
func (f *fakeAPI) CreateHome(_ context.Context, req api.HomeCreateRequest) (*api.Home, error) {
	f.posts++
	f.createdHome = &req
	f.nextID++
	home := api.Home{ID: f.nextID, Name: req.Name, Address: req.Address, Usage: req.Usage}
	f.homes = append(f.homes, home)
	return &home, nil
}

func (f *fakeAPI) ListInstitutions(context.Context) ([]api.Institution, error) {
	return f.institutions, nil
}
func (f *fakeAPI) CreateInstitution(_ context.Context, req api.InstitutionCreateRequest) (*api.Institution, error) {
	f.posts++
	f.nextID++
	item := api.Institution{ID: f.nextID, Name: req.Name, CategoryID: req.CategoryID}
	f.institutions = append(f.institutions, item)
	return &item, nil
}

func (f *fakeAPI) ListBanks(context.Context) ([]api.Bank, error) { return f.banks, nil }
func (f *fakeAPI) CreateBank(_ context.Context, req api.BankCreateRequest) (*api.Bank, error) {
	f.posts++
	f.nextID++
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	item := api.Bank{ID: f.nextID, Name: req.Name, ShortCode: req.ShortCode, Country: req.Country, Notes: req.Notes, Active: active}
	f.banks = append(f.banks, item)
	return &item, nil
}

func (f *fakeAPI) ListCurrencies(context.Context) ([]api.Currency, error) { return f.currencies, nil }
func (f *fakeAPI) ListInstitutionCategories(context.Context) ([]api.InstitutionCategory, error) {
	return f.categories, nil
}
func (f *fakeAPI) ListServices(_ context.Context, homeID *int64) ([]api.Service, error) {
	if homeID == nil {
		return f.services, nil
	}
	var out []api.Service
	for _, item := range f.services {
		if item.HomeID == *homeID {
			out = append(out, item)
		}
	}
	return out, nil
}
func (f *fakeAPI) CreateService(_ context.Context, req api.ServiceCreateRequest) (*api.Service, error) {
	f.posts++
	f.createdSvc = &req
	f.nextID++
	item := api.Service{
		ID: f.nextID, HomeID: req.HomeID, Name: req.Name, Institution: req.Institution,
		CurrencyID: req.CurrencyID, Frequency: req.Frequency, SuggestedAmount: req.SuggestedAmount,
		Active: req.Active, IconKey: req.IconKey, BillingType: req.BillingType,
		BillingDay: req.BillingDay, AutoGenerate: req.AutoGenerate, IsRecurring: req.IsRecurring,
	}
	f.services = append(f.services, item)
	return &item, nil
}

func testConfig(writes bool) config.Config {
	return config.Config{Transport: config.TransportStdio, AllowWrites: writes, Location: time.UTC}
}

func newTestToolset(f *fakeAPI, writes bool) *Toolset {
	return New(f, testConfig(writes))
}

func TestHomesCreateAndDuplicate(t *testing.T) {
	f := &fakeAPI{}
	tool := newTestToolset(f, true)

	result, output, err := tool.homesCreate(context.Background(), nil, HomesCreateInput{Name: "Hogar Demo", Address: "Calle Demo 123"})
	if err != nil || result.IsError {
		t.Fatalf("create: err=%v result=%#v", err, result)
	}
	home, ok := output.(*api.Home)
	if !ok || home.ID == 0 || home.Usage != "rental" {
		t.Fatalf("home=%#v", output)
	}

	result, output, _ = tool.homesCreate(context.Background(), nil, HomesCreateInput{Name: "  hogar demo "})
	if !result.IsError {
		t.Fatal("quiero already_exists")
	}
	payload := output.(map[string]any)
	if payload["error"] != "already_exists" {
		t.Fatalf("payload=%v", payload)
	}
	if f.posts != 1 {
		t.Fatalf("posts=%d, quiero 1", f.posts)
	}
}

func TestHomesInvalidUsage(t *testing.T) {
	f := &fakeAPI{}
	tool := newTestToolset(f, true)
	result, output, _ := tool.homesCreate(context.Background(), nil, HomesCreateInput{Name: "Hogar Demo", Usage: "otro"})
	if !result.IsError || output.(map[string]any)["error"] != "invalid_value" {
		t.Fatalf("result=%#v output=%v", result, output)
	}
	if f.posts != 0 {
		t.Fatalf("posts=%d, quiero 0", f.posts)
	}
}

func TestWritesDisabled(t *testing.T) {
	f := &fakeAPI{}
	tool := newTestToolset(f, false)
	_, output, _ := tool.homesCreate(context.Background(), nil, HomesCreateInput{Name: "Hogar Demo"})
	if output.(map[string]any)["error"] != "writes_disabled" {
		t.Fatalf("output=%v", output)
	}
	if f.posts != 0 {
		t.Fatalf("posts=%d", f.posts)
	}
}

func TestInstitutionCategoryNotFound(t *testing.T) {
	f := &fakeAPI{categories: []api.InstitutionCategory{{ID: 3, Key: "servicios", Name: "Servicios"}}}
	tool := newTestToolset(f, true)
	result, output, _ := tool.institutionsCreate(context.Background(), nil, InstitutionsCreateInput{
		Name: "Proveedor Demo", Category: "no-existe",
	})
	if !result.IsError {
		t.Fatal("quiero not_found")
	}
	payload := output.(map[string]any)
	if payload["error"] != "not_found" {
		t.Fatalf("payload=%v", payload)
	}
	if hint, _ := payload["hint"].(string); !strings.Contains(hint, "3=servicios") {
		t.Fatalf("hint=%q", hint)
	}
}

func TestBanksCountryInvalid(t *testing.T) {
	f := &fakeAPI{}
	tool := newTestToolset(f, true)
	result, output, _ := tool.banksCreate(context.Background(), nil, BanksCreateInput{Name: "Banco Demo", Country: "NIC"})
	if !result.IsError || output.(map[string]any)["error"] != "invalid_value" {
		t.Fatalf("result=%#v output=%v", result, output)
	}
}

func TestServicesCreateResolvesReferences(t *testing.T) {
	f := &fakeAPI{
		homes:      []api.Home{{ID: 7, Name: "Hogar Demo"}},
		currencies: []api.Currency{{ID: 1, Code: "NIO", Name: "Córdoba"}},
		institutions: []api.Institution{
			{ID: 9, Name: "Proveedor Demo"},
		},
	}
	tool := newTestToolset(f, true)
	result, output, err := tool.servicesCreate(context.Background(), nil, ServicesCreateInput{
		Name: "Internet Demo", Home: "hogar demo", Currency: "nio", Institution: "Proveedor Demo",
	})
	if err != nil || result.IsError {
		t.Fatalf("create: err=%v result=%#v", err, result)
	}
	out, ok := output.(ServicesCreateOutput)
	if !ok || out.ID == 0 || !out.CurrentBillGenerated {
		t.Fatalf("output=%#v", output)
	}
	if f.createdSvc == nil || f.createdSvc.HomeID != 7 || f.createdSvc.CurrencyID != 1 || f.createdSvc.InstitutionID == nil || *f.createdSvc.InstitutionID != 9 {
		t.Fatalf("req=%#v", f.createdSvc)
	}
}

func TestServicesAmbiguousHome(t *testing.T) {
	f := &fakeAPI{
		homes:      []api.Home{{ID: 7, Name: "Hogar Demo"}, {ID: 8, Name: "Hogar Demo"}},
		currencies: []api.Currency{{ID: 1, Code: "NIO", Name: "Córdoba"}},
	}
	tool := newTestToolset(f, true)
	_, output, _ := tool.servicesCreate(context.Background(), nil, ServicesCreateInput{
		Name: "Internet Demo", Home: "Hogar Demo", Currency: "NIO",
	})
	if output.(map[string]any)["error"] != "ambiguous_reference" {
		t.Fatalf("output=%v", output)
	}
}

func TestProtocolToolsList(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "p40la-mcp-test", Version: "0.1.0"}, nil)
	Register(server, &fakeAPI{}, testConfig(true))

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("conectar server: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("conectar cliente: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	names := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"entities_search", "homes_create", "institutions_create", "banks_create", "services_create"} {
		if !names[want] {
			t.Fatalf("falta %q en %v", want, names)
		}
	}
}

func TestInstitutionsCreate(t *testing.T) {
	f := &fakeAPI{categories: []api.InstitutionCategory{{ID: 3, Key: "servicios", Name: "Servicios"}}}
	tool := newTestToolset(f, true)
	result, output, err := tool.institutionsCreate(context.Background(), nil, InstitutionsCreateInput{
		Name: "Proveedor Demo", Category: "servicios",
	})
	if err != nil || result.IsError {
		t.Fatalf("create: err=%v result=%#v", err, result)
	}
	item := output.(*api.Institution)
	if item.ID == 0 || item.CategoryID == nil || *item.CategoryID != 3 {
		t.Fatalf("item=%#v", item)
	}
}

func TestBanksCreate(t *testing.T) {
	f := &fakeAPI{}
	tool := newTestToolset(f, true)
	result, output, err := tool.banksCreate(context.Background(), nil, BanksCreateInput{
		Name: "Banco Demo", ShortCode: "DEMO", Country: "ni", Notes: "Cuenta sintética",
	})
	if err != nil || result.IsError {
		t.Fatalf("create: err=%v result=%#v", err, result)
	}
	item := output.(*api.Bank)
	if item.ID == 0 || item.Country != "NI" || !item.Active {
		t.Fatalf("item=%#v", item)
	}
}
