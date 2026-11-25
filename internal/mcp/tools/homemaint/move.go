package homemaint

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

const maxMoveItems = 100

// moveTargetInput describe la reparación destino que se crea si no existe.
type moveTargetInput struct {
	PerformedAt   *string `json:"performed_at,omitempty" jsonschema:"Fecha YYYY-MM-DD del destino nuevo (obligatoria si no hay target_maintenance_id)"`
	Kind          *string `json:"kind,omitempty" jsonschema:"Tipo del destino: corrective o preventive; por defecto corrective"`
	Area          *string `json:"area,omitempty" jsonschema:"Área del destino; por defecto other"`
	InstitutionID *int64  `json:"institution_id,omitempty" jsonschema:"ID del proveedor o contratista del destino (opcional)"`
	Description   *string `json:"description,omitempty" jsonschema:"Descripción del destino; máximo 500 caracteres"`
}

// moveInput es la entrada de home_maintenance_item_move.
type moveInput struct {
	HomeID              int64            `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	SourceMaintenanceID int64            `json:"source_maintenance_id" jsonschema:"ID de la reparación que contiene las líneas (obligatorio)"`
	ItemIDs             []int64          `json:"item_ids" jsonschema:"IDs de las líneas a mover (1 a 100)"`
	ItemType            *string          `json:"item_type,omitempty" jsonschema:"Salvaguarda opcional: labor o supplies"`
	TargetMaintenanceID *int64           `json:"target_maintenance_id,omitempty" jsonschema:"ID de la reparación destino existente; excluyente con target"`
	Target              *moveTargetInput `json:"target,omitempty" jsonschema:"Datos para crear la reparación destino; excluyente con target_maintenance_id"`
	DeleteSourceIfEmpty *bool            `json:"delete_source_if_empty,omitempty" jsonschema:"Borra el origen si queda sin líneas; por defecto false"`
	DryRun              *bool            `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) registerMove(server *mcp.Server) {
	destructive := true
	notIdempotent := false
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_maintenance_item_move",
		Description: "Mueve líneas de mano de obra y/o insumos de una reparación a otra del mismo hogar. " +
			"El destino puede existir (target_maintenance_id) o crearse en la misma llamada (target). " +
			"Valida que las líneas pertenezcan al origen y no permite mover al mismo mantenimiento. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: notIdempotent, OpenWorldHint: &openWorld},
	}, t.move)
}

