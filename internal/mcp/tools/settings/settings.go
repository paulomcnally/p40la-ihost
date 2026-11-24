// Package settings implementa la tool MCP de solo lectura del estado de
// configuración e integraciones, con allowlist de campos (SPEC-154, Ola 1).
package settings

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de la tool de configuración.
type Toolset struct {
	client api.ReadAPI
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.ReadAPI, cfg config.Config) *Toolset {
	return &Toolset{client: client, cfg: cfg}
}

// Register monta la tool de configuración en el servidor MCP.
func Register(server *mcp.Server, client api.ReadAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "settings_status",
		Description: "Devuelve el estado de configuración e integraciones: horas de proceso, zona horaria, país, " +
			"estado de SMTP, voz, Telegram, webhook y formato de moneda. Omite secretos, emails y chat IDs. Solo lectura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.status)
}

func (t *Toolset) status(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	var raw map[string]any
	if err := t.client.ReadJSON(ctx, "/api/system-settings", &raw); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(toolkit.Project(raw, allowedKeys...))
}

// allowedKeys es la allowlist de campos seguros (SEC-NO-PII).
var allowedKeys = []string{
	"billing_generation_hour",
	"alert_check_hour",
	"timezone",
	"country_code",
	"smtp_host",
	"smtp_port",
	"smtp_from_name",
	"smtp_configured",
	"email_alerts_enabled",
	"voicemonkey_enabled",
	"voicemonkey_send_alerts",
	"voicemonkey_configured",
	"webhook_enabled",
	"webhook_base_url",
	"automation_configured",
	"currency_thousands_separator",
	"currency_decimal_separator",
	"currency_decimal_digits",
	"telegram_bot_enabled",
	"telegram_bot_configured",
	"telegram_bot_chat_ids_count",
	"telegram_bot_separator_length",
	"telegram_bot_show_months",
}
