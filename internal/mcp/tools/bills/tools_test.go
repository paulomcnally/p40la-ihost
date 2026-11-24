package bills

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

type fakeAPI struct {
	service   *api.Service
	bills     []api.Bill
	genReq    api.GenerateRequest
	payReq    api.PayRequest
	genResult *api.GenerateResult
	payResult *api.PayResult
	genErr    error
	payErr    error
	listErr   error
	calls     int
}

func (f *fakeAPI) GetService(context.Context, int64) (*api.Service, error) {
	if f.service == nil {
		return &api.Service{ID: 1, Name: "Servicio Demo"}, nil
	}
	return f.service, nil
}

func (f *fakeAPI) ListBills(context.Context, int64) ([]api.Bill, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.bills, nil
}

func (f *fakeAPI) GenerateBills(_ context.Context, _ int64, req api.GenerateRequest) (*api.GenerateResult, error) {
	f.calls++
	f.genReq = req
	if f.genErr != nil {
		return nil, f.genErr
	}
	if f.genResult != nil {
		return f.genResult, nil
	}
	return &api.GenerateResult{ServiceID: 1, From: req.From, To: req.To, DryRun: req.DryRun}, nil
}

func (f *fakeAPI) PayBills(_ context.Context, _ int64, req api.PayRequest) (*api.PayResult, error) {
	f.calls++
	f.payReq = req
	if f.payErr != nil {
		return nil, f.payErr
	}
	if f.payResult != nil {
		return f.payResult, nil
	}
	return &api.PayResult{ServiceID: 1, PaidAt: req.PaidAt, DryRun: req.DryRun}, nil
}

func fixedNow() time.Time {
	return time.Date(2026, time.November, 23, 15, 0, 0, 0, time.UTC)
}

func testConfig() config.Config {
	return config.Config{Timezone: "UTC", Location: time.UTC}
}

func newTestToolset(fake *fakeAPI, allowWrites bool) *Toolset {
	toolset := New(fake, config.Config{
		AllowWrites: allowWrites,
		Timezone:    "UTC",
		Location:    time.UTC,
	})
	toolset.now = fixedNow
	return toolset
}

func resultErrorCode(t *testing.T, result any) string {
	t.Helper()
	payload, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("resultado de error inesperado: %#v", result)
	}
	code, _ := payload["error"].(string)
	return code
}

