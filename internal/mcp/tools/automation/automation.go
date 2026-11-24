// Package automation implementa las tools MCP de solo lectura de
// automatizaciones y su historial (SPEC-154, Ola 1).
package automation

import (
	"context"
	"fmt"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de las tools de automatizaciones.
type Toolset struct {
	client api.ReadAPI
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.ReadAPI, cfg config.Config) *Toolset {
	return &Toolset{client: client, cfg: cfg}
}

// Register monta las tools de automatizaciones en el servidor MCP.
func Register(server *mcp.Server, client api.ReadAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	openWorld := false
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld}

	mcp.AddTool(server, &mcp.Tool{
		Name: "automation_accounts_list",
		Description: "Lista las cuentas de automatización con su institución, plugin, programación y última ejecución. " +
			"Omite el identificador de cuenta y la URL de webhook. Solo lectura.",
		Annotations: readOnly,
	}, t.accountsList)

	mcp.AddTool(server, &mcp.Tool{
		Name: "automation_runs_list",
		Description: "Lista las facturas consultadas y los logs de entrega de una cuenta de automatización. " +
			"Omite payloads crudos y URLs de webhook. Solo lectura.",
		Annotations: readOnly,
	}, t.runsList)
}

func (t *Toolset) accountsList(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	var items []map[string]any
	if err := t.client.ReadJSON(ctx, "/api/automation/accounts", &items); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"count":    len(items),
		"accounts": toolkit.OmitAll(items, sensitiveKeys...),
	})
}

type runsListInput struct {
	AccountID int64 `json:"account_id" jsonschema:"ID de la cuenta de automatización (obligatorio)"`
	Limit     *int  `json:"limit,omitempty" jsonschema:"Máximo de logs de webhook (1-200, default 50)"`
}

func (t *Toolset) runsList(ctx context.Context, _ *mcp.CallToolRequest, input runsListInput) (*mcp.CallToolResult, any, error) {
	if input.AccountID <= 0 {
		return toolkit.ToolError("invalid_request", "account_id debe ser mayor que cero", "")
	}
	limit := 50
	if input.Limit != nil {
		limit = *input.Limit
	}
	if limit < 1 || limit > 200 {
		return toolkit.ToolError("invalid_request", "limit debe estar entre 1 y 200", "")
	}

	var bills []map[string]any
	if err := t.client.ReadJSON(ctx, fmt.Sprintf("/api/automation/accounts/%d/bills", input.AccountID), &bills); err != nil {
		return toolkit.APIError(err)
	}
	query := url.Values{}
	query.Set("limit", fmt.Sprintf("%d", limit))
	var logs []map[string]any
	if err := t.client.ReadJSON(ctx, toolkit.WithQuery(fmt.Sprintf("/api/automation/accounts/%d/webhook/logs", input.AccountID), query), &logs); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"account_id":   input.AccountID,
		"bills_count":  len(bills),
		"bills":        toolkit.OmitAll(bills, sensitiveKeys...),
		"logs_count":   len(logs),
		"webhook_logs": toolkit.OmitAll(logs, sensitiveKeys...),
	})
}

var sensitiveKeys = []string{
	"identifier", "webhook_url", "payload", "raw", "request_body",
	"response_body", "credentials", "webhook_uuid",
}
