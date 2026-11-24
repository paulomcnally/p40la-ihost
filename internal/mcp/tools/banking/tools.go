// Package banking implementa las tools MCP de cuentas bancarias y tarjetas de
// crédito de p40la-ihost (SPEC-152).
package banking

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

// Toolset agrupa las dependencias de las tools de banca.
type Toolset struct {
	client api.API
	cfg    config.Config
}

// New crea un Toolset.
func New(client api.API, cfg config.Config) *Toolset {
	return &Toolset{client: client, cfg: cfg}
}

// Register monta las seis tools de banca en el servidor MCP.
func Register(server *mcp.Server, client api.API, cfg config.Config) {
	New(client, cfg).Register(server)
}

// Register monta las tools del Toolset.
func (t *Toolset) Register(server *mcp.Server) {
	openWorld := false

	mcp.AddTool(server, &mcp.Tool{
		Name: "bank_accounts_search",
		Description: "Busca cuentas bancarias por alias, número de cuenta, beneficiario o banco. " +
			"Solo lectura. query, bank/bank_id, active y limit son opcionales (limit default 10, máximo 50). " +
			"Ordena coincidencias exactas, luego prefijo y luego contenido.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.accountsSearch)

	mcp.AddTool(server, &mcp.Tool{
		Name: "bank_accounts_create",
		Description: "Crea una cuenta bancaria. Requiere bank (nombre, short_code o bank_id), alias, account_number, " +
			"currency (código o id) y beneficiary. account_type (savings default o checking), notes y active son opcionales. " +
			"Si la cuenta ya existe en el banco devuelve already_exists. Requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: boolPtr(true),
			IdempotentHint:  false,
			OpenWorldHint:   &openWorld,
		},
	}, t.accountsCreate)

	mcp.AddTool(server, &mcp.Tool{
		Name: "bank_accounts_update",
		Description: "Edita una cuenta bancaria por id. Solo cambia los campos enviados; el resto se conserva. " +
			"El banco no se cambia. Requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: boolPtr(true),
			IdempotentHint:  true,
			OpenWorldHint:   &openWorld,
		},
	}, t.accountsUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name: "credit_cards_search",
		Description: "Busca tarjetas de crédito por alias, últimos cuatro dígitos, titular o banco. " +
			"Solo lectura. query, bank/bank_id, active y limit son opcionales (limit default 10, máximo 50).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, t.cardsSearch)

	mcp.AddTool(server, &mcp.Tool{
		Name: "credit_cards_create",
		Description: "Crea una tarjeta de crédito. Requiere bank (nombre, short_code o bank_id), alias, network " +
			"(visa, mastercard, amex, diners u other), last4 (4 dígitos) y currency (código o id). " +
			"Defaults: statement_day=1, payment_due_day=1, min_payment_pct=4, limit_currency=NIO, color=blue, icon=credit, active=true. " +
			"Si el alias ya existe en el banco devuelve already_exists. Requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: boolPtr(true),
			IdempotentHint:  false,
			OpenWorldHint:   &openWorld,
		},
	}, t.cardsCreate)

	mcp.AddTool(server, &mcp.Tool{
		Name: "credit_cards_update",
		Description: "Edita una tarjeta de crédito por id. Solo cambia los campos enviados; el resto se conserva. " +
			"El banco no se cambia y secondary_currency=\"\" limpia la moneda secundaria. Requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: boolPtr(true),
			IdempotentHint:  true,
			OpenWorldHint:   &openWorld,
		},
	}, t.cardsUpdate)
}

// writeEnabled aplica la compuerta de escritura del MCP.
func (t *Toolset) writeEnabled() (bool, *mcp.CallToolResult, any) {
	if t.cfg.WritesEnabled() {
		return true, nil, nil
	}
	hint := "Definí P40LA_MCP_ALLOW_WRITES=true para ejecutar cambios."
	if t.cfg.IsHTTP() {
		hint = "Definí P40LA_MCP_ALLOW_WRITES=true y P40LA_MCP_REMOTE_WRITES=true para ejecutar cambios."
	}
	result, output, _ := toolError("writes_disabled",
		"Las escrituras están deshabilitadas en este servidor MCP.", hint)
	return false, result, output
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}
