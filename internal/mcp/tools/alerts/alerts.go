// Package alerts implementa la tool MCP de solo lectura del catálogo de
// alertas y sus canales habilitados (SPEC-154, Ola 1).
package alerts

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de la tool de alertas.
type Toolset struct {
	client api.ReadAPI
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.ReadAPI, cfg config.Config) *Toolset {
	return &Toolset{client: client, cfg: cfg}
}

// Register monta la tool de alertas en el servidor MCP.
func Register(server *mcp.Server, client api.ReadAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "alerts_list",
		Description: "Lista las alertas del sistema con sus canales habilitados (email, voz, Telegram). " +
			"Omite el texto de voz porque contiene un nombre personal. Solo lectura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.list)
}

func (t *Toolset) list(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	var items []map[string]any
	if err := t.client.ReadJSON(ctx, "/api/alerts", &items); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"count":  len(items),
		"alerts": toolkit.OmitAll(items, "speech"),
	})
}
