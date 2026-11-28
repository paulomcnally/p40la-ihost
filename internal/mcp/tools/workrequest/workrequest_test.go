package workrequest

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
	listResult *api.HomeWorkRequestList
	listErr    error
	items      map[int64]*api.HomeWorkRequest
	getErr     error

	createResult  *api.HomeWorkRequest
	createErr     error
	updateResult  *api.HomeWorkRequest
	updateErr     error
	convertResult *api.HomeWorkRequestConvertResult
	convertErr    error
	statusResult  *api.HomeWorkRequest
	statusErr     error

	listCalls    int
	getCalls     int
	createCalls  int
	updateCalls  int
	convertCalls int
	statusCalls  int

	lastFilter  api.HomeWorkRequestFilter
	lastCreate  api.HomeWorkRequestRequest
	lastUpdate  api.HomeWorkRequestRequest
	lastConvert api.HomeWorkRequestConvertRequest
	lastStatus  api.HomeWorkRequestStatusRequest
}

func (f *fakeAPI) ListHomeWorkRequests(_ context.Context, _ int64, filter api.HomeWorkRequestFilter) (*api.HomeWorkRequestList, error) {
	f.listCalls++
	f.lastFilter = filter
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResult == nil {
		return &api.HomeWorkRequestList{}, nil
	}
	return f.listResult, nil
}

func (f *fakeAPI) GetHomeWorkRequest(_ context.Context, _, workRequestID int64) (*api.HomeWorkRequest, error) {
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	if item, ok := f.items[workRequestID]; ok {
		return item, nil
	}
	return nil, &api.APIError{Status: 404, Code: "work_request_not_found", Message: "Reporte no encontrado"}
}

func (f *fakeAPI) CreateHomeWorkRequest(_ context.Context, _ int64, req api.HomeWorkRequestRequest) (*api.HomeWorkRequest, error) {
	f.createCalls++
	f.lastCreate = req
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResult != nil {
		return f.createResult, nil
	}
	return &api.HomeWorkRequest{
		ID: 10, HomeID: 3, Title: req.Title, Description: req.Description,
		Category: req.Category, Priority: req.Priority, Status: "reported", ReportedAt: req.ReportedAt,
	}, nil
}

func (f *fakeAPI) UpdateHomeWorkRequest(_ context.Context, _, _ int64, req api.HomeWorkRequestRequest) (*api.HomeWorkRequest, error) {
	f.updateCalls++
	f.lastUpdate = req
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return f.updateResult, nil
}

func (f *fakeAPI) ConvertHomeWorkRequest(_ context.Context, _, _ int64, req api.HomeWorkRequestConvertRequest) (*api.HomeWorkRequestConvertResult, error) {
	f.convertCalls++
	f.lastConvert = req
	if f.convertErr != nil {
		return nil, f.convertErr
	}
	return f.convertResult, nil
}

func (f *fakeAPI) SetHomeWorkRequestStatus(_ context.Context, _, _ int64, req api.HomeWorkRequestStatusRequest) (*api.HomeWorkRequest, error) {
	f.statusCalls++
	f.lastStatus = req
	if f.statusErr != nil {
		return nil, f.statusErr
	}
	return f.statusResult, nil
}

func newServer(t *testing.T, fake *fakeAPI, cfg config.Config) *mcp.ClientSession {
	t.Helper()
	if cfg.Location == nil {
		cfg.Location = time.UTC
	}
	toolset := New(fake, cfg)
	toolset.SetClock(func() time.Time { return time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC) })
	return toolkitest.NewServer(t, toolset.Register)
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

func demoWorkRequest(id int64, status string) *api.HomeWorkRequest {
	return &api.HomeWorkRequest{
		ID: id, HomeID: 3, Title: "Filtración demo", Description: "Gotea cuando llueve",
		Category: "repair", Priority: "high", Status: status, ReportedAt: "2026-11-26",
	}
}

func TestWorkRequestToolsRegistered(t *testing.T) {
	session := newServer(t, &fakeAPI{}, config.Config{})
	names := toolkitest.ToolNames(t, session)
	for _, name := range []string{
		"home_work_request_list",
		"home_work_request_get",
		"home_work_request_create",
		"home_work_request_update",
		"home_work_request_convert",
		"home_work_request_status",
	} {
		if !names[name] {
			t.Fatalf("falta la tool %s", name)
		}
	}
}

