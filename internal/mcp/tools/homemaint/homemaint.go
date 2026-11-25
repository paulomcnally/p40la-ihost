// Package homemaint implementa las tools MCP de reparaciones y mantenimientos
// del hogar: listado de lectura, alta de la reparación, registro de mano de
// obra e insumos y movimiento de líneas entre reparaciones.
package homemaint

import (
	"context"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

const dateLayout = "2006-01-02"

// Toolset agrupa las dependencias de las tools de reparaciones.
type Toolset struct {
	client api.HomeMaintenanceAPI
	cfg    config.Config
	now    func() time.Time
}

// New crea un Toolset con el reloj real.
func New(client api.HomeMaintenanceAPI, cfg config.Config) *Toolset {
	if cfg.Location == nil {
		cfg.Location = time.Local
	}
	return &Toolset{client: client, cfg: cfg, now: time.Now}
}

// Register monta las tools de reparaciones en el servidor MCP.
func Register(server *mcp.Server, client api.HomeMaintenanceAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	t.registerList(server)
	t.registerCreate(server)
	t.registerLabor(server)
	t.registerSupplies(server)
	t.registerMove(server)
}

func (t *Toolset) today() string {
	return t.now().In(t.cfg.Location).Format(dateLayout)
}

type listInput struct {
	HomeID       int64   `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
	Year         *int    `json:"year,omitempty" jsonschema:"Filtra por año (1900-9999)"`
	Kind         *string `json:"kind,omitempty" jsonschema:"Tipo: corrective o preventive"`
	Date         *string `json:"date,omitempty" jsonschema:"Fecha exacta YYYY-MM-DD; no se combina con year"`
	ItemType     *string `json:"item_type,omitempty" jsonschema:"Filtra reparaciones con líneas labor o supplies"`
	Query        *string `json:"query,omitempty" jsonschema:"Texto en la descripción de la reparación o de sus líneas (máximo 100)"`
	IncludeItems *bool   `json:"include_items,omitempty" jsonschema:"Si es true incluye las líneas de cada reparación"`
}

func (t *Toolset) registerList(server *mcp.Server) {
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "home_maintenance_list",
		Description: "Lista las reparaciones y mantenimientos de una vivienda con totales por moneda " +
			"(mano de obra e insumos). Filtros opcionales de año, tipo de reparación, fecha exacta, tipo de línea " +
			"y texto; include_items carga las líneas. Solo lectura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.list)
}

func (t *Toolset) list(ctx context.Context, _ *mcp.CallToolRequest, input listInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	filter := api.HomeMaintenanceFilter{}
	if input.Year != nil {
		if *input.Year < 1900 || *input.Year > 9999 {
			return toolkit.ToolError("invalid_request", "year debe estar entre 1900 y 9999", "")
		}
		filter.Year = *input.Year
	}
	if input.Kind != nil && strings.TrimSpace(*input.Kind) != "" {
		switch strings.TrimSpace(*input.Kind) {
		case "corrective", "preventive":
			filter.Kind = strings.TrimSpace(*input.Kind)
		default:
			return toolkit.ToolError("invalid_request", "kind debe ser corrective o preventive", "")
		}
	}
	if input.Date != nil && strings.TrimSpace(*input.Date) != "" {
		if filter.Year > 0 {
			return toolkit.ToolError("invalid_request", "date y year no se pueden combinar", "")
		}
		parsed, err := time.Parse(dateLayout, strings.TrimSpace(*input.Date))
		if err != nil {
			return toolkit.ToolError("invalid_request", "date debe tener formato YYYY-MM-DD", "")
		}
		filter.Date = parsed.Format(dateLayout)
	}
	if input.ItemType != nil && strings.TrimSpace(*input.ItemType) != "" {
		itemType := strings.TrimSpace(*input.ItemType)
		if itemType != itemTypeLabor && itemType != itemTypeSupplies {
			return toolkit.ToolError("invalid_request", "item_type debe ser labor o supplies", "")
		}
		filter.ItemType = itemType
	}
	if input.Query != nil {
		query := strings.TrimSpace(*input.Query)
		if len([]rune(query)) > maxQueryLen {
			return toolkit.ToolError("invalid_request", "query no puede superar los 100 caracteres", "")
		}
		filter.Query = query
	}
	if input.IncludeItems != nil {
		filter.IncludeItems = *input.IncludeItems
	}

	result, err := t.client.ListHomeMaintenances(ctx, input.HomeID, filter)
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"home_id":            input.HomeID,
		"count":              len(result.Items),
		"maintenances":       result.Items,
		"totals_by_currency": result.TotalsByCurrency,
		"truncated":          result.Truncated,
	})
}
