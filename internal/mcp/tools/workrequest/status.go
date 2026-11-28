package workrequest

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// statusInput es la entrada de home_work_request_status.
type statusInput struct {
	HomeID        int64  `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	WorkRequestID int64  `json:"work_request_id" jsonschema:"ID del reporte (obligatorio)"`
	Status        string `json:"status" jsonschema:"Estado destino: dismissed para descartar o reported para reactivar"`
	DryRun        *bool  `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) registerStatus(server *mcp.Server) {
	destructive := true
	idempotent := true
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_work_request_status",
		Description: "Descarta un reporte pendiente (status=dismissed) o lo reactiva (status=reported). " +
			"No admite converted ni otros estados y no crea ni borra reparaciones. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: idempotent, OpenWorldHint: &openWorld},
	}, t.status)
}

func (t *Toolset) status(ctx context.Context, _ *mcp.CallToolRequest, input statusInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.WorkRequestID <= 0 {
		return toolkit.ToolError("invalid_request", "work_request_id debe ser mayor que cero", "")
	}
	target := strings.TrimSpace(input.Status)
	if target != "reported" && target != "dismissed" {
		return toolkit.ToolError("invalid_status", "status debe ser reported o dismissed", "")
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	current, err := t.client.GetHomeWorkRequest(ctx, input.HomeID, input.WorkRequestID)
	if err != nil {
		return toolkit.APIError(err)
	}
	if current.Status == "converted" {
		return toolkit.ToolError("invalid_status", "Un reporte convertido no cambia de estado", "")
	}
	if current.Status == target {
		return toolkit.JSONResult(map[string]any{
			"status":          "unchanged",
			"dry_run":         dryRun,
			"work_request_id": current.ID,
			"work_request":    current,
		})
	}

	result := map[string]any{
		"work_request_id": current.ID,
		"dry_run":         dryRun,
		"from_status":     current.Status,
		"to_status":       target,
	}
	if dryRun {
		result["status"] = "planned"
		return toolkit.JSONResult(result)
	}

	updated, err := t.client.SetHomeWorkRequestStatus(ctx, input.HomeID, input.WorkRequestID, api.HomeWorkRequestStatusRequest{Status: target})
	if err != nil {
		return toolkit.APIError(err)
	}
	result["status"] = "updated"
	result["work_request"] = updated
	return toolkit.JSONResult(result)
}
