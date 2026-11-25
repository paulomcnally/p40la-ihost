package homemaint

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// itemDeleteInput es la entrada de home_maintenance_item_delete.
type itemDeleteInput struct {
	HomeID        int64 `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	MaintenanceID int64 `json:"maintenance_id" jsonschema:"ID de la reparación que contiene la línea (obligatorio)"`
	ItemID        int64 `json:"item_id" jsonschema:"ID de la línea a borrar (obligatorio)"`
	DryRun        *bool `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) registerItemDelete(server *mcp.Server) {
	destructive := true
	idempotent := true
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_maintenance_item_delete",
		Description: "Borra una línea de mano de obra o insumos de una reparación. No borra la cabecera: " +
			"si era la última línea, la reparación queda vacía y se informa maintenance_empty=true. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: idempotent, OpenWorldHint: &openWorld},
	}, t.itemDelete)
}

func (t *Toolset) itemDelete(ctx context.Context, _ *mcp.CallToolRequest, input itemDeleteInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.MaintenanceID <= 0 {
		return toolkit.ToolError("invalid_request", "maintenance_id debe ser mayor que cero", "")
	}
	if input.ItemID <= 0 {
		return toolkit.ToolError("invalid_request", "item_id debe ser mayor que cero", "")
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	maintenance, err := t.client.GetHomeMaintenance(ctx, input.HomeID, input.MaintenanceID)
	if err != nil {
		return toolkit.APIError(err)
	}
	current := findMaintenanceItem(maintenance.Items, input.ItemID)
	if current == nil {
		return toolkit.ToolError("item_not_found",
			"La línea no pertenece a la reparación indicada",
			"Usá home_maintenance_get o home_maintenance_list con include_items=true para obtener los IDs correctos.")
	}

	remaining := len(maintenance.Items) - 1
	if remaining < 0 {
		remaining = 0
	}
	maintenanceEmpty := remaining == 0
	if dryRun {
		dryRunWarnings := make([]string, 0)
		if maintenanceEmpty {
			dryRunWarnings = append(dryRunWarnings, "La reparación quedará sin líneas. home_maintenance_item_delete no borra la cabecera.")
		}
		return toolkit.JSONResult(map[string]any{
			"status":               "planned",
			"dry_run":              true,
			"maintenance_id":       input.MaintenanceID,
			"item_id":              input.ItemID,
			"would_delete":         itemDeleteSummary(current),
			"remaining_item_count": remaining,
			"maintenance_empty":    maintenanceEmpty,
			"totals_by_currency":   maintenance.TotalsByCurrency,
			"warnings":             dryRunWarnings,
		})
	}

	if err := t.client.DeleteHomeMaintenanceItem(ctx, input.HomeID, input.MaintenanceID, input.ItemID); err != nil {
		return toolkit.APIError(err)
	}

	warnings := make([]string, 0)
	if maintenanceEmpty {
		warnings = append(warnings, "La reparación quedó sin líneas. home_maintenance_item_delete no borra la cabecera; usá home_maintenance_create o mové líneas si querés conservarla.")
	}
	result := map[string]any{
		"status":               "deleted",
		"dry_run":              false,
		"maintenance_id":       input.MaintenanceID,
		"item_id":              input.ItemID,
		"remaining_item_count": remaining,
		"maintenance_empty":    maintenanceEmpty,
		"warnings":             warnings,
	}

	refreshed, err := t.client.GetHomeMaintenance(ctx, input.HomeID, input.MaintenanceID)
	if err != nil {
		warnings = append(warnings, "La línea se borró, pero no se pudieron recalcular los totales: "+err.Error())
		result["warnings"] = warnings
		result["totals_by_currency"] = []api.HomeMaintenanceCurrencyTotal{}
		return toolkit.JSONResult(result)
	}
	result["remaining_item_count"] = len(refreshed.Items)
	result["maintenance_empty"] = len(refreshed.Items) == 0
	result["totals_by_currency"] = refreshed.TotalsByCurrency
	return toolkit.JSONResult(result)
}

// itemDeleteSummary resume la línea que se borraría en dry_run.
func itemDeleteSummary(item *api.HomeMaintenanceItem) map[string]any {
	return map[string]any{
		"id":            item.ID,
		"item_type":     item.ItemType,
		"description":   item.Description,
		"quantity":      item.Quantity,
		"unit":          item.Unit,
		"amount":        item.Amount,
		"currency_id":   item.CurrencyID,
		"currency_code": item.CurrencyCode,
	}
}
