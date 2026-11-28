package workrequest

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// getInput es la entrada de home_work_request_get.
type getInput struct {
	HomeID        int64 `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	WorkRequestID int64 `json:"work_request_id" jsonschema:"ID del reporte (obligatorio)"`
}

func (t *Toolset) registerGet(server *mcp.Server) {
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_work_request_get",
		Description: "Devuelve el detalle de un reporte de vivienda: título, descripción, categoría, prioridad, " +
			"estado, fecha, vínculo a la reparación cuando fue convertido y metadatos de su evidencia " +
			"(fotos/videos, sin binarios). Solo lectura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.get)
}

func (t *Toolset) get(ctx context.Context, _ *mcp.CallToolRequest, input getInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.WorkRequestID <= 0 {
		return toolkit.ToolError("invalid_request", "work_request_id debe ser mayor que cero", "")
	}
	item, err := t.client.GetHomeWorkRequest(ctx, input.HomeID, input.WorkRequestID)
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"home_id":          input.HomeID,
		"attachment_count": item.AttachmentCount,
		"work_request":     item,
	})
}
