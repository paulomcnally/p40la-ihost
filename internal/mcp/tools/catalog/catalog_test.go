package catalog

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

func TestServiceGet(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{
		"/api/services/12": {Body: map[string]any{
			"id": 12, "name": "Servicio Demo", "billing_day": 10, "is_recurring": true,
		}},
	}}
	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	if !toolkitest.ToolNames(t, session)["services_get"] {
		t.Fatal("falta services_get")
	}
	out := toolkitest.Call(t, session, "services_get", map[string]any{"service_id": 12})
	if !strings.Contains(out, `"billing_day": 10`) {
		t.Fatalf("detalle inesperado: %s", out)
	}
}
