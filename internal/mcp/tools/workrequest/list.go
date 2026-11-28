package workrequest

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// listInput es la entrada de home_work_request_list.
type listInput struct {
	HomeID   int64   `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	Status   *string `json:"status,omitempty" jsonschema:"Filtra por estado: reported, converted o dismissed"`
	Category *string `json:"category,omitempty" jsonschema:"Filtra por categoría: repair, improvement, maintenance u other"`
}

func (t *Toolset) registerList(server *mcp.Server) {
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_work_request_list",
		Description: "Lista los reportes de problemas y trabajos deseados de una vivienda, con filtros opcionales " +
			"de estado y categoría. Devuelve el conteo de pendientes (status=reported) y la cantidad de adjuntos " +
			"de cada reporte. Solo lectura; no aplica la compuerta de escritura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.list)
}

func (t *Toolset) list(ctx context.Context, _ *mcp.CallToolRequest, input listInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	filter := api.HomeWorkRequestFilter{}
	if input.Status != nil && strings.TrimSpace(*input.Status) != "" {
		status := strings.TrimSpace(*input.Status)
		if !contains(validStatuses, status) {
			return toolkit.ToolError("invalid_request", "status debe ser reported, converted o dismissed", "")
		}
		filter.Status = status
	}
	if input.Category != nil && strings.TrimSpace(*input.Category) != "" {
		category := strings.TrimSpace(*input.Category)
		if !contains(validCategories, category) {
			return toolkit.ToolError("invalid_request", "category debe ser repair, improvement, maintenance u other", "")
		}
		filter.Category = category
	}

	result, err := t.client.ListHomeWorkRequests(ctx, input.HomeID, filter)
	if err != nil {
		return toolkit.APIError(err)
	}
	items := result.Items
	if items == nil {
		items = []api.HomeWorkRequest{}
	}
	return toolkit.JSONResult(map[string]any{
		"home_id":       input.HomeID,
		"count":         len(items),
		"pending_count": result.PendingCount,
		"truncated":     result.Truncated,
		"work_requests": items,
	})
}
