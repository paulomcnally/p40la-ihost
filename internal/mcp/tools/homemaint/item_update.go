package homemaint

import (
	"context"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

var itemFieldOrder = []string{"item_type", "description", "quantity", "unit", "amount", "currency_id", "institution_id"}

// itemUpdateInput es la entrada de home_maintenance_item_update.
type itemUpdateInput struct {
	HomeID        int64    `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	MaintenanceID int64    `json:"maintenance_id" jsonschema:"ID de la reparación que contiene la línea (obligatorio)"`
	ItemID        int64    `json:"item_id" jsonschema:"ID de la línea a editar (obligatorio)"`
	ItemType      *string  `json:"item_type,omitempty" jsonschema:"Tipo: labor o supplies; omitir conserva"`
	Description   *string  `json:"description,omitempty" jsonschema:"Descripción de la línea (máximo 200); omitir conserva"`
	Quantity      *float64 `json:"quantity,omitempty" jsonschema:"Cantidad para insumos (mayor que cero); labor fuerza 1; omitir conserva"`
	Unit          *string  `json:"unit,omitempty" jsonschema:"Unidad de insumos; labor fuerza unit; omitir conserva"`
	Amount        *float64 `json:"amount,omitempty" jsonschema:"Monto de la línea (mayor o igual a 0); omitir conserva"`
	CurrencyID    *int64   `json:"currency_id,omitempty" jsonschema:"ID de la moneda; omitir conserva"`
	InstitutionID *int64   `json:"institution_id,omitempty" jsonschema:"ID del proveedor o contratista; 0 limpia; omitir conserva"`
	DryRun        *bool    `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) registerItemUpdate(server *mcp.Server) {
	destructive := true
	idempotent := true
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_maintenance_item_update",
		Description: "Edita una línea de mano de obra o insumos de una reparación. Solo cambia los campos enviados; " +
			"los omitidos conservan su valor. Cambiar item_type a labor fuerza quantity=1 y unit=unit. " +
			"institution_id=0 limpia el proveedor. Devuelve la línea y los totales por moneda recalculados. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: idempotent, OpenWorldHint: &openWorld},
	}, t.itemUpdate)
}

