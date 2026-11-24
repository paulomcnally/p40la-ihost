// Package debts implementa las tools MCP de solo lectura de cuotas de deuda
// (SPEC-154, Ola 1).
package debts

import (
	"context"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de las tools de deudas.
type Toolset struct {
	client api.ReadAPI
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.ReadAPI, cfg config.Config) *Toolset {
	return &Toolset{client: client, cfg: cfg}
}

// Register monta las tools de deudas en el servidor MCP.
func Register(server *mcp.Server, client api.ReadAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "debt_bills_list",
		Description: "Lista cuotas de deuda por deuda (debt_id) o por mes (period YYYY-MM). " +
			"Exactamente uno de los dos modos es obligatorio. No reconcilia deudas ni crea cuotas. Solo lectura.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.billsList)
}

type billsListInput struct {
	DebtID *int64  `json:"debt_id,omitempty" jsonschema:"ID de la deuda (modo por deuda)"`
	Period *string `json:"period,omitempty" jsonschema:"Período YYYY-MM (modo por mes)"`
}

type billsListOutput struct {
	Mode   string           `json:"mode"`
	DebtID *int64           `json:"debt_id,omitempty"`
	Period string           `json:"period,omitempty"`
	Count  int              `json:"count"`
	Bills  []map[string]any `json:"bills"`
}

func (t *Toolset) billsList(ctx context.Context, _ *mcp.CallToolRequest, input billsListInput) (*mcp.CallToolResult, any, error) {
	hasDebt := input.DebtID != nil
	hasPeriod := input.Period != nil && *input.Period != ""
	if hasDebt == hasPeriod {
		return toolkit.ToolError("invalid_request",
			"indicá debt_id o period (YYYY-MM), pero no ambos",
			"Ejemplo: debt_id=1 o period=2026-11.")
	}
	if hasDebt && *input.DebtID <= 0 {
		return toolkit.ToolError("invalid_request", "debt_id debe ser mayor que cero", "")
	}

	var items []map[string]any
	output := billsListOutput{}
	if hasDebt {
		output.Mode = "debt"
		output.DebtID = input.DebtID
		if err := t.client.ReadJSON(ctx, fmt.Sprintf("/api/debts/%d/bills", *input.DebtID), &items); err != nil {
			return toolkit.APIError(err)
		}
	} else {
		year, month, err := toolkit.Period(*input.Period)
		if err != nil {
			return toolkit.ToolError("invalid_period", err.Error(), "Usá el formato YYYY-MM, por ejemplo 2026-11.")
		}
		output.Mode = "month"
		output.Period = fmt.Sprintf("%04d-%02d", year, month)
		query := url.Values{}
		query.Set("year", fmt.Sprintf("%d", year))
		query.Set("month", fmt.Sprintf("%d", month))
		if err := t.client.ReadJSON(ctx, toolkit.WithQuery("/api/debt-bills", query), &items); err != nil {
			return toolkit.APIError(err)
		}
	}
	output.Count = len(items)
	output.Bills = toolkit.OmitAll(items, "invoice_file", "receipt_file")
	return toolkit.JSONResult(output)
}