func TestListFiltersAndOutput(t *testing.T) {
	fake := &fakeAPI{listResult: &api.HomeWorkRequestList{
		Items:        []api.HomeWorkRequest{*demoWorkRequest(10, "reported")},
		PendingCount: 2,
		Truncated:    true,
	}}
	session := newServer(t, fake, config.Config{})
	out := callJSON(t, session, "home_work_request_list", map[string]any{
		"home_id": 3, "status": "reported", "category": "repair",
	})
	if out["count"] != float64(1) || out["pending_count"] != float64(2) || out["truncated"] != true {
		t.Fatalf("salida inesperada: %v", out)
	}
	if fake.lastFilter.Status != "reported" || fake.lastFilter.Category != "repair" {
		t.Fatalf("filtros inesperados: %+v", fake.lastFilter)
	}
}

func TestListRejectsInvalidInput(t *testing.T) {
	session := newServer(t, &fakeAPI{}, config.Config{})
	cases := []map[string]any{
		{"home_id": 0},
		{"home_id": 3, "status": "open"},
		{"home_id": 3, "category": "broken"},
	}
	for _, args := range cases {
		out := callJSON(t, session, "home_work_request_list", args)
		if out["error"] != "invalid_request" {
			t.Fatalf("args %v: esperaba invalid_request, obtuve %v", args, out)
		}
	}
}

func TestGetDetailWithEvidence(t *testing.T) {
	item := demoWorkRequest(10, "reported")
	item.AttachmentCount = 2
	item.Attachments = []api.HomeWorkRequestAttachment{
		{ID: 1, WorkRequestID: 10, MediaType: "image", FileName: "foto-demo.jpg", MimeType: "image/jpeg", SizeBytes: 1024},
		{ID: 2, WorkRequestID: 10, MediaType: "video", FileName: "video-demo.mp4", MimeType: "video/mp4", SizeBytes: 2048},
	}
	session := newServer(t, &fakeAPI{items: map[int64]*api.HomeWorkRequest{10: item}}, config.Config{})
	out := callJSON(t, session, "home_work_request_get", map[string]any{"home_id": 3, "work_request_id": 10})
	if out["attachment_count"] != float64(2) {
		t.Fatalf("attachment_count = %v", out["attachment_count"])
	}
	workRequest, _ := out["work_request"].(map[string]any)
	attachments, _ := workRequest["attachments"].([]any)
	if len(attachments) != 2 || attachments[0].(map[string]any)["media_type"] != "image" {
		t.Fatalf("adjuntos inesperados: %v", workRequest["attachments"])
	}
}

func TestCreateDryRunByDefaultAndWrite(t *testing.T) {
	fake := &fakeAPI{}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_work_request_create", map[string]any{
		"home_id": 3, "title": "Agregar canaleta demo",
	})
	if out["status"] != "planned" || out["dry_run"] != true || fake.createCalls != 0 {
		t.Fatalf("dry_run inesperado: %v calls=%d", out, fake.createCalls)
	}
	planned, _ := out["planned"].(map[string]any)
	if planned["category"] != "repair" || planned["priority"] != "normal" || planned["reported_at"] != "2026-11-27" {
		t.Fatalf("defaults inesperados: %v", planned)
	}

	out = callJSON(t, session, "home_work_request_create", map[string]any{
		"home_id": 3, "title": "Agregar canaleta demo", "description": "Para el agua de lluvia",
		"category": "improvement", "priority": "low", "reported_at": "2026-11-25", "dry_run": false,
	})
	if out["status"] != "created" || out["dry_run"] != false || fake.createCalls != 1 {
		t.Fatalf("alta inesperada: %v calls=%d", out, fake.createCalls)
	}
	if fake.lastCreate.Category != "improvement" || fake.lastCreate.Priority != "low" ||
		fake.lastCreate.ReportedAt != "2026-11-25" || fake.lastCreate.Title != "Agregar canaleta demo" {
		t.Fatalf("request inesperado: %+v", fake.lastCreate)
	}
}

func TestCreateValidationAndWritesDisabled(t *testing.T) {
	session := newServer(t, &fakeAPI{}, config.Config{AllowWrites: false})
	cases := []map[string]any{
		{"home_id": 3, "title": "  "},
		{"home_id": 3, "title": "Reporte demo", "category": "broken"},
		{"home_id": 3, "title": "Reporte demo", "priority": "urgent"},
		{"home_id": 3, "title": "Reporte demo", "reported_at": "2026-11-28"},
		{"home_id": 3, "title": "Reporte demo", "reported_at": "26/11/2026"},
	}
	for _, args := range cases {
		out := callJSON(t, session, "home_work_request_create", args)
		if out["error"] != "invalid_request" {
			t.Fatalf("args %v: esperaba invalid_request, obtuve %v", args, out)
		}
	}
	out := callJSON(t, session, "home_work_request_create", map[string]any{
		"home_id": 3, "title": "Reporte demo", "dry_run": false,
	})
	if out["error"] != "writes_disabled" {
		t.Fatalf("esperaba writes_disabled, obtuve %v", out)
	}
}