func TestListFiltersAndTotals(t *testing.T) {
	fake := &fakeAPI{
		service: &api.Service{ID: 12, Name: "Servicio Demo"},
		bills: []api.Bill{
			{ID: 1, Year: 2026, Month: 1, Amount: 100, Status: "pending"},
			{ID: 2, Year: 2026, Month: 2, Amount: 200, Status: "paid"},
			{ID: 3, Year: 2026, Month: 3, Amount: 300, Status: "pending"},
		},
	}
	toolset := newTestToolset(fake, false)
	from, to, status := "2026-02", "2026-03", "pending"

	_, output, err := toolset.list(context.Background(), nil, ListInput{
		ServiceID: 12, From: &from, To: &to, Status: &status,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	result, ok := output.(ListOutput)
	if !ok {
		t.Fatalf("output inesperado: %#v", output)
	}
	if result.Count != 1 || result.Bills[0].ID != 3 || result.TotalAmount != 300 {
		t.Fatalf("listado = %+v", result)
	}
	if result.Pending != 1 || result.Paid != 0 {
		t.Fatalf("conteos = pending:%d paid:%d", result.Pending, result.Paid)
	}
}

func TestGenerateDefaultsAndWriteGate(t *testing.T) {
	fake := &fakeAPI{}
	toolset := newTestToolset(fake, false)

	_, output, err := toolset.generate(context.Background(), nil, GenerateInput{ServiceID: 1, From: "2026-01"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	result, ok := output.(*api.GenerateResult)
	if !ok {
		t.Fatalf("output inesperado: %#v", output)
	}
	if !result.DryRun {
		t.Fatal("dry_run debe ser true por defecto")
	}
	if fake.genReq.To != "2026-11" {
		t.Fatalf("to = %q, quiero 2026-11", fake.genReq.To)
	}

	notDry := false
	disabled, disabledOutput, err := toolset.generate(context.Background(), nil, GenerateInput{
		ServiceID: 1, From: "2026-01", DryRun: &notDry,
	})
	if err != nil {
		t.Fatalf("generate con escritura: %v", err)
	}
	if disabled == nil || !disabled.IsError {
		t.Fatalf("quiero error de escritura deshabilitada, obtuve %#v", disabled)
	}
	if code := resultErrorCode(t, disabledOutput); code != "writes_disabled" {
		t.Fatalf("código = %q, quiero writes_disabled", code)
	}
	if fake.calls != 1 {
		t.Fatalf("llamadas = %d, quiero 1 (solo el dry run)", fake.calls)
	}

	enabled := newTestToolset(fake, true)
	_, _, err = enabled.generate(context.Background(), nil, GenerateInput{
		ServiceID: 1, From: "2026-01", DryRun: &notDry,
	})
	if err != nil {
		t.Fatalf("generate con escrituras habilitadas: %v", err)
	}
	if fake.genReq.DryRun {
		t.Fatal("quiero dry_run=false cuando la compuerta está activa")
	}
}

func TestGenerateRangeValidation(t *testing.T) {
	toolset := newTestToolset(&fakeAPI{}, false)
	to := "2026-12"
	result, output, err := toolset.generate(context.Background(), nil, GenerateInput{
		ServiceID: 1, From: "2026-01", To: &to,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("quiero error para rango futuro")
	}
	if code := resultErrorCode(t, output); code != "invalid_range" {
		t.Fatalf("código = %q, quiero invalid_range", code)
	}
}

func TestPaySelectorDefaultsAndValidation(t *testing.T) {
	fake := &fakeAPI{}
	toolset := newTestToolset(fake, false)
	upTo := "2026-11"

	_, output, err := toolset.pay(context.Background(), nil, PayInput{
		ServiceID: 1, AllPendingUpTo: &upTo,
	})
	if err != nil {
		t.Fatalf("pay: %v", err)
	}
	result, ok := output.(*api.PayResult)
	if !ok {
		t.Fatalf("output inesperado: %#v", output)
	}
	if !result.DryRun {
		t.Fatal("dry_run debe ser true por defecto")
	}
	if fake.payReq.PaidAt != "2026-11-23" {
		t.Fatalf("paid_at = %q, quiero 2026-11-23", fake.payReq.PaidAt)
	}
	if fake.payReq.AllPendingUpTo != "2026-11" {
		t.Fatalf("selector = %q", fake.payReq.AllPendingUpTo)
	}

	cases := []struct {
		name  string
		input PayInput
	}{
		{name: "sin selector", input: PayInput{ServiceID: 1}},
		{name: "doble selector", input: PayInput{ServiceID: 1, BillIDs: []int64{1}, Month: &upTo}},
		{name: "rango incompleto", input: PayInput{ServiceID: 1, From: &upTo}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, output, err := toolset.pay(context.Background(), nil, tc.input)
			if err != nil {
				t.Fatalf("pay: %v", err)
			}
			if result == nil || !result.IsError {
				t.Fatal("quiero error de selector")
			}
			if code := resultErrorCode(t, output); code != "invalid_selector" {
				t.Fatalf("código = %q, quiero invalid_selector", code)
			}
		})
	}
}

func TestPayRejectsFutureDateAndWriteGate(t *testing.T) {
	fake := &fakeAPI{}
	toolset := newTestToolset(fake, false)
	upTo := "2026-11"
	future := "2026-12-01"

	result, output, err := toolset.pay(context.Background(), nil, PayInput{
		ServiceID: 1, AllPendingUpTo: &upTo, PaidAt: &future,
	})
	if err != nil {
		t.Fatalf("pay: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("quiero error de fecha futura")
	}
	if code := resultErrorCode(t, output); code != "invalid_paid_at" {
		t.Fatalf("código = %q, quiero invalid_paid_at", code)
	}

	notDry := false
	result, output, err = toolset.pay(context.Background(), nil, PayInput{
		ServiceID: 1, AllPendingUpTo: &upTo, DryRun: &notDry,
	})
	if err != nil {
		t.Fatalf("pay: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatal("quiero error de escritura deshabilitada")
	}
	if code := resultErrorCode(t, output); code != "writes_disabled" {
		t.Fatalf("código = %q, quiero writes_disabled", code)
	}
	if fake.calls != 0 {
		t.Fatalf("llamadas = %d, quiero 0", fake.calls)
	}
}

func TestAPIErrorMapping(t *testing.T) {
	fake := &fakeAPI{genErr: &api.APIError{Status: 404, Code: "service_not_found", Message: "el servicio no existe"}}
	toolset := newTestToolset(fake, false)
	_, output, err := toolset.generate(context.Background(), nil, GenerateInput{ServiceID: 99, From: "2026-01"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if code := resultErrorCode(t, output); code != "service_not_found" {
		t.Fatalf("código = %q, quiero service_not_found", code)
	}
}

func TestResultJSONContent(t *testing.T) {
	result, _, err := jsonResult(map[string]string{"status": "ok"})
	if err != nil {
		t.Fatalf("jsonResult: %v", err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("content = %d, quiero 1", len(result.Content))
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content inesperado: %#v", result.Content[0])
	}
	var decoded map[string]string
	if err := json.Unmarshal([]byte(text.Text), &decoded); err != nil {
		t.Fatalf("decodificar content: %v", err)
	}
	if decoded["status"] != "ok" {
		t.Fatalf("status = %q, quiero ok", decoded["status"])
	}
}

func TestHTTPRemoteWritesRequireBothGates(t *testing.T) {
	fake := &fakeAPI{service: &api.Service{ID: 1, Name: "Servicio Demo"}}
	toolset := New(fake, config.Config{
		Transport:    config.TransportHTTP,
		AllowWrites:  true,
		RemoteWrites: false,
		Timezone:     "UTC",
		Location:     time.UTC,
	})
	toolset.now = fixedNow

	notDry := false
	result, output, err := toolset.generate(context.Background(), nil, GenerateInput{
		ServiceID: 1, From: "2026-01", DryRun: &notDry,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("quiero writes_disabled, obtuve %#v", result)
	}
	if code := resultErrorCode(t, output); code != "writes_disabled" {
		t.Fatalf("código = %q", code)
	}
	if fake.calls != 0 {
		t.Fatalf("llamadas = %d, quiero 0", fake.calls)
	}
}
