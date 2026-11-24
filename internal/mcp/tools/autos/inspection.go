package autos

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

// InspectionInput es la entrada de autos_inspection_register.
type InspectionInput struct {
	AutoID         int64   `json:"auto_id" jsonschema:"ID del auto (obligatorio)"`
	PerformedAt    string  `json:"performed_at" jsonschema:"Fecha de la inspección YYYY-MM-DD, no futura (obligatorio)"`
	InspectionYear int64   `json:"inspection_year" jsonschema:"Año del ciclo que cumple la inspección (obligatorio)"`
	InstitutionID  int64   `json:"institution_id" jsonschema:"ID del centro de inspección mecánica (obligatorio)"`
	Result         string  `json:"result" jsonschema:"Resultado: approved o rejected (obligatorio)"`
	Notes          *string `json:"notes,omitempty" jsonschema:"Notas de la inspección (opcional)"`
	DryRun         *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) inspectionRegister(ctx context.Context, _ *mcp.CallToolRequest, input InspectionInput) (*mcp.CallToolResult, any, error) {
	performedAt, err := t.validateDate(input.PerformedAt, "inspección")
	if err != nil {
		return toolError("invalid_date", err.Error(), "Usá una fecha YYYY-MM-DD que no sea futura.")
	}
	if input.AutoID <= 0 {
		return toolError("invalid_request", "auto_id debe ser mayor que cero", "")
	}
	result := strings.ToLower(strings.TrimSpace(input.Result))
	if result != "approved" && result != "rejected" {
		return toolError("invalid_request", "el resultado debe ser approved o rejected", "")
	}
	if input.InspectionYear <= 0 {
		return toolError("invalid_request", "inspection_year es obligatorio", "")
	}
	if input.InstitutionID <= 0 {
		return toolError("invalid_request", "institution_id es obligatorio", "Usá el ID de un centro de inspección mecánica.")
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	req := api.InspectionRequest{
		PerformedAt:    performedAt,
		InspectionYear: input.InspectionYear,
		InstitutionID:  input.InstitutionID,
		Result:         result,
		Notes:          stringValue(input.Notes, ""),
	}

	if dryRun {
		if _, err := t.client.GetAuto(ctx, input.AutoID); err != nil {
			return apiError(err)
		}
		return jsonResult(WriteOutput{
			Status:  "planned",
			DryRun:  true,
			Planned: req,
		})
	}

	inspection, err := t.client.CreateInspection(ctx, input.AutoID, req)
	if err != nil {
		return apiError(err)
	}

	outputFull := WriteOutput{
		Status:     "created",
		DryRun:     false,
		Inspection: inspection,
	}
	if list, err := t.client.ListInspections(ctx, input.AutoID); err == nil && list != nil {
		schedule := list.Schedule
		outputFull.Schedule = &schedule
	} else if err != nil {
		outputFull.Warnings = append(outputFull.Warnings,
			"la inspección se creó pero no se pudo recalcular el cronograma: "+err.Error())
	}
	return jsonResult(outputFull)
}
