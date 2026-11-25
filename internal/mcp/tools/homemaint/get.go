package homemaint

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// getInput es la entrada de home_maintenance_get.
type getInput struct {
	HomeID        int64 `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	MaintenanceID int64 `json:"maintenance_id" jsonschema:"ID de la reparación (obligatorio)"`
}

func (t *Toolset) registerGet(server *mcp.Server) {
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_maintenance_get",
		Description: "Devuelve una reparación o mantenimiento del hogar con sus líneas de mano de obra e insumos " +
			"y los totales por moneda. Es el detalle completo, sin el tope de 100 reparaciones de " +
			"home_maintenance_list. Solo lectura; no aplica la compuerta de escritura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.get)
}

func (t *Toolset) get(ctx context.Context, _ *mcp.CallToolRequest, input getInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.MaintenanceID <= 0 {
		return toolkit.ToolError("invalid_request", "maintenance_id debe ser mayor que cero", "")
	}
	maintenance, err := t.client.GetHomeMaintenance(ctx, input.HomeID, input.MaintenanceID)
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"home_id":     input.HomeID,
		"item_count":  len(maintenance.Items),
		"maintenance": maintenance,
	})
}