func (t *Toolset) itemUpdate(ctx context.Context, _ *mcp.CallToolRequest, input itemUpdateInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	if input.MaintenanceID <= 0 {
		return toolkit.ToolError("invalid_request", "maintenance_id debe ser mayor que cero", "")
	}
	if input.ItemID <= 0 {
		return toolkit.ToolError("invalid_request", "item_id debe ser mayor que cero", "")
	}
	if input.ItemType == nil && input.Description == nil && input.Quantity == nil && input.Unit == nil &&
		input.Amount == nil && input.CurrencyID == nil && input.InstitutionID == nil {
		return toolkit.ToolError("invalid_request", "Enviá al menos un campo para editar", "")
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
	proposed := itemRequestFrom(current)

	if input.ItemType != nil {
		itemType := strings.TrimSpace(*input.ItemType)
		if itemType != itemTypeLabor && itemType != itemTypeSupplies {
			return toolkit.ToolError("invalid_item_type", "item_type debe ser labor o supplies", "")
		}
		proposed.ItemType = itemType
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		if description == "" {
			return toolkit.ToolError("item_description_required", "La descripción de la línea es obligatoria", "")
		}
		if utf8.RuneCountInString(description) > maxItemDescLen {
			return toolkit.ToolError("item_description_too_long", "description no puede superar los 200 caracteres", "")
		}
		proposed.Description = description
	}
	if input.Amount != nil {
		if math.IsNaN(*input.Amount) || math.IsInf(*input.Amount, 0) || *input.Amount < 0 {
			return toolkit.ToolError("invalid_amount", "amount debe ser un número mayor o igual a 0", "")
		}
		proposed.Amount = roundMoney(*input.Amount)
	}
	if input.CurrencyID != nil {
		if *input.CurrencyID <= 0 {
			return toolkit.ToolError("currency_required", "currency_id debe ser mayor que cero", "")
		}
		proposed.CurrencyID = *input.CurrencyID
	}
	if input.Quantity != nil {
		if math.IsNaN(*input.Quantity) || math.IsInf(*input.Quantity, 0) || *input.Quantity <= 0 {
			return toolkit.ToolError("invalid_quantity", "quantity debe ser mayor que cero", "")
		}
		proposed.Quantity = *input.Quantity
	}
	if input.Unit != nil {
		unit := strings.TrimSpace(*input.Unit)
		if !contains(validUnits, unit) {
			return toolkit.ToolError("invalid_unit", "unit no es una unidad válida", "")
		}
		proposed.Unit = unit
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
	if proposed.ItemType == itemTypeLabor {
		proposed.Quantity = 1
		proposed.Unit = unitDefault
	} else {
		if proposed.Quantity <= 0 {
			return toolkit.ToolError("invalid_quantity", "quantity debe ser mayor que cero", "")
		}
		if proposed.Unit == "" {
			proposed.Unit = unitDefault
		}
		if !contains(validUnits, proposed.Unit) {
			return toolkit.ToolError("invalid_unit", "unit no es una unidad válida", "")
		}
	}

	changed := changedFields(itemSnapshot(itemRequestFrom(current)), itemSnapshot(proposed), itemFieldOrder)
	if len(changed) == 0 {
		return toolkit.JSONResult(map[string]any{
			"status":         "unchanged",
			"dry_run":        dryRun,
			"maintenance_id": input.MaintenanceID,
			"item_id":        input.ItemID,
			"changed_fields": []string{},
			"item":           current,
		})
	}

	result := map[string]any{
		"dry_run":        dryRun,
		"maintenance_id": input.MaintenanceID,
		"item_id":        input.ItemID,
		"changed_fields": changed,
	}
	if dryRun {
		result["status"] = "planned"
		result["current"] = itemSnapshot(itemRequestFrom(current))
		result["proposed"] = itemSnapshot(proposed)
		return toolkit.JSONResult(result)
	}

	updated, err := t.client.UpdateHomeMaintenanceItem(ctx, input.HomeID, input.MaintenanceID, input.ItemID, proposed)
	if err != nil {
		return toolkit.APIError(err)
	}
	result["status"] = "updated"
	result["item"] = updated
	refreshed, err := t.client.GetHomeMaintenance(ctx, input.HomeID, input.MaintenanceID)
	if err != nil {
		result["warnings"] = []string{"La línea se actualizó, pero no se pudieron recalcular los totales: " + err.Error()}
		return toolkit.JSONResult(result)
	}
	result["totals_by_currency"] = refreshed.TotalsByCurrency
	return toolkit.JSONResult(result)
}

// findMaintenanceItem devuelve la línea por ID dentro de una reparación.
func findMaintenanceItem(items []api.HomeMaintenanceItem, id int64) *api.HomeMaintenanceItem {
	for index := range items {
		if items[index].ID == id {
			return &items[index]
		}
	}
	return nil
}

// itemRequestFrom copia los campos editables de una línea.
func itemRequestFrom(item *api.HomeMaintenanceItem) api.HomeMaintenanceItemRequest {
	return api.HomeMaintenanceItemRequest{
		ItemType:      item.ItemType,
		Description:   item.Description,
		Quantity:      item.Quantity,
		Unit:          item.Unit,
		Amount:        item.Amount,
		CurrencyID:    item.CurrencyID,
		InstitutionID: item.InstitutionID,
	}
}

// itemSnapshot normaliza una línea para comparar y previsualizar.
func itemSnapshot(request api.HomeMaintenanceItemRequest) map[string]any {
	snapshot := map[string]any{
		"item_type":   request.ItemType,
		"description": request.Description,
		"quantity":    request.Quantity,
		"unit":        request.Unit,
		"amount":      request.Amount,
		"currency_id": request.CurrencyID,
	}
	if request.InstitutionID != nil {
		snapshot["institution_id"] = *request.InstitutionID
	} else {
		snapshot["institution_id"] = nil
	}
	return snapshot
}

// roundMoney redondea a dos decimales como el backend.
func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}
