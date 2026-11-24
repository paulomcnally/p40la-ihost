package autos

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

const maxMaintenanceItems = 20

// MaintenanceItemInput es una línea de mano de obra o insumos.
type MaintenanceItemInput struct {
	ItemType    string  `json:"item_type" jsonschema:"Tipo de línea: labor (mano de obra) o supplies (insumos)"`
	Description string  `json:"description" jsonschema:"Descripción de la línea"`
	Amount      float64 `json:"amount" jsonschema:"Monto de la línea en la moneda del mantenimiento"`
}

// MaintenanceInput es la entrada de autos_maintenance_register.
type MaintenanceInput struct {
	AutoID        int64                  `json:"auto_id" jsonschema:"ID del auto (obligatorio)"`
	PerformedAt   string                 `json:"performed_at" jsonschema:"Fecha del mantenimiento YYYY-MM-DD, no futura (obligatorio)"`
	Mileage       int64                  `json:"mileage" jsonschema:"Kilometraje al momento del mantenimiento (obligatorio)"`
	InstitutionID int64                  `json:"institution_id" jsonschema:"ID del taller mecánico (obligatorio)"`
	CurrencyID    int64                  `json:"currency_id" jsonschema:"ID de la moneda (obligatorio)"`
	Description   *string                `json:"description,omitempty" jsonschema:"Descripción del mantenimiento (opcional)"`
	Items         []MaintenanceItemInput `json:"items,omitempty" jsonschema:"Líneas opcionales de mano de obra e insumos (máximo 20)"`
	DryRun        *bool                  `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) maintenanceRegister(ctx context.Context, _ *mcp.CallToolRequest, input MaintenanceInput) (*mcp.CallToolResult, any, error) {
	performedAt, err := t.validateDate(input.PerformedAt, "mantenimiento")
	if err != nil {
		return toolError("invalid_date", err.Error(), "Usá una fecha YYYY-MM-DD que no sea futura.")
	}
	if input.AutoID <= 0 {
		return toolError("invalid_request", "auto_id debe ser mayor que cero", "")
	}
	if input.Mileage < 0 {
		return toolError("invalid_request", "el kilometraje no puede ser negativo", "")
	}
	if input.InstitutionID <= 0 {
		return toolError("invalid_request", "institution_id es obligatorio", "Usá el ID de un taller mecánico.")
	}
	if input.CurrencyID <= 0 {
		return toolError("invalid_request", "currency_id es obligatorio", "Consultá las monedas configuradas en el sistema.")
	}
	if len(input.Items) > maxMaintenanceItems {
		return toolError("invalid_request", fmt.Sprintf("el máximo es %d líneas por mantenimiento", maxMaintenanceItems), "")
	}
	for i, item := range input.Items {
		if err := validateMaintenanceItem(item); err != nil {
			return toolError("invalid_request", fmt.Sprintf("línea %d: %s", i+1, err.Error()), "")
		}
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	req := api.MaintenanceRequest{
		PerformedAt:   performedAt,
		Mileage:       input.Mileage,
		InstitutionID: input.InstitutionID,
		Description:   stringValue(input.Description, ""),
		CurrencyID:    input.CurrencyID,
	}

	if dryRun {
		if _, err := t.client.GetAuto(ctx, input.AutoID); err != nil {
			return apiError(err)
		}
		return jsonResult(WriteOutput{
			Status: "planned",
			DryRun: true,
			Planned: map[string]any{
				"auto_id":     input.AutoID,
				"maintenance": req,
				"items":       input.Items,
			},
		})
	}

	maintenance, err := t.client.CreateMaintenance(ctx, input.AutoID, req)
	if err != nil {
		return apiError(err)
	}

	result := WriteOutput{
		Status:      "created",
		DryRun:      false,
		Maintenance: maintenance,
		Items:       []api.MaintenanceItem{},
	}
	for i, item := range input.Items {
		created, err := t.client.CreateMaintenanceItem(ctx, input.AutoID, maintenance.ID, api.MaintenanceItemRequest{
			ItemType:    strings.TrimSpace(item.ItemType),
			Description: strings.TrimSpace(item.Description),
			Amount:      item.Amount,
		})
		if err != nil {
			result.Status = "created_partial"
			result.Partial = true
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("la línea %d no se pudo crear: %s", i+1, err.Error()))
			break
		}
		result.Items = append(result.Items, *created)
	}

	if detail, err := t.client.GetMaintenance(ctx, input.AutoID, maintenance.ID); err == nil && detail != nil {
		result.Maintenance = detail
	}
	return jsonResult(result)
}

func validateMaintenanceItem(item MaintenanceItemInput) error {
	itemType := strings.TrimSpace(item.ItemType)
	if itemType != "labor" && itemType != "supplies" {
		return fmt.Errorf("item_type debe ser labor o supplies")
	}
	if strings.TrimSpace(item.Description) == "" {
		return fmt.Errorf("la descripción es requerida")
	}
	if item.Amount < 0 {
		return fmt.Errorf("el monto no puede ser negativo")
	}
	return nil
}

// validateDate valida una fecha YYYY-MM-DD y que no sea futura.
func (t *Toolset) validateDate(value, label string) (string, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("la fecha del %s debe tener formato YYYY-MM-DD", label)
	}
	today, err := time.Parse("2006-01-02", t.today())
	if err != nil {
		return "", fmt.Errorf("no se pudo resolver la fecha actual")
	}
	if parsed.After(today) {
		return "", fmt.Errorf("la fecha del %s no puede ser futura", label)
	}
	return parsed.Format("2006-01-02"), nil
}