func (t *Toolset) move(ctx context.Context, _ *mcp.CallToolRequest, input moveInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.SourceMaintenanceID <= 0 {
		return toolkit.ToolError("invalid_request", "source_maintenance_id debe ser mayor que cero", "")
	}
	if len(input.ItemIDs) == 0 || len(input.ItemIDs) > maxMoveItems {
		return toolkit.ToolError("invalid_request", "item_ids debe tener entre 1 y 100 líneas", "")
	}
	seen := make(map[int64]struct{}, len(input.ItemIDs))
	for _, id := range input.ItemIDs {
		if id <= 0 {
			return toolkit.ToolError("invalid_request", "item_ids solo acepta IDs positivos", "")
		}
		if _, ok := seen[id]; ok {
			return toolkit.ToolError("invalid_request", "item_ids no admite líneas repetidas", "")
		}
		seen[id] = struct{}{}
	}
	itemType := ""
	if input.ItemType != nil && strings.TrimSpace(*input.ItemType) != "" {
		itemType = strings.TrimSpace(*input.ItemType)
		if itemType != itemTypeLabor && itemType != itemTypeSupplies {
			return toolkit.ToolError("invalid_request", "item_type debe ser labor o supplies", "")
		}
	}

	targets := 0
	if input.TargetMaintenanceID != nil {
		targets++
	}
	if input.Target != nil {
		targets++
	}
	if targets == 0 {
		return toolkit.ToolError("target_required", "Indicá target_maintenance_id o target", "Usá home_maintenance_list para buscar la reparación destino o enviá target para crearla.")
	}
	if targets > 1 {
		return toolkit.ToolError("target_conflict", "target_maintenance_id y target son excluyentes", "")
	}
	if input.TargetMaintenanceID != nil {
		if *input.TargetMaintenanceID <= 0 {
			return toolkit.ToolError("invalid_request", "target_maintenance_id debe ser mayor que cero", "")
		}
		if *input.TargetMaintenanceID == input.SourceMaintenanceID {
			return toolkit.ToolError("same_maintenance", "El destino no puede ser la misma reparación de origen", "")
		}
	}

	var targetReq *api.HomeMaintenanceTargetRequest
	if input.Target != nil {
		if input.Target.PerformedAt == nil || strings.TrimSpace(*input.Target.PerformedAt) == "" {
			return toolkit.ToolError("invalid_request", "target.performed_at es obligatoria", "")
		}
		date, err := t.validateDate(*input.Target.PerformedAt)
		if err != nil {
			return invalidToolError(err)
		}
		kind := "corrective"
		if input.Target.Kind != nil && strings.TrimSpace(*input.Target.Kind) != "" {
			kind = strings.TrimSpace(*input.Target.Kind)
			if kind != "corrective" && kind != "preventive" {
				return toolkit.ToolError("invalid_request", "target.kind debe ser corrective o preventive", "")
			}
		}
		area := "other"
		if input.Target.Area != nil && strings.TrimSpace(*input.Target.Area) != "" {
			area = strings.TrimSpace(*input.Target.Area)
			if !contains(validAreas, area) {
				return toolkit.ToolError("invalid_request", "target.area no es válida", "")
			}
		}
		description := ""
		if input.Target.Description != nil {
			description = strings.TrimSpace(*input.Target.Description)
		}
		if utf8.RuneCountInString(description) > maxMaintenanceDescLen {
			return toolkit.ToolError("invalid_request", "target.description no puede superar los 500 caracteres", "")
		}
		institutionID := input.Target.InstitutionID
		if institutionID != nil && *institutionID <= 0 {
			institutionID = nil
		}
		targetReq = &api.HomeMaintenanceTargetRequest{
			PerformedAt:   date,
			Kind:          kind,
			Area:          area,
			InstitutionID: institutionID,
			Description:   description,
		}
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	source, err := t.client.GetHomeMaintenance(ctx, input.HomeID, input.SourceMaintenanceID)
	if err != nil {
		return toolkit.APIError(err)
	}
	moved, remaining, missing, mismatchID := splitMoveItems(source.Items, input.ItemIDs, itemType)
	if len(missing) > 0 {
		return toolkit.ToolError("item_not_found",
			fmt.Sprintf("Estas líneas no pertenecen a la reparación %d: %s", source.ID, idList(missing)),
			"Usá home_maintenance_list con include_items=true para obtener los IDs correctos.")
	}
	if mismatchID > 0 {
		return toolkit.ToolError("invalid_request",
			fmt.Sprintf("La línea %d no es de tipo %s", mismatchID, itemType),
			"Quitá item_type o elegí líneas del mismo tipo.")
	}

	targetSummary := map[string]any{}
	if input.TargetMaintenanceID != nil {
		target, err := t.client.GetHomeMaintenance(ctx, input.HomeID, *input.TargetMaintenanceID)
		if err != nil {
			return toolkit.APIError(err)
		}
		targetSummary = maintenanceSummary(target)
		targetSummary["action"] = "use_existing"
	} else {
		targetSummary = map[string]any{
			"action":       "create_maintenance",
			"performed_at": targetReq.PerformedAt,
			"kind":         targetReq.Kind,
			"area":         targetReq.Area,
			"description":  targetReq.Description,
			"item_count":   0,
		}
		if targetReq.InstitutionID != nil {
			targetSummary["institution_id"] = *targetReq.InstitutionID
		}
	}

	deleteSource := input.DeleteSourceIfEmpty != nil && *input.DeleteSourceIfEmpty
	warnings := make([]string, 0)
	if deleteSource && len(remaining) > 0 {
		warnings = append(warnings, fmt.Sprintf("El origen conserva %d líneas; no se borrará.", len(remaining)))
	}

	if dryRun {
		return toolkit.JSONResult(map[string]any{
			"status":                  "planned",
			"dry_run":                 true,
			"source":                  maintenanceSummary(source),
			"target":                  targetSummary,
			"moved_items":             moved,
			"remaining_items":         remaining,
			"source_item_count_after": len(remaining),
			"would_delete_source":     deleteSource && len(remaining) == 0,
			"warnings":                warnings,
		})
	}

	result, err := t.client.MoveHomeMaintenanceItems(ctx, input.HomeID, input.SourceMaintenanceID, api.HomeMaintenanceMoveRequest{
		ItemIDs:             input.ItemIDs,
		ItemType:            itemType,
		TargetMaintenanceID: input.TargetMaintenanceID,
		Target:              targetReq,
		DeleteSourceIfEmpty: deleteSource,
	})
	if err != nil {
		return toolkit.APIError(err)
	}
	if deleteSource && !result.SourceDeleted && result.SourceItemCount > 0 {
		warnings = append(warnings, fmt.Sprintf("El origen conserva %d líneas; no se borró.", result.SourceItemCount))
	}
	response := map[string]any{
		"status":                "moved",
		"dry_run":               false,
		"source_maintenance_id": result.SourceMaintenanceID,
		"target_maintenance_id": result.TargetMaintenanceID,
		"target_created":        result.TargetCreated,
		"moved_count":           result.MovedCount,
		"moved_items":           result.MovedItems,
		"source_item_count":     result.SourceItemCount,
		"source_deleted":        result.SourceDeleted,
	}
	if result.Target != nil {
		response["target"] = result.Target
	}
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}
	return toolkit.JSONResult(response)
}

// splitMoveItems separa las líneas pedidas de las que quedan en el origen y
// reporta IDs ausentes o de otro tipo.
func splitMoveItems(items []api.HomeMaintenanceItem, ids []int64, itemType string) (moved, remaining []api.HomeMaintenanceItem, missing []int64, mismatchID int64) {
	byID := make(map[int64]api.HomeMaintenanceItem, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	selected := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		selected[id] = struct{}{}
		item, ok := byID[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		if itemType != "" && item.ItemType != itemType {
			mismatchID = item.ID
			continue
		}
		moved = append(moved, item)
	}
	for _, item := range items {
		if _, ok := selected[item.ID]; !ok {
			remaining = append(remaining, item)
		}
	}
	return moved, remaining, missing, mismatchID
}

func maintenanceSummary(item *api.HomeMaintenance) map[string]any {
	return map[string]any{
		"maintenance_id": item.ID,
		"performed_at":   item.PerformedAt,
		"kind":           item.Kind,
		"area":           item.Area,
		"description":    item.Description,
		"item_count":     len(item.Items),
	}
}

func idList(ids []int64) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d", id))
	}
	return strings.Join(parts, ", ")
}
