// Package rental implementa las tools MCP de solo lectura de alquileres y
// portafolio inmobiliario (SPEC-154, Ola 1).
package rental

import (
	"context"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de las tools de alquileres.
type Toolset struct {
	client api.ReadAPI
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.ReadAPI, cfg config.Config) *Toolset {
	return &Toolset{client: client, cfg: cfg}
}

// Register monta las tools de alquileres en el servidor MCP.
func Register(server *mcp.Server, client api.ReadAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	openWorld := false
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld}

	mcp.AddTool(server, &mcp.Tool{
		Name: "rental_leases_list",
		Description: "Lista contratos de alquiler con filtros opcionales de estado y vivienda. " +
			"Devuelve saldos derivados y omit múltiples campos de contacto. Solo lectura.",
		Annotations: readOnly,
	}, t.leasesList)

	mcp.AddTool(server, &mcp.Tool{
		Name: "rental_lease_get",
		Description: "Devuelve el detalle de un contrato de alquiler y su balance (cargado, pagado y pendiente). " +
			"Omite emails y teléfonos de las partes. Solo lectura.",
		Annotations: readOnly,
	}, t.leaseGet)

	mcp.AddTool(server, &mcp.Tool{
		Name: "rental_home_status",
		Description: "Devuelve el estado de una vivienda: publicación, contrato activo, apartado activo, " +
			"cargos pendientes y recordatorios. Solo lectura.",
		Annotations: readOnly,
	}, t.homeStatus)

	mcp.AddTool(server, &mcp.Tool{
		Name: "rental_portfolio_summary",
		Description: "Resume el portafolio de alquileres de un mes: ingresos, pendientes, gastos y estado por vivienda. " +
			"Si no se indica período usa el mes actual. Solo lectura.",
		Annotations: readOnly,
	}, t.portfolioSummary)
}

type leasesListInput struct {
	Status *string `json:"status,omitempty" jsonschema:"Filtro de estado del contrato (draft, active, terminated)"`
	HomeID *int64  `json:"home_id,omitempty" jsonschema:"Filtra por vivienda"`
}

func (t *Toolset) leasesList(ctx context.Context, _ *mcp.CallToolRequest, input leasesListInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID != nil && *input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	query := url.Values{}
	if input.Status != nil {
		query.Set("status", *input.Status)
	}
	if input.HomeID != nil {
		query.Set("home_id", fmt.Sprintf("%d", *input.HomeID))
	}
	var items []map[string]any
	if err := t.client.ReadJSON(ctx, toolkit.WithQuery("/api/rental/leases", query), &items); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(toolkit.OmitDeep(map[string]any{
		"count":  len(items),
		"leases": items,
	}, sensitiveKeys...))
}

type leaseGetInput struct {
	LeaseID int64 `json:"lease_id" jsonschema:"ID del contrato de alquiler (obligatorio)"`
}

type leaseGetOutput struct {
	Lease   map[string]any `json:"lease"`
	Balance any            `json:"balance"`
}

func (t *Toolset) leaseGet(ctx context.Context, _ *mcp.CallToolRequest, input leaseGetInput) (*mcp.CallToolResult, any, error) {
	if input.LeaseID <= 0 {
		return toolkit.ToolError("invalid_request", "lease_id debe ser mayor que cero", "")
	}
	var lease map[string]any
	if err := t.client.ReadJSON(ctx, fmt.Sprintf("/api/rental/leases/%d", input.LeaseID), &lease); err != nil {
		return toolkit.APIError(err)
	}
	var balance any
	if err := t.client.ReadJSON(ctx, fmt.Sprintf("/api/rental/leases/%d/balance", input.LeaseID), &balance); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(toolkit.OmitDeep(leaseGetOutput{
		Lease:   lease,
		Balance: balance,
	}, sensitiveKeys...))
}

type homeStatusInput struct {
	HomeID int64 `json:"home_id" jsonschema:"ID de la vivienda (obligatorio)"`
}

func (t *Toolset) homeStatus(ctx context.Context, _ *mcp.CallToolRequest, input homeStatusInput) (*mcp.CallToolResult, any, error) {
	if input.HomeID <= 0 {
		return toolkit.ToolError("invalid_request", "home_id debe ser mayor que cero", "")
	}
	var status map[string]any
	if err := t.client.ReadJSON(ctx, fmt.Sprintf("/api/rental/homes/%d", input.HomeID), &status); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(toolkit.OmitDeep(status, sensitiveKeys...))
}

type portfolioInput struct {
	Period *string `json:"period,omitempty" jsonschema:"Período YYYY-MM (default: mes actual)"`
}

func (t *Toolset) portfolioSummary(ctx context.Context, _ *mcp.CallToolRequest, input portfolioInput) (*mcp.CallToolResult, any, error) {
	year, month, err := t.period(input.Period)
	if err != nil {
		return toolkit.ToolError("invalid_period", err.Error(), "Usá el formato YYYY-MM, por ejemplo 2026-11.")
	}
	query := url.Values{}
	query.Set("year", fmt.Sprintf("%d", year))
	query.Set("month", fmt.Sprintf("%d", month))
	var summary map[string]any
	if err := t.client.ReadJSON(ctx, toolkit.WithQuery("/api/rental/portfolio/summary", query), &summary); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(toolkit.OmitDeep(summary, sensitiveKeys...))
}

func (t *Toolset) period(raw *string) (int, int, error) {
	if raw == nil || *raw == "" {
		year, month := toolkit.CurrentPeriod(t.cfg.Location)
		return year, month, nil
	}
	return toolkit.Period(*raw)
}

// sensitiveKeys son datos de contacto que no aportan a la consulta del agente.
var sensitiveKeys = []string{
	"email", "notify_email", "phone", "person_email", "tenant_email",
	"guarantor_email", "applicant_email", "agent_email", "agency_email",
	"extra_email", "extra_emails",
}
