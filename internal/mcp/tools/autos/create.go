package autos

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
)

// CreateInput es la entrada de autos_create.
type CreateInput struct {
	Year   int64   `json:"year" jsonschema:"Año del auto, entre 1900 y 2100 (obligatorio)"`
	Brand  string  `json:"brand" jsonschema:"Marca del auto (obligatorio)"`
	Model  string  `json:"model" jsonschema:"Modelo del auto (obligatorio)"`
	Color  string  `json:"color" jsonschema:"Color canónico del auto, por ejemplo white o black (obligatorio)"`
	Icon   *string `json:"icon,omitempty" jsonschema:"Ícono del auto; default vehicle"`
	Motor  *string `json:"motor,omitempty" jsonschema:"Número de motor (opcional)"`
	Chasis *string `json:"chasis,omitempty" jsonschema:"Número de chasis (opcional)"`
	VIN    *string `json:"vin,omitempty" jsonschema:"VIN (opcional)"`
	Placa  *string `json:"placa,omitempty" jsonschema:"Placa (opcional)"`
	DryRun *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) create(ctx context.Context, _ *mcp.CallToolRequest, input CreateInput) (*mcp.CallToolResult, any, error) {
	req := autoRequest(input.Year, input.Brand, input.Model, input.Color, input.Icon, input.Motor, input.Chasis, input.VIN, input.Placa)
	if err := validateAutoRequest(req); err != nil {
		return toolError("invalid_request", err.Error(), autoValidationHint())
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	if dryRun {
		planned := api.Auto{
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

	auto, err := t.client.CreateAuto(ctx, req)
	if err != nil {
		return apiError(err)
	}
	return jsonResult(WriteOutput{
		Status: "created",
		DryRun: false,
		Auto:   autoOutput(auto),
	})
}

func autoRequest(year int64, brand, model, color string, icon, motor, chasis, vin, placa *string) api.AutoRequest {
	return api.AutoRequest{
		Year:   year,
		Brand:  strings.TrimSpace(brand),
		Model:  strings.TrimSpace(model),
		Color:  strings.ToLower(strings.TrimSpace(color)),
		Icon:   stringValue(icon, "vehicle"),
		Motor:  stringValue(motor, ""),
		Chasis: stringValue(chasis, ""),
		VIN:    stringValue(vin, ""),
		Placa:  stringValue(placa, ""),
	}
}

func validateAutoRequest(req api.AutoRequest) error {
	if req.Brand == "" {
		return fmt.Errorf("la marca es requerida")
	}
	if req.Model == "" {
		return fmt.Errorf("el modelo es requerido")
	}
	if req.Color == "" {
		return fmt.Errorf("el color es requerido")
	}
	if req.Year < 1900 || req.Year > 2100 {
		return fmt.Errorf("el año debe estar entre 1900 y 2100")
	}
	return nil
}

func autoValidationHint() string {
	return "El color debe ser un código de la paleta canónica (white, black, gray, silver, red, blue, green, yellow, orange, brown, beige, gold, burgundy, light_blue, pink, purple)."
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
