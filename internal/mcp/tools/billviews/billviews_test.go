package billviews

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

func TestBillViews(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{
		"/api/service-bills?month=11&year=2026": {Body: map[string]any{
			"bills": []any{}, "total": 0,
		}},
		"/api/service-bills/summary?from=2026-01&to=2026-03": {Body: map[string]any{
			"from": "2026-01", "to": "2026-03", "periods": []any{},
		}},
		"/api/service-bills/projection?from=2026-01&months=3": {Body: map[string]any{
			"months": []any{},
		}},
	}}

	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	names := toolkitest.ToolNames(t, session)
	for _, want := range []string{"bills_by_month", "bills_summary", "bills_projection"} {
		if !names[want] {
			t.Fatalf("falta la tool %q", want)
		}
	}

	byMonth := toolkitest.Call(t, session, "bills_by_month", map[string]any{"period": "2026-11"})
	if !strings.Contains(byMonth, `"total": 0`) {
		t.Fatalf("vista mensual inesperada: %s", byMonth)
	}

	summary := toolkitest.Call(t, session, "bills_summary", map[string]any{"from": "2026-01", "to": "2026-03"})
	if !strings.Contains(summary, `"from": "2026-01"`) {
		t.Fatalf("resumen inesperado: %s", summary)
	}

	projection := toolkitest.Call(t, session, "bills_projection", map[string]any{"from": "2026-01", "months": 3})
	if !strings.Contains(projection, `"months"`) {
		t.Fatalf("proyección inesperada: %s", projection)
	}
}

func TestBillSummaryRejectsInvertedRange(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{}}
	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	out := toolkitest.Call(t, session, "bills_summary", map[string]any{"from": "2026-03", "to": "2026-01"})
	if !strings.Contains(out, `"error":"invalid_range"`) {
		t.Fatalf("esperaba invalid_range: %s", out)
	}
}
