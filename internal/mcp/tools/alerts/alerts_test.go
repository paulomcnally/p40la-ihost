package alerts

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

func TestAlertsListOmitsSpeech(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{
		"/api/alerts": {Body: []map[string]any{
			{"key": "bills_due", "title": "Facturas por vencer", "speech": "Demo Personal", "mail_enabled": true},
		}},
	}}
	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	if !toolkitest.ToolNames(t, session)["alerts_list"] {
		t.Fatal("falta alerts_list")
	}
	out := toolkitest.Call(t, session, "alerts_list", map[string]any{})
	if strings.Contains(out, "speech") || strings.Contains(out, "Demo Personal") {
		t.Fatalf("no debe exponer el texto de voz: %s", out)
	}
	if !strings.Contains(out, `"key": "bills_due"`) {
		t.Fatalf("alerta inesperada: %s", out)
	}
}
