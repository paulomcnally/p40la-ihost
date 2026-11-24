package bills

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// GenerateInput es la entrada de bills_generate.
type GenerateInput struct {
	ServiceID int64    `json:"service_id" jsonschema:"ID del servicio (obligatorio)"`
	From      string   `json:"from" jsonschema:"Período inicial YYYY-MM (obligatorio)"`
	To        *string  `json:"to,omitempty" jsonschema:"Período final YYYY-MM (default mes actual)"`
	Amount    *float64 `json:"amount,omitempty" jsonschema:"Monto de cada factura; si se omite se usa la última factura o el monto sugerido del servicio"`
	DryRun    *bool    `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
}

func (t *Toolset) generate(ctx context.Context, _ *mcp.CallToolRequest, input GenerateInput) (*mcp.CallToolResult, any, error) {
	if input.ServiceID <= 0 {
		return toolkit.ToolError("invalid_request", "service_id debe ser mayor que cero", "")
	}
	from, err := parsePeriod(input.From)
	if err != nil {
		return toolkit.ToolError("invalid_range", err.Error(), "")
	}
	to := t.currentPeriod()
	if input.To != nil && strings.TrimSpace(*input.To) != "" {
		parsed, err := parsePeriod(*input.To)
		if err != nil {
			return toolkit.ToolError("invalid_range", err.Error(), "")
		}
		to = parsed
	}
	if to.before(from) {
		return toolkit.ToolError("invalid_range", "el período final no puede ser anterior al inicial", "")
	}
	if to.after(t.currentPeriod()) {
		return toolkit.ToolError("invalid_range", "no se pueden generar facturas de períodos futuros", "")
	}
	if monthsBetween(from, to) > 36 {
		return toolkit.ToolError("invalid_range", "el rango máximo es de 36 meses", "")
	}

	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	result, err := t.client.GenerateBills(ctx, input.ServiceID, api.GenerateRequest{
		From:   from.String(),
		To:     to.String(),
		Amount: input.Amount,
		DryRun: dryRun,
	})
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(result)
}
