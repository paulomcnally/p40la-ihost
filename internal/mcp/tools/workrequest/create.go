package workrequest

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// createInput es la entrada de home_work_request_create.
type createInput struct {
	HomeID      int64   `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	Title       string  `json:"title" jsonschema:"Título del reporte; obligatorio, máximo 200 caracteres"`
	Description *string `json:"description,omitempty" jsonschema:"Detalle del problema o trabajo; máximo 2000 caracteres"`
	Category    *string `json:"category,omitempty" jsonschema:"Categoría: repair, improvement, maintenance u other; por defecto repair"`
	Priority    *string `json:"priority,omitempty" jsonschema:"Prioridad: low, normal o high; por defecto normal"`
	ReportedAt  *string `json:"reported_at,omitempty" jsonschema:"Fecha YYYY-MM-DD, no futura; por defecto hoy"`
	DryRun      *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) registerCreate(server *mcp.Server) {
	destructive := false
	notIdempotent := false
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_work_request_create",
		Description: "Crea un reporte de problema o trabajo deseado en una vivienda. Nace en estado reported, " +
			"sin insumos ni mano de obra, y la evidencia se adjunta después desde la web. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: notIdempotent, OpenWorldHint: &openWorld},
	}, t.create)
}

func (t *Toolset) create(ctx context.Context, _ *mcp.CallToolRequest, input createInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return toolkit.ToolError("invalid_request", "title es obligatorio", "")
	}
	if utf8.RuneCountInString(title) > maxTitleLen {
		return toolkit.ToolError("invalid_request", "title no puede superar los 200 caracteres", "")
	}

	description := ""
	if input.Description != nil {
		description = strings.TrimSpace(*input.Description)
	}
	if utf8.RuneCountInString(description) > maxDescriptionLen {
		return toolkit.ToolError("invalid_request", "description no puede superar los 2000 caracteres", "")
	}

	category := "repair"
	if input.Category != nil && strings.TrimSpace(*input.Category) != "" {
		category = strings.TrimSpace(*input.Category)
		if !contains(validCategories, category) {
			return toolkit.ToolError("invalid_request", "category debe ser repair, improvement, maintenance u other", "")
		}
	}

	priority := "normal"
	if input.Priority != nil && strings.TrimSpace(*input.Priority) != "" {
		priority = strings.TrimSpace(*input.Priority)
		if !contains(validPriorities, priority) {
			return toolkit.ToolError("invalid_request", "priority debe ser low, normal o high", "")
		}
	}

	reportedAt := t.today()
	if input.ReportedAt != nil && strings.TrimSpace(*input.ReportedAt) != "" {
		validated, err := t.validateDate(*input.ReportedAt)
		if err != nil {
			return invalidToolError(err)
		}
		reportedAt = validated
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	request := api.HomeWorkRequestRequest{
		Title:       title,
		Description: description,
		Category:    category,
		Priority:    priority,
		ReportedAt:  reportedAt,
	}
	if dryRun {
		return toolkit.JSONResult(map[string]any{
			"status":  "planned",
			"dry_run": true,
			"planned": requestSnapshot(request),
		})
	}

	created, err := t.client.CreateHomeWorkRequest(ctx, input.HomeID, request)
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"status":       "created",
		"dry_run":      false,
		"work_request": created,
	})
}
