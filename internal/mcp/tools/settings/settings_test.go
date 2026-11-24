package settings

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

func TestSettingsStatusAllowlist(t *testing.T) {
	fake := &toolkitest.Fake{Responses: map[string]toolkitest.Response{
		"/api/system-settings": {Body: map[string]any{
			"timezone":              "America/Managua",
			"country_code":          "NI",
			"smtp_configured":       true,
			"smtp_from_email":       "demo@example.com",
			"alert_emails":          "demo@example.com",
			"telegram_bot_chat_ids": "123456",
			"smtp_password":         "demo-secret",
			"webhook_base_url":      "https://example.com",
		}},
	}}
	session := toolkitest.NewServer(t, func(server *mcp.Server) {
		Register(server, fake, config.Config{})
	})
	if !toolkitest.ToolNames(t, session)["settings_status"] {
		t.Fatal("falta settings_status")
	}
	out := toolkitest.Call(t, session, "settings_status", map[string]any{})
	for _, secret := range []string{"smtp_from_email", "alert_emails", "telegram_bot_chat_ids", "smtp_password"} {
		if strings.Contains(out, secret) {
			t.Fatalf("no debe exponer %q: %s", secret, out)
		}
	}
	if !strings.Contains(out, `"timezone": "America/Managua"`) {
		t.Fatalf("estado inesperado: %s", out)
	}
}