func TestUpdateMergesAndKeepsStatus(t *testing.T) {
	current := demoWorkRequest(10, "reported")
	fake := &fakeAPI{
		items:        map[int64]*api.HomeWorkRequest{10: current},
		updateResult: &api.HomeWorkRequest{ID: 10, HomeID: 3, Title: "Filtración revisada", Description: "Gotea cuando llueve", Category: "repair", Priority: "low", Status: "reported", ReportedAt: "2026-11-26"},
	}
	session := newServer(t, fake, config.Config{AllowWrites: true})
	out := callJSON(t, session, "home_work_request_update", map[string]any{
		"home_id": 3, "work_request_id": 10, "title": "Filtración revisada", "priority": "low", "dry_run": false,
	})
	if out["status"] != "updated" || fake.updateCalls != 1 {
		t.Fatalf("edición inesperada: %v calls=%d", out, fake.updateCalls)
	}
	if fake.lastUpdate.Title != "Filtración revisada" || fake.lastUpdate.Priority != "low" ||
		fake.lastUpdate.Category != "repair" || fake.lastUpdate.ReportedAt != "2026-11-26" {
		t.Fatalf("merge inesperado: %+v", fake.lastUpdate)
	}
	changed, _ := out["changed_fields"].([]any)
	if len(changed) != 2 {
		t.Fatalf("changed_fields = %v", out["changed_fields"])
	}
}

func TestUpdateDryRunAndUnchanged(t *testing.T) {
	current := demoWorkRequest(10, "reported")
	fake := &fakeAPI{items: map[int64]*api.HomeWorkRequest{10: current}}
	session := newServer(t, fake, config.Config{AllowWrites: true})

	out := callJSON(t, session, "home_work_request_update", map[string]any{
		"home_id": 3, "work_request_id": 10, "description": "Nueva descripción demo",
	})
	if out["status"] != "planned" || out["dry_run"] != true || fake.updateCalls != 0 {
		t.Fatalf("dry_run inesperado: %v calls=%d", out, fake.updateCalls)
	}

	out = callJSON(t, session, "home_work_request_update", map[string]any{
		"home_id": 3, "work_request_id": 10, "title": "Filtración demo", "dry_run": false,
	})
	if out["status"] != "unchanged" || fake.updateCalls != 0 {
		t.Fatalf("unchanged inesperado: %v calls=%d", out, fake.updateCalls)
	}
}

func TestUpdateRejectsEmptyTitleAndNoFields(t *testing.T) {
	session := newServer(t, &fakeAPI{items: map[int64]*api.HomeWorkRequest{10: demoWorkRequest(10, "reported")}}, config.Config{AllowWrites: true})
	for _, args := range []map[string]any{
		{"home_id": 3, "work_request_id": 10},
		{"home_id": 3, "work_request_id": 10, "title": "  "},
	} {
		out := callJSON(t, session, "home_work_request_update", args)
		if out["error"] != "invalid_request" {
			t.Fatalf("args %v: esperaba invalid_request, obtuve %v", args, out)
		}
	}
}

func TestConvertDryRunWriteAndErrors(t *testing.T) {
	current := demoWorkRequest(10, "reported")
	fake := &fakeAPI{
		items: map[int64]*api.HomeWorkRequest{10: current},
		convertResult: &api.HomeWorkRequestConvertResult{
			MaintenanceID: 42,
			WorkRequest: func() *api.HomeWorkRequest {
				item := demoWorkRequest(10, "converted")
				id := int64(42)
				item.MaintenanceID = &id
				return item
			}(),
		},
	}
	session := newServer(t, fake, config.Config{AllowWrites: true})

	out := callJSON(t, session, "home_work_request_convert", map[string]any{"home_id": 3, "work_request_id": 10})
	if out["status"] != "planned" || fake.convertCalls != 0 {
		t.Fatalf("dry_run inesperado: %v calls=%d", out, fake.convertCalls)
	}

	out = callJSON(t, session, "home_work_request_convert", map[string]any{
		"home_id": 3, "work_request_id": 10, "kind": "preventive", "area": "roof", "dry_run": false,
	})
	if out["status"] != "converted" || out["maintenance_id"] != float64(42) || fake.convertCalls != 1 {
		t.Fatalf("conversión inesperada: %v calls=%d", out, fake.convertCalls)
	}
	if fake.lastConvert.Kind != "preventive" || fake.lastConvert.Area != "roof" {
		t.Fatalf("convert request inesperado: %+v", fake.lastConvert)
	}

	converted := demoWorkRequest(11, "converted")
	session = newServer(t, &fakeAPI{items: map[int64]*api.HomeWorkRequest{11: converted}}, config.Config{AllowWrites: true})
	out = callJSON(t, session, "home_work_request_convert", map[string]any{"home_id": 3, "work_request_id": 11})
	if out["error"] != "already_converted" {
		t.Fatalf("esperaba already_converted, obtuve %v", out)
	}

	dismissed := demoWorkRequest(12, "dismissed")
	session = newServer(t, &fakeAPI{items: map[int64]*api.HomeWorkRequest{12: dismissed}}, config.Config{AllowWrites: true})
	out = callJSON(t, session, "home_work_request_convert", map[string]any{"home_id": 3, "work_request_id": 12})
	if out["error"] != "invalid_status" {
		t.Fatalf("esperaba invalid_status, obtuve %v", out)
	}
}

