package workrequest

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// convertInput es la entrada de home_work_request_convert.
type convertInput struct {
	HomeID        int64   `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	WorkRequestID int64   `json:"work_request_id" jsonschema:"ID del reporte (obligatorio)"`
	Kind          *string `json:"kind,omitempty" jsonschema:"Tipo de reparación: corrective o preventive; por defecto corrective"`
	Area          *string `json:"area,omitempty" jsonschema:"Área: plumbing, electrical, roof, paint, carpentry, appliances, garden, cleaning u other; por defecto other"`
	DryRun        *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) registerConvert(server *mcp.Server) {
	destructive := true
	notIdempotent := false
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_work_request_convert",
		Description: "Convierte un reporte pendiente (status=reported) en una reparación vacía de home.maintenance, " +
			"copia su título/descripción, enlaza maintenance_id y deja el reporte en status=converted. " +
			"La evidencia permanece en el reporte. Una segunda conversión devuelve already_converted. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: notIdempotent, OpenWorldHint: &openWorld},
	}, t.convert)
}

func (t *Toolset) convert(ctx context.Context, _ *mcp.CallToolRequest, input convertInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.WorkRequestID <= 0 {
		return toolkit.ToolError("invalid_request", "work_request_id debe ser mayor que cero", "")
	}

	kind := "corrective"
	if input.Kind != nil && strings.TrimSpace(*input.Kind) != "" {
		kind = strings.TrimSpace(*input.Kind)
		if !contains(validKinds, kind) {
			return toolkit.ToolError("invalid_request", "kind debe ser corrective o preventive", "")
		}
	}
	area := "other"
	if input.Area != nil && strings.TrimSpace(*input.Area) != "" {
		area = strings.TrimSpace(*input.Area)
		if !contains(validAreas, area) {
			return toolkit.ToolError("invalid_request", "area debe ser plumbing, electrical, roof, paint, carpentry, appliances, garden, cleaning u other", "")
		}
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
		return toolkit.ToolError("already_converted", "El reporte ya fue convertido a reparación", "")
	}
	if current.Status != "reported" {
		return toolkit.ToolError("invalid_status", "Solo un reporte pendiente se puede convertir", "")
	}

	if dryRun {
		return toolkit.JSONResult(map[string]any{
			"status":  "planned",
			"dry_run": true,
			"planned": map[string]any{
				"action":          "convert_work_request",
				"work_request_id": current.ID,
				"title":           current.Title,
				"kind":            kind,
				"area":            area,
				"maintenance_id":  current.MaintenanceID,
			},
		})
	}

	result, err := t.client.ConvertHomeWorkRequest(ctx, input.HomeID, input.WorkRequestID, api.HomeWorkRequestConvertRequest{
		Kind: kind,
		Area: area,
	})
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"status":         "converted",
		"dry_run":        false,
		"maintenance_id": result.MaintenanceID,
		"work_request":   result.WorkRequest,
	})
}
