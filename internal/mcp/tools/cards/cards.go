// Package cards implementa las tools MCP de solo lectura de estados de cuenta
// de tarjetas de crédito (SPEC-154, Ola 1).
package cards

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// Toolset agrupa las dependencias de las tools de tarjetas.
type Toolset struct {
	client api.ReadAPI
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.ReadAPI, cfg config.Config) *Toolset {
	return &Toolset{client: client, cfg: cfg}
}

// Register monta las tools de tarjetas en el servidor MCP.
func Register(server *mcp.Server, client api.ReadAPI, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	openWorld := false
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld}

	mcp.AddTool(server, &mcp.Tool{
		Name: "credit_card_statements_list",
		Description: "Lista los estados de cuenta de una tarjeta de crédito, del más reciente al más antiguo, " +
			"con cantidad de movimientos por estado. Solo lectura.",
		Annotations: readOnly,
	}, t.statementsList)

	mcp.AddTool(server, &mcp.Tool{
		Name: "credit_card_statements_get",
		Description: "Devuelve un estado de cuenta con su resumen por tipo de movimiento y su conciliación " +
			"(base cash y base balance). Solo lectura.",
		Annotations: readOnly,
	}, t.statementGet)
}

type statementsListInput struct {
	CardID int64 `json:"card_id" jsonschema:"ID de la tarjeta de crédito (obligatorio)"`
}

func (t *Toolset) statementsList(ctx context.Context, _ *mcp.CallToolRequest, input statementsListInput) (*mcp.CallToolResult, any, error) {
	if input.CardID <= 0 {
		return toolkit.ToolError("invalid_request", "card_id debe ser mayor que cero", "")
	}
	var items []map[string]any
	if err := t.client.ReadJSON(ctx, fmt.Sprintf("/api/credit-cards/%d/statements", input.CardID), &items); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"card_id":    input.CardID,
		"count":      len(items),
		"statements": items,
	})
}

type statementGetInput struct {
	StatementID int64 `json:"statement_id" jsonschema:"ID del estado de cuenta (obligatorio)"`
}

func (t *Toolset) statementGet(ctx context.Context, _ *mcp.CallToolRequest, input statementGetInput) (*mcp.CallToolResult, any, error) {
	if input.StatementID <= 0 {
		return toolkit.ToolError("invalid_request", "statement_id debe ser mayor que cero", "")
	}
	var statement map[string]any
	if err := t.client.ReadJSON(ctx, fmt.Sprintf("/api/credit-card-statements/%d", input.StatementID), &statement); err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(statement)
}
