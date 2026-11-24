package homemaint

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

func TestHomeMaintenanceList(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{
		"/api/homes/3/maintenances?kind=preventive&year=2026": {Body: map[string]any{
			"items": []map[string]any{
				{"id": 1, "performed_at": "2026-03-10", "kind": "preventive"},
			},
			"totals_by_currency": []map[string]any{
				{"currency_code": "NIO", "labor_cost": 500.0, "supplies_cost": 200.0, "total": 700.0},
			},
		}},
	}}
	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	if !toolkitest.ToolNames(t, session)["home_maintenance_list"] {
		t.Fatal("falta home_maintenance_list")
	}
	out := toolkitest.Call(t, session, "home_maintenance_list", map[string]any{
		"home_id": 3, "year": 2026, "kind": "preventive",
	})
	if !strings.Contains(out, `"count": 1`) {
		t.Fatalf("listado inesperado: %s", out)
	}
	if !strings.Contains(out, `"total": 700`) {
		t.Fatalf("faltan totales por moneda: %s", out)
	}
}

func TestHomeMaintenanceRejectsBadKind(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{}}
	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	out := toolkitest.Call(t, session, "home_maintenance_list", map[string]any{
		"home_id": 3, "kind": "otro",
	})
	if !strings.Contains(out, `"error":"invalid_request"`) {
		t.Fatalf("esperaba invalid_request: %s", out)
	}
}
