// Package fleet implementa la tool MCP de solo lectura de alertas de
// inspección de la flota de autos (SPEC-154, Ola 1).
package fleet

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de la tool de alertas de inspección.
type Toolset struct {
	client api.ReadAPI
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.ReadAPI, cfg config.Config) *Toolset {
	return &Toolset{client: client, cfg: cfg}
}

// Register monta la tool de alertas de inspección en el servidor MCP.
func Register(server *mcp.Server, client api.ReadAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "autos_inspection_alerts",
		Description: "Devuelve el estado de inspección de toda la flota: al día, por vencer (≤ 60 días) " +
			"y vencidas. Solo lectura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.alerts)
}

func (t *Toolset) alerts(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	var alerts any
	if err := t.client.ReadJSON(ctx, "/api/autos/inspection-alerts", &alerts); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(alerts)
}
