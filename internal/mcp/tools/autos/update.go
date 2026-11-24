package autos

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

// UpdateInput es la entrada de autos_update.
type UpdateInput struct {
	AutoID int64   `json:"auto_id" jsonschema:"ID del auto a actualizar (obligatorio)"`
	Year   int64   `json:"year" jsonschema:"Año del auto, entre 1900 y 2100 (obligatorio)"`
	Brand  string  `json:"brand" jsonschema:"Marca del auto (obligatorio)"`
	Model  string  `json:"model" jsonschema:"Modelo del auto (obligatorio)"`
	Color  string  `json:"color" jsonschema:"Color canónico del auto (obligatorio)"`
	Icon   *string `json:"icon,omitempty" jsonschema:"Ícono del auto; default vehicle"`
	Motor  *string `json:"motor,omitempty" jsonschema:"Número de motor; vacío para limpiarlo"`
	Chasis *string `json:"chasis,omitempty" jsonschema:"Número de chasis; vacío para limpiarlo"`
	VIN    *string `json:"vin,omitempty" jsonschema:"VIN; vacío para limpiarlo"`
	Placa  *string `json:"placa,omitempty" jsonschema:"Placa; vacío para limpiarla"`
	DryRun *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) update(ctx context.Context, _ *mcp.CallToolRequest, input UpdateInput) (*mcp.CallToolResult, any, error) {
	if input.AutoID <= 0 {
		return toolError("invalid_request", "auto_id debe ser mayor que cero", "")
	}
	req := autoRequest(input.Year, input.Brand, input.Model, input.Color, input.Icon, input.Motor, input.Chasis, input.VIN, input.Placa)
	if err := validateAutoRequest(req); err != nil {
		return toolError("invalid_request", err.Error(), autoValidationHint())
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	if dryRun {
		if _, err := t.client.GetAuto(ctx, input.AutoID); err != nil {
			return apiError(err)
		}
		planned := api.Auto{
			ID:     input.AutoID,
			Year:   req.Year,
			Brand:  req.Brand,
			Model:  req.Model,
			Color:  req.Color,
			Icon:   req.Icon,
			Motor:  req.Motor,
			Chasis: req.Chasis,
			VIN:    req.VIN,
			Placa:  req.Placa,
		}
		return jsonResult(WriteOutput{
			Status:  "planned",
			DryRun:  true,
			Planned: autoOutput(&planned),
		})
	}

	auto, err := t.client.UpdateAuto(ctx, input.AutoID, req)
	if err != nil {
		return apiError(err)
	}
	return jsonResult(WriteOutput{
		Status: "updated",
		DryRun: false,
		Auto:   autoOutput(auto),
	})
}
