package pension

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

func TestPensionTools(t *testing.T) {
	query := "?month=11&year=2026"
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{
		"/api/pension/records" + query: {Body: []map[string]any{
			{"id": 1, "amount": 100.0, "currency": "NIO", "status": "paid", "child_name": "Demo"},
			{"id": 2, "amount": 50.0, "currency": "NIO", "status": "pending", "child_name": "Demo"},
		}},
		"/api/pension/salary-payments" + query: {Body: []map[string]any{
			{"id": 9, "amount": 900.0, "currency": "NIO", "status": "received", "employer": "Proveedor Demo"},
		}},
		"/api/pension/closing/2026/11": {Body: map[string]any{"closed": false}},
		"/api/children":                {Body: []map[string]any{{"id": 1, "first_name": "Demo", "last_name": "Test"}}},
		"/api/pension-categories":      {Body: []map[string]any{{"id": 1, "name": "Colegio", "share_percent": 50}}},
		"/api/salaries":                {Body: []map[string]any{{"id": 1, "employer": "Proveedor Demo"}}},
		"/api/pension/configs":         {Body: []map[string]any{{"id": 1, "child_id": 1}}},
		"/api/currencies":              {Body: []map[string]any{{"id": 1, "code": "NIO"}}},
	}}

	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	names := toolkitest.ToolNames(t, session)
	for _, want := range []string{"pension_month_summary", "pension_catalogs_list"} {
		if !names[want] {
			t.Fatalf("falta la tool %q", want)
		}
	}

	summary := toolkitest.Call(t, session, "pension_month_summary", map[string]any{"period": "2026-11"})
	for _, want := range []string{`"paid_count": 1`, `"pending_count": 1`, `"closed": false`, `"currency": "NIO"`} {
		if !strings.Contains(summary, want) {
			t.Fatalf("resumen sin %q: %s", want, summary)
		}
	}

	catalogs := toolkitest.Call(t, session, "pension_catalogs_list", map[string]any{})
	for _, want := range []string{`"children"`, `"pension_categories"`, `"currencies"`} {
		if !strings.Contains(catalogs, want) {
			t.Fatalf("catálogos sin %q: %s", want, catalogs)
		}
	}
}

func TestPensionRejectsBadPeriod(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{}}
	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	out := toolkitest.Call(t, session, "pension_month_summary", map[string]any{"period": "2026-13"})
	if !strings.Contains(out, `"error":"invalid_period"`) {
		t.Fatalf("esperaba invalid_period: %s", out)
	}
}
