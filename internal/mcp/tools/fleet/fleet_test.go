package fleet

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

func TestInspectionAlerts(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{
		"/api/autos/inspection-alerts": {Body: []map[string]any{
			{"auto_id": 1, "status": "due_soon"},
		}},
	}}
	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	if !toolkitest.ToolNames(t, session)["autos_inspection_alerts"] {
		t.Fatal("falta autos_inspection_alerts")
	}
	out := toolkitest.Call(t, session, "autos_inspection_alerts", map[string]any{})
	if !strings.Contains(out, `"due_soon"`) {
		t.Fatalf("alertas inesperadas: %s", out)
	}
}
