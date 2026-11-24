package homemaint

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// laborInput es la entrada de home_maintenance_labor_register.
type laborInput struct {
	HomeID         int64   `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	MaintenanceID  *int64  `json:"maintenance_id,omitempty" jsonschema:"ID de una reparación existente (opcional)"`
	PerformedAt    *string `json:"performed_at,omitempty" jsonschema:"Fecha YYYY-MM-DD, no futura; obligatoria si no hay maintenance_id"`
	Area           *string `json:"area,omitempty" jsonschema:"Área: plumbing, electrical, roof, paint, carpentry, appliances, garden, cleaning u other"`
	Description    string  `json:"description" jsonschema:"Descripción de la mano de obra (obligatoria; máximo 200)"`
	Amount         float64 `json:"amount" jsonschema:"Monto de la línea (mayor o igual a 0)"`
	CurrencyID     int64   `json:"currency_id" jsonschema:"ID de la moneda (obligatorio)"`
	InstitutionID  *int64  `json:"institution_id,omitempty" jsonschema:"ID del proveedor o contratista (opcional)"`
	DryRun         *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
	AllowDuplicate *bool   `json:"allow_duplicate,omitempty" jsonschema:"Si es true permite un duplicado el mismo día; por defecto false"`
}

func (t *Toolset) registerLabor(server *mcp.Server) {
	destructive := false
	notIdempotent := false
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_maintenance_labor_register",
		Description: "Registra una línea de mano de obra en una reparación del hogar. " +
			"Busca las líneas del mismo día y devuelve status=duplicate si ya existe una equivalente; " +
			"usá allow_duplicate=true solo para un registro distinto. Si no hay maintenance_id, reutiliza " +
			"la reparación del día o la crea con la línea en una transacción. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: notIdempotent, OpenWorldHint: &openWorld},
	}, t.laborRegister)
}

func (t *Toolset) laborRegister(ctx context.Context, _ *mcp.CallToolRequest, input laborInput) (*mcp.CallToolResult, any, error) {
	return t.registerLine(ctx, writeRequest{
		HomeID:         input.HomeID,
		MaintenanceID:  input.MaintenanceID,
		PerformedAt:    stringValue(input.PerformedAt),
		Area:           stringValue(input.Area),
		Description:    input.Description,
		Amount:         input.Amount,
		CurrencyID:     input.CurrencyID,
		InstitutionID:  input.InstitutionID,
		DryRun:         input.DryRun,
		AllowDuplicate: input.AllowDuplicate != nil && *input.AllowDuplicate,
		ItemType:       itemTypeLabor,
		Quantity:       1,
		Unit:           unitDefault,
	})
}
