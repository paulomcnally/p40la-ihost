package homemaint

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

const maxMaintenanceDescLen = 500

// createInput es la entrada de home_maintenance_create.
type createInput struct {
	HomeID        int64   `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	PerformedAt   *string `json:"performed_at,omitempty" jsonschema:"Fecha YYYY-MM-DD, no futura (obligatoria)"`
	Kind          *string `json:"kind,omitempty" jsonschema:"Tipo: corrective o preventive; por defecto corrective"`
	Area          *string `json:"area,omitempty" jsonschema:"Área: plumbing, electrical, roof, paint, carpentry, appliances, garden, cleaning u other"`
	InstitutionID *int64  `json:"institution_id,omitempty" jsonschema:"ID del proveedor o contratista (opcional)"`
	Description   *string `json:"description,omitempty" jsonschema:"Descripción libre; máximo 500 caracteres"`
	DryRun        *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) registerCreate(server *mcp.Server) {
	destructive := false
	notIdempotent := false
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_maintenance_create",
		Description: "Crea una reparación o mantenimiento del hogar SIN líneas: 0 insumos y 0 mano de obra. " +
			"Las líneas se agregan después con home_maintenance_labor_register o home_maintenance_supplies_register. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: notIdempotent, OpenWorldHint: &openWorld},
	}, t.create)
}

func (t *Toolset) create(ctx context.Context, _ *mcp.CallToolRequest, input createInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.PerformedAt == nil || strings.TrimSpace(*input.PerformedAt) == "" {
		return toolkit.ToolError("invalid_request", "performed_at es obligatoria", "")
	}
	date, err := t.validateDate(*input.PerformedAt)
	if err != nil {
		return invalidToolError(err)
	}

	kind := "corrective"
	if input.Kind != nil && strings.TrimSpace(*input.Kind) != "" {
		kind = strings.TrimSpace(*input.Kind)
		if kind != "corrective" && kind != "preventive" {
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
	description := ""
	if input.Description != nil {
		description = strings.TrimSpace(*input.Description)
	}
	if utf8.RuneCountInString(description) > maxMaintenanceDescLen {
		return toolkit.ToolError("invalid_request", "description no puede superar los 500 caracteres", "")
	}
	if input.InstitutionID != nil && *input.InstitutionID <= 0 {
		input.InstitutionID = nil
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	planned := map[string]any{
		"action":       "create_maintenance",
		"performed_at": date,
		"kind":         kind,
		"area":         area,
		"description":  description,
		"item_count":   0,
	}
	if input.InstitutionID != nil {
		planned["institution_id"] = *input.InstitutionID
	}
	if dryRun {
		return toolkit.JSONResult(map[string]any{
			"status":  "planned",
			"dry_run": true,
			"planned": planned,
		})
	}

	created, err := t.client.CreateHomeMaintenance(ctx, input.HomeID, api.HomeMaintenanceRequest{
		PerformedAt:   date,
		Kind:          kind,
		Area:          area,
		InstitutionID: input.InstitutionID,
		Description:   description,
	})
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"status":      "created",
		"dry_run":     false,
		"item_count":  len(created.Items),
		"maintenance": created,
	})
}