func TestStatusDismissReactivateAndErrors(t *testing.T) {
	reported := demoWorkRequest(10, "reported")
	dismissed := demoWorkRequest(10, "dismissed")
	fake := &fakeAPI{
		items:        map[int64]*api.HomeWorkRequest{10: reported},
		statusResult: dismissed,
	}
	session := newServer(t, fake, config.Config{AllowWrites: true})

	out := callJSON(t, session, "home_work_request_status", map[string]any{
		"home_id": 3, "work_request_id": 10, "status": "dismissed",
	})
	if out["status"] != "planned" || fake.statusCalls != 0 {
		t.Fatalf("dry_run inesperado: %v calls=%d", out, fake.statusCalls)
	}

	out = callJSON(t, session, "home_work_request_status", map[string]any{
		"home_id": 3, "work_request_id": 10, "status": "dismissed", "dry_run": false,
	})
	if out["status"] != "updated" || fake.statusCalls != 1 || fake.lastStatus.Status != "dismissed" {
		t.Fatalf("descarte inesperado: %v calls=%d", out, fake.statusCalls)
	}

	fake = &fakeAPI{
		items:        map[int64]*api.HomeWorkRequest{10: dismissed},
		statusResult: reported,
	}
	session = newServer(t, fake, config.Config{AllowWrites: true})
	out = callJSON(t, session, "home_work_request_status", map[string]any{
		"home_id": 3, "work_request_id": 10, "status": "reported", "dry_run": false,
	})
	if out["status"] != "updated" || fake.lastStatus.Status != "reported" {
		t.Fatalf("reactivación inesperada: %v", out)
	}

	fake.items[10] = reported
	out = callJSON(t, session, "home_work_request_status", map[string]any{
		"home_id": 3, "work_request_id": 10, "status": "reported", "dry_run": false,
	})
	if out["status"] != "unchanged" || fake.statusCalls != 1 {
		t.Fatalf("idempotencia inesperada: %v calls=%d", out, fake.statusCalls)
	}

	converted := demoWorkRequest(11, "converted")
	session = newServer(t, &fakeAPI{items: map[int64]*api.HomeWorkRequest{11: converted}}, config.Config{AllowWrites: true})
	out = callJSON(t, session, "home_work_request_status", map[string]any{
		"home_id": 3, "work_request_id": 11, "status": "dismissed",
	})
	if out["error"] != "invalid_status" {
		t.Fatalf("esperaba invalid_status, obtuve %v", out)
	}
}

func TestWritesDisabledForConvertAndStatus(t *testing.T) {
	fake := &fakeAPI{items: map[int64]*api.HomeWorkRequest{10: demoWorkRequest(10, "reported")}}
	session := newServer(t, fake, config.Config{AllowWrites: false})
	for _, call := range []struct {
		name string
		args map[string]any
	}{
		{"home_work_request_convert", map[string]any{"home_id": 3, "work_request_id": 10, "dry_run": false}},
		{"home_work_request_status", map[string]any{"home_id": 3, "work_request_id": 10, "status": "dismissed", "dry_run": false}},
	} {
		out := callJSON(t, session, call.name, call.args)
		if out["error"] != "writes_disabled" {
			t.Fatalf("%s: esperaba writes_disabled, obtuve %v", call.name, out)
		}
	}
	if fake.convertCalls != 0 || fake.statusCalls != 0 {
		t.Fatalf("escrituras con compuerta cerrada: convert=%d status=%d", fake.convertCalls, fake.statusCalls)
	}
}

func TestAPIErrorPropagation(t *testing.T) {
	fake := &fakeAPI{listErr: &api.APIError{Status: 404, Code: "home_not_found", Message: "Vivienda no encontrada"}}
	session := newServer(t, fake, config.Config{})
	out := callJSON(t, session, "home_work_request_list", map[string]any{"home_id": 99})
	if out["error"] != "home_not_found" || !strings.Contains(out["message"].(string), "Vivienda") {
		t.Fatalf("error inesperado: %v", out)
	}
}
