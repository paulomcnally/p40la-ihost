package automation

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

func TestAutomationTools(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{
		"/api/automation/accounts": {Body: []map[string]any{
			{"id": 1, "label": "Demo", "identifier": "demo-account-001", "webhook_url": "https://example.com/hook"},
		}},
		"/api/automation/accounts/1/bills": {Body: []map[string]any{
			{"id": 5, "period": "2026-11", "payload": map[string]any{"secret": "demo"}},
		}},
		"/api/automation/accounts/1/webhook/logs?limit=10": {Body: []map[string]any{
			{"id": 7, "status": 200, "webhook_url": "https://example.com/hook"},
		}},
	}}

	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	names := toolkitest.ToolNames(t, session)
	for _, want := range []string{"automation_accounts_list", "automation_runs_list"} {
		if !names[want] {
			t.Fatalf("falta la tool %q", want)
		}
	}

	accounts := toolkitest.Call(t, session, "automation_accounts_list", map[string]any{})
	if strings.Contains(accounts, "identifier") || strings.Contains(accounts, "webhook_url") {
		t.Fatalf("no debe exponer identificadores ni webhooks: %s", accounts)
	}

	runs := toolkitest.Call(t, session, "automation_runs_list", map[string]any{"account_id": 1, "limit": 10})
	if strings.Contains(runs, "payload") || strings.Contains(runs, "webhook_url") {
		t.Fatalf("no debe exponer payloads ni webhooks: %s", runs)
	}
}
