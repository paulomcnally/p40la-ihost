// Package homemaint implementa la tool MCP de solo lectura de reparaciones y
// mantenimientos del hogar (SPEC-154, Ola 1).
package homemaint

import (
	"context"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de la tool de reparaciones.
type Toolset struct {
	client api.ReadAPI
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.ReadAPI, cfg config.Config) *Toolset {
	return &Toolset{client: client, cfg: cfg}
}

// Register monta la tool de reparaciones en el servidor MCP.
func Register(server *mcp.Server, client api.ReadAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_maintenance_list",
		Description: "Lista las reparaciones y mantenimientos de una vivienda con totales por moneda " +
			"(mano de obra e insumos). Filtros opcionales de año y tipo (corrective, preventive). Solo lectura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.list)
}

type listInput struct {
	HomeID int64   `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	Year   *int    `json:"year,omitempty" jsonschema:"Filtra por año (1900-9999)"`
	Kind   *string `json:"kind,omitempty" jsonschema:"Tipo: corrective o preventive"`
}

func (t *Toolset) list(ctx context.Context, _ *mcp.CallToolRequest, input listInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	query := url.Values{}
	if input.Year != nil {
		if *input.Year < 1900 || *input.Year > 9999 {
			return toolkit.ToolError("invalid_request", "year debe estar entre 1900 y 9999", "")
		}
		query.Set("year", fmt.Sprintf("%d", *input.Year))
	}
	if input.Kind != nil && *input.Kind != "" {
		switch *input.Kind {
		case "corrective", "preventive":
			query.Set("kind", *input.Kind)
		default:
			return toolkit.ToolError("invalid_request", "kind debe ser corrective o preventive", "")
		}
	}
	var items []map[string]any
	if err := t.client.ReadJSON(ctx, toolkit.WithQuery(fmt.Sprintf("/api/homes/%d/maintenances", input.HomeID), query), &items); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"home_id":      input.HomeID,
		"count":        len(items),
		"maintenances": items,
	})
}
