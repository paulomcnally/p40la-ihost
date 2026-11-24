// Package billviews implementa las tools MCP de solo lectura de resúmenes,
// proyecciones y vista mensual de facturas de servicios (SPEC-154, Ola 1).
package billviews

import (
	"context"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de las tools de vistas de facturas.
type Toolset struct {
	client api.ReadAPI
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.ReadAPI, cfg config.Config) *Toolset {
	return &Toolset{client: client, cfg: cfg}
}

// Register monta las tools de vistas de facturas en el servidor MCP.
func Register(server *mcp.Server, client api.ReadAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	openWorld := false
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld}

	mcp.AddTool(server, &mcp.Tool{
		Name: "bills_by_month",
		Description: "Lista las facturas de todos los servicios de un mes (vista global multi-servicio). " +
			"Si no se indica período usa el mes actual. Solo lectura; no genera facturas.",
		Annotations: readOnly,
	}, t.byMonth)

	mcp.AddTool(server, &mcp.Tool{
		Name: "bills_summary",
		Description: "Resume las facturas de servicios por período entre dos meses YYYY-MM: confirmadas, " +
			"pendientes y pagadas. Solo lectura.",
		Annotations: readOnly,
	}, t.summary)

	mcp.AddTool(server, &mcp.Tool{
		Name: "bills_projection",
		Description: "Proyecta los próximos meses separando facturas pendientes confirmadas de estimaciones " +
			"de servicios recurrentes. months va de 1 a 24. Solo lectura.",
		Annotations: readOnly,
	}, t.projection)
}

type byMonthInput struct {
	Period *string `json:"period,omitempty" jsonschema:"Período YYYY-MM (default: mes actual)"`
	HomeID *int64  `json:"home_id,omitempty" jsonschema:"Filtra por vivienda"`
}

func (t *Toolset) byMonth(ctx context.Context, _ *mcp.CallToolRequest, input byMonthInput) (*mcp.CallToolResult, any, error) {
	year, month, err := t.period(input.Period)
	if err != nil {
		return toolkit.ToolError("invalid_period", err.Error(), "Usá el formato YYYY-MM, por ejemplo 2026-11.")
	}
	if input.HomeID != nil && *input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	query := url.Values{}
	query.Set("year", fmt.Sprintf("%d", year))
	query.Set("month", fmt.Sprintf("%d", month))
	if input.HomeID != nil {
		query.Set("home_id", fmt.Sprintf("%d", *input.HomeID))
	}
	var view any
	if err := t.client.ReadJSON(ctx, toolkit.WithQuery("/api/service-bills", query), &view); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(view)
}

type summaryInput struct {
	From   string `json:"from" jsonschema:"Período inicial YYYY-MM (obligatorio)"`
	To     string `json:"to" jsonschema:"Período final YYYY-MM (obligatorio)"`
	HomeID *int64 `json:"home_id,omitempty" jsonschema:"Filtra por vivienda"`
}

func (t *Toolset) summary(ctx context.Context, _ *mcp.CallToolRequest, input summaryInput) (*mcp.CallToolResult, any, error) {
	fromYear, fromMonth, err := toolkit.Period(input.From)
	if err != nil {
		return toolkit.ToolError("invalid_period", "from: "+err.Error(), "Usá el formato YYYY-MM.")
	}
	toYear, toMonth, err := toolkit.Period(input.To)
	if err != nil {
		return toolkit.ToolError("invalid_period", "to: "+err.Error(), "Usá el formato YYYY-MM.")
	}
	if fromYear > toYear || (fromYear == toYear && fromMonth > toMonth) {
		return toolkit.ToolError("invalid_range", "from no puede ser posterior a to", "")
	}
	if input.HomeID != nil && *input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	query := url.Values{}
	query.Set("from", input.From)
	query.Set("to", input.To)
	if input.HomeID != nil {
		query.Set("home_id", fmt.Sprintf("%d", *input.HomeID))
	}
	var summary any
	if err := t.client.ReadJSON(ctx, toolkit.WithQuery("/api/service-bills/summary", query), &summary); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(summary)
}

type projectionInput struct {
	From   string `json:"from" jsonschema:"Mes inicial YYYY-MM (obligatorio)"`
	Months *int   `json:"months,omitempty" jsonschema:"Cantidad de meses (1-24, default 6)"`
	HomeID *int64 `json:"home_id,omitempty" jsonschema:"Filtra por vivienda"`
}

func (t *Toolset) projection(ctx context.Context, _ *mcp.CallToolRequest, input projectionInput) (*mcp.CallToolResult, any, error) {
	if _, _, err := toolkit.Period(input.From); err != nil {
		return toolkit.ToolError("invalid_period", "from: "+err.Error(), "Usá el formato YYYY-MM.")
	}
	months := 6
	if input.Months != nil {
		months = *input.Months
	}
	if months < 1 || months > 24 {
		return toolkit.ToolError("invalid_request", "months debe estar entre 1 y 24", "")
	}
	if input.HomeID != nil && *input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	query := url.Values{}
	query.Set("from", input.From)
	query.Set("months", fmt.Sprintf("%d", months))
	if input.HomeID != nil {
		query.Set("home_id", fmt.Sprintf("%d", *input.HomeID))
	}
	var projection any
	if err := t.client.ReadJSON(ctx, toolkit.WithQuery("/api/service-bills/projection", query), &projection); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(projection)
}

func (t *Toolset) period(raw *string) (int, int, error) {
	if raw == nil || *raw == "" {
		year, month := toolkit.CurrentPeriod(t.cfg.Location)
		return year, month, nil
	}
	return toolkit.Period(*raw)
}
