package workrequest

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// updateInput es la entrada de home_work_request_update. Los campos editables
// son punteros para distinguir "omitir" (conservar) de "enviar".
type updateInput struct {
	HomeID        int64   `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	WorkRequestID int64   `json:"work_request_id" jsonschema:"ID del reporte (obligatorio)"`
	Title         *string `json:"title,omitempty" jsonschema:"Título; máximo 200; omitir conserva"`
	Description   *string `json:"description,omitempty" jsonschema:"Descripción; vacío limpia; máximo 2000; omitir conserva"`
	Category      *string `json:"category,omitempty" jsonschema:"Categoría: repair, improvement, maintenance u other; omitir conserva"`
	Priority      *string `json:"priority,omitempty" jsonschema:"Prioridad: low, normal o high; omitir conserva"`
	ReportedAt    *string `json:"reported_at,omitempty" jsonschema:"Fecha YYYY-MM-DD, no futura; omitir conserva"`
	DryRun        *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) registerUpdate(server *mcp.Server) {
	destructive := true
	idempotent := true
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_work_request_update",
		Description: "Edita un reporte existente. Solo cambia los campos enviados; los omitidos conservan su valor " +
			"y el estado no se modifica. description vacía limpia la descripción. " +
			"Si no hay cambios efectivos devuelve status=unchanged. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: idempotent, OpenWorldHint: &openWorld},
	}, t.update)
}

func (t *Toolset) update(ctx context.Context, _ *mcp.CallToolRequest, input updateInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.WorkRequestID <= 0 {
		return toolkit.ToolError("invalid_request", "work_request_id debe ser mayor que cero", "")
	}
	if input.Title == nil && input.Description == nil && input.Category == nil &&
		input.Priority == nil && input.ReportedAt == nil {
		return toolkit.ToolError("invalid_request", "Enviá al menos un campo para editar", "")
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	current, err := t.client.GetHomeWorkRequest(ctx, input.HomeID, input.WorkRequestID)
	if err != nil {
		return toolkit.APIError(err)
	}
	proposed := workRequestFrom(current)

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return toolkit.ToolError("invalid_request", "title no puede quedar vacío", "")
		}
		if utf8.RuneCountInString(title) > maxTitleLen {
			return toolkit.ToolError("invalid_request", "title no puede superar los 200 caracteres", "")
		}
		proposed.Title = title
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		if utf8.RuneCountInString(description) > maxDescriptionLen {
			return toolkit.ToolError("invalid_request", "description no puede superar los 2000 caracteres", "")
		}
		proposed.Description = description
	}
	if input.Category != nil {
		category := strings.TrimSpace(*input.Category)
		if !contains(validCategories, category) {
			return toolkit.ToolError("invalid_request", "category debe ser repair, improvement, maintenance u other", "")
		}
		proposed.Category = category
	}
	if input.Priority != nil {
		priority := strings.TrimSpace(*input.Priority)
		if !contains(validPriorities, priority) {
			return toolkit.ToolError("invalid_request", "priority debe ser low, normal o high", "")
		}
		proposed.Priority = priority
	}
	if input.ReportedAt != nil {
		reportedAt, err := t.validateDate(*input.ReportedAt)
		if err != nil {
			return invalidToolError(err)
		}
		proposed.ReportedAt = reportedAt
	}

	changed := changedFields(workRequestFrom(current), proposed)
	if len(changed) == 0 {
		return toolkit.JSONResult(map[string]any{
			"status":          "unchanged",
			"dry_run":         dryRun,
			"work_request_id": input.WorkRequestID,
			"changed_fields":  []string{},
			"work_request":    current,
		})
	}

	result := map[string]any{
		"work_request_id": input.WorkRequestID,
		"changed_fields":  changed,
		"dry_run":         dryRun,
	}
	if dryRun {
		result["status"] = "planned"
		result["current"] = requestSnapshot(workRequestFrom(current))
		result["proposed"] = requestSnapshot(proposed)
		return toolkit.JSONResult(result)
	}

	updated, err := t.client.UpdateHomeWorkRequest(ctx, input.HomeID, input.WorkRequestID, proposed)
	if err != nil {
		return toolkit.APIError(err)
	}
	result["status"] = "updated"
	result["work_request"] = updated
	return toolkit.JSONResult(result)
}
