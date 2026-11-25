package homemaint

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

var headerFieldOrder = []string{"performed_at", "kind", "area", "institution_id", "description"}

// updateInput es la entrada de home_maintenance_update. Todos los campos
// editables son punteros para distinguir "omitir" (conservar) de "enviar".
type updateInput struct {
	HomeID        int64   `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	MaintenanceID int64   `json:"maintenance_id" jsonschema:"ID de la reparación (obligatorio)"`
	PerformedAt   *string `json:"performed_at,omitempty" jsonschema:"Fecha YYYY-MM-DD, no futura; omitir conserva"`
	Kind          *string `json:"kind,omitempty" jsonschema:"Tipo: corrective o preventive; omitir conserva"`
	Area          *string `json:"area,omitempty" jsonschema:"Área: plumbing, electrical, roof, paint, carpentry, appliances, garden, cleaning u other; omitir conserva"`
	InstitutionID *int64  `json:"institution_id,omitempty" jsonschema:"ID del proveedor o contratista; 0 limpia; omitir conserva"`
	Description   *string `json:"description,omitempty" jsonschema:"Descripción libre; vacío limpia; máximo 500; omitir conserva"`
	DryRun        *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) registerUpdate(server *mcp.Server) {
	destructive := true
	idempotent := true
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_maintenance_update",
		Description: "Edita una reparación existente. Solo cambia los campos enviados; los omitidos conservan su valor. " +
			"institution_id=0 limpia la institución y description vacía limpia la descripción. " +
			"No cambia las líneas: usá home_maintenance_item_update para eso. " +
			"Si no hay cambios efectivos devuelve status=unchanged. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: idempotent, OpenWorldHint: &openWorld},
	}, t.update)
}

func (t *Toolset) update(ctx context.Context, _ *mcp.CallToolRequest, input updateInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.MaintenanceID <= 0 {
		return toolkit.ToolError("invalid_request", "maintenance_id debe ser mayor que cero", "")
	}
	if input.PerformedAt == nil && input.Kind == nil && input.Area == nil &&
		input.InstitutionID == nil && input.Description == nil {
		return toolkit.ToolError("invalid_request", "Enviá al menos un campo para editar", "")
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	current, err := t.client.GetHomeMaintenance(ctx, input.HomeID, input.MaintenanceID)
	if err != nil {
		return toolkit.APIError(err)
	}
	proposed := headerRequestFrom(current)

	if input.PerformedAt != nil {
		date, err := t.validateDate(*input.PerformedAt)
		if err != nil {
			return invalidToolError(err)
		}
		proposed.PerformedAt = date
	}
	if input.Kind != nil {
		kind := strings.TrimSpace(*input.Kind)
		if kind != "corrective" && kind != "preventive" {
			return toolkit.ToolError("invalid_kind", "kind debe ser corrective o preventive", "")
		}
		proposed.Kind = kind
	}
	if input.Area != nil {
		area := strings.TrimSpace(*input.Area)
		if !contains(validAreas, area) {
			return toolkit.ToolError("invalid_area", "area debe ser plumbing, electrical, roof, paint, carpentry, appliances, garden, cleaning u other", "")
		}
		proposed.Area = area
	}
	if input.InstitutionID != nil {
		switch {
		case *input.InstitutionID < 0:
			return toolkit.ToolError("invalid_request", "institution_id no puede ser negativo; usá 0 para limpiar", "")
		case *input.InstitutionID == 0:
			proposed.InstitutionID = nil
		default:
			proposed.InstitutionID = input.InstitutionID
		}
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		if utf8.RuneCountInString(description) > maxMaintenanceDescLen {
			return toolkit.ToolError("description_too_long", "description no puede superar los 500 caracteres", "")
		}
		proposed.Description = description
	}

	changed := changedFields(headerSnapshot(headerRequestFrom(current)), headerSnapshot(proposed), headerFieldOrder)
	if len(changed) == 0 {
		return toolkit.JSONResult(map[string]any{
			"status":         "unchanged",
			"dry_run":        dryRun,
			"maintenance_id": input.MaintenanceID,
			"changed_fields": []string{},
			"maintenance":    current,
		})
	}

	result := map[string]any{
		"dry_run":        dryRun,
		"maintenance_id": input.MaintenanceID,
		"changed_fields": changed,
	}
	if dryRun {
		result["status"] = "planned"
		result["current"] = headerSnapshot(headerRequestFrom(current))
		result["proposed"] = headerSnapshot(proposed)
		return toolkit.JSONResult(result)
	}

	updated, err := t.client.UpdateHomeMaintenance(ctx, input.HomeID, input.MaintenanceID, proposed)
	if err != nil {
		return toolkit.APIError(err)
	}
	result["status"] = "updated"
	result["maintenance"] = updated
	return toolkit.JSONResult(result)
}

// headerRequestFrom copia la cabecera editable de una reparación.
func headerRequestFrom(maintenance *api.HomeMaintenance) api.HomeMaintenanceUpdateRequest {
	return api.HomeMaintenanceUpdateRequest{
		PerformedAt:   maintenance.PerformedAt,
		Kind:          maintenance.Kind,
		Area:          maintenance.Area,
		InstitutionID: maintenance.InstitutionID,
		Description:   maintenance.Description,
	}
}

// headerSnapshot normaliza la cabecera para comparar y previsualizar.
func headerSnapshot(request api.HomeMaintenanceUpdateRequest) map[string]any {
	snapshot := map[string]any{
		"performed_at": request.PerformedAt,
		"kind":         request.Kind,
		"area":         request.Area,
		"description":  request.Description,
	}
	if request.InstitutionID != nil {
		snapshot["institution_id"] = *request.InstitutionID
	} else {
		snapshot["institution_id"] = nil
	}
	return snapshot
}

// changedFields devuelve los campos que difieren, en el orden indicado.
func changedFields(current, proposed map[string]any, order []string) []string {
	changed := make([]string, 0, len(order))
	for _, field := range order {
		if !valuesEqual(current[field], proposed[field]) {
			changed = append(changed, field)
		}
	}
	return changed
}

// valuesEqual compara valores normalizados de un snapshot.
func valuesEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a == b
}

// int64PtrEqual compara dos punteros a int64 por valor.
func int64PtrEqual(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
