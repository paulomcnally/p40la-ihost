// Package pension implementa las tools MCP de solo lectura de pensión,
// salarios y catálogos familiares (SPEC-154, Ola 1).
package pension

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de las tools de pensión.
type Toolset struct {
	client api.ReadAPI
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.ReadAPI, cfg config.Config) *Toolset {
	if cfg.Location == nil {
		cfg.Location = time.Local
	}
	return &Toolset{client: client, cfg: cfg}
}

// Register monta las tools de pensión en el servidor MCP.
func Register(server *mcp.Server, client api.ReadAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	openWorld := false

	mcp.AddTool(server, &mcp.Tool{
		Name: "pension_month_summary",
		Description: "Resume un mes de pensión: registros de manutención, pagos de salario y estado de cierre. " +
			"Devuelve totales agrupados por moneda y conteos por estado. Solo lectura; no modifica datos.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.monthSummary)

	mcp.AddTool(server, &mcp.Tool{
		Name: "pension_catalogs_list",
		Description: "Lista los catálogos necesarios para operar pensión: hijos, categorías, salarios, " +
			"configuraciones de generación automática y monedas. Solo lectura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.catalogs)
}

type monthSummaryInput struct {
	Period  *string `json:"period,omitempty" jsonschema:"Período YYYY-MM (default: mes actual)"`
	ChildID *int64  `json:"child_id,omitempty" jsonschema:"Filtra los registros por hijo (opcional)"`
}

type monthSummaryOutput struct {
	Year           int                     `json:"year"`
	Month          int                     `json:"month"`
	Period         string                  `json:"period"`
	Closed         bool                    `json:"closed"`
	ClosedAt       any                     `json:"closed_at,omitempty"`
	RecordsCount   int                     `json:"records_count"`
	PaidCount      int                     `json:"paid_count"`
	PendingCount   int                     `json:"pending_count"`
	RejectedCount  int                     `json:"rejected_count"`
	RecordTotals   []toolkit.CurrencyTotal `json:"record_totals"`
	SalaryCount    int                     `json:"salary_count"`
	ReceivedCount  int                     `json:"received_count"`
	SalaryTotals   []toolkit.CurrencyTotal `json:"salary_totals"`
	Records        []map[string]any        `json:"records"`
	SalaryPayments []map[string]any        `json:"salary_payments"`
}

func (t *Toolset) monthSummary(ctx context.Context, _ *mcp.CallToolRequest, input monthSummaryInput) (*mcp.CallToolResult, any, error) {
	year, month, err := t.period(input.Period)
	if err != nil {
		return toolkit.ToolError("invalid_period", err.Error(), "Usá el formato YYYY-MM, por ejemplo 2026-11.")
	}
	if input.ChildID != nil && *input.ChildID <= 0 {
		return toolkit.ToolError("invalid_request", "child_id debe ser mayor que cero", "")
	}

	query := url.Values{}
	query.Set("year", fmt.Sprintf("%d", year))
	query.Set("month", fmt.Sprintf("%d", month))
	if input.ChildID != nil {
		query.Set("child_id", fmt.Sprintf("%d", *input.ChildID))
	}

	var records []map[string]any
	if err := t.client.ReadJSON(ctx, toolkit.WithQuery("/api/pension/records", query), &records); err != nil {
		return toolkit.APIError(err)
	}
	var payments []map[string]any
	if err := t.client.ReadJSON(ctx, toolkit.WithQuery("/api/pension/salary-payments", query), &payments); err != nil {
		return toolkit.APIError(err)
	}
	var closing struct {
		Closed   bool `json:"closed"`
		ClosedAt any  `json:"closed_at"`
	}
	if err := t.client.ReadJSON(ctx, fmt.Sprintf("/api/pension/closing/%d/%d", year, month), &closing); err != nil {
		return toolkit.APIError(err)
	}

	output := monthSummaryOutput{
		Year:           year,
		Month:          month,
		Period:         fmt.Sprintf("%04d-%02d", year, month),
		Closed:         closing.Closed,
		ClosedAt:       closing.ClosedAt,
		Records:        toolkit.ProjectAll(records, recordKeys...),
		SalaryPayments: toolkit.ProjectAll(payments, salaryKeys...),
		RecordTotals:   toolkit.TotalsByCurrency(records, "currency", "amount"),
		SalaryTotals:   toolkit.TotalsByCurrency(payments, "currency", "amount"),
	}
	output.RecordsCount = len(records)
	output.SalaryCount = len(payments)
	for _, record := range records {
		switch record["status"] {
		case "paid":
			output.PaidCount++
		case "pending":
			output.PendingCount++
		case "rejected":
			output.RejectedCount++
		}
	}
	for _, payment := range payments {
		if payment["status"] == "received" {
			output.ReceivedCount++
		}
	}
	return toolkit.JSONResult(output)
}

type catalogsOutput struct {
	Children   []map[string]any `json:"children"`
	Categories []map[string]any `json:"pension_categories"`
	Salaries   []map[string]any `json:"salaries"`
	Configs    []map[string]any `json:"configs"`
	Currencies []map[string]any `json:"currencies"`
}

func (t *Toolset) catalogs(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	children, err := t.readProjected(ctx, "/api/children", childKeys...)
	if err != nil {
		return toolkit.APIError(err)
	}
	categories, err := t.readProjected(ctx, "/api/pension-categories", categoryKeys...)
	if err != nil {
		return toolkit.APIError(err)
	}
	salaries, err := t.readProjected(ctx, "/api/salaries", salaryListKeys...)
	if err != nil {
		return toolkit.APIError(err)
	}
	configs, err := t.readProjected(ctx, "/api/pension/configs", configKeys...)
	if err != nil {
		return toolkit.APIError(err)
	}
	currencies, err := t.readProjected(ctx, "/api/currencies", currencyKeys...)
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(catalogsOutput{
		Children:   children,
		Categories: categories,
		Salaries:   salaries,
		Configs:    configs,
		Currencies: currencies,
	})
}

func (t *Toolset) readProjected(ctx context.Context, path string, keys ...string) ([]map[string]any, error) {
	var items []map[string]any
	if err := t.client.ReadJSON(ctx, path, &items); err != nil {
		return nil, err
	}
	return toolkit.ProjectAll(items, keys...), nil
}

func (t *Toolset) period(raw *string) (int, int, error) {
	if raw == nil || *raw == "" {
		year, month := toolkit.CurrentPeriod(t.cfg.Location)
		return year, month, nil
	}
	return toolkit.Period(*raw)
}

var recordKeys = []string{
	"id", "child_id", "child_name", "pension_category_id", "category_name",
	"year", "month", "amount", "currency", "status", "paid_at", "payment_method",
	"original_amount", "original_currency",
}

var salaryKeys = []string{
	"id", "salary_id", "employer", "year", "month", "amount", "currency",
	"status", "received_amount", "received_at",
}

var childKeys = []string{"id", "first_name", "last_name", "birth_date"}

var categoryKeys = []string{"id", "name", "share_percent", "currency", "auto_generate"}

var salaryListKeys = []string{"id", "employer", "amount", "currency_id", "payment_day", "active"}

var configKeys = []string{
	"id", "child_id", "child_name", "pension_category_id", "category_name",
	"amount", "currency", "is_active", "auto_generate",
}

var currencyKeys = []string{"id", "code", "name", "symbol"}
