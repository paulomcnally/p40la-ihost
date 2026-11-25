package budget

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

func (t *Toolset) registerLists(server *mcp.Server) {
	openWorld := false
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld}

	mcp.AddTool(server, &mcp.Tool{
		Name: "budget_categories_list",
		Description: "Lista los grupos y las categorías activas del presupuesto (estilo YNAB) con su id, " +
			"nombre, ícono, objetivo y cuenta vinculada si existe. Sirve para mapear un gasto a su categoría " +
			"antes de registrarlo. No son cuentas bancarias. Solo lectura.",
		Annotations: readOnly,
	}, t.categoriesList)

	mcp.AddTool(server, &mcp.Tool{
		Name: "budget_accounts_list",
		Description: "Lista las cuentas activas del presupuesto con su moneda y saldo calculado. Toda " +
			"transacción del presupuesto exige una cuenta de esta lista; no son cuentas bancarias " +
			"(para esas usá bank_accounts_search). Solo lectura.",
		Annotations: readOnly,
	}, t.accountsList)

	mcp.AddTool(server, &mcp.Tool{
		Name: "budget_transactions_list",
		Description: "Lista las transacciones de un mes del presupuesto (year y month obligatorios). " +
			"Útil para verificar lo ya registrado antes de crear un gasto. Solo lectura.",
		Annotations: readOnly,
	}, t.transactionsList)

	mcp.AddTool(server, &mcp.Tool{
		Name: "budget_month_view",
		Description: "Devuelve la vista mensual del presupuesto por grupo y categoría con lo asignado, " +
			"la actividad y lo disponible por moneda, más los totales por moneda. Solo lectura.",
		Annotations: readOnly,
	}, t.monthView)
}

func (t *Toolset) categoriesList(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	groups, err := t.client.ListCategoryGroups(ctx)
	if err != nil {
		return toolkit.APIError(err)
	}
	if groups == nil {
		groups = []api.CategoryGroup{}
	}
	return toolkit.JSONResult(map[string]any{
		"groups": groups,
		"count":  len(groups),
	})
}

func (t *Toolset) accountsList(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	accounts, err := t.client.ListAccounts(ctx)
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"accounts": accounts,
		"count":    len(accounts),
	})
}

type transactionsListInput struct {
	Year  int `json:"year" jsonschema:"Año del presupuesto (obligatorio)"`
	Month int `json:"month" jsonschema:"Mes del presupuesto, 1 a 12 (obligatorio)"`
}

func (t *Toolset) transactionsList(ctx context.Context, _ *mcp.CallToolRequest, input transactionsListInput) (*mcp.CallToolResult, any, error) {
	if result, output, invalid := validatePeriod(input.Year, input.Month); invalid {
		return result, output, nil
	}
	transactions, err := t.client.ListTransactions(ctx, input.Year, input.Month)
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(map[string]any{
		"year":         input.Year,
		"month":        input.Month,
		"count":        len(transactions),
		"transactions": transactions,
	})
}

type monthViewInput struct {
	Year  int `json:"year" jsonschema:"Año del presupuesto (obligatorio)"`
	Month int `json:"month" jsonschema:"Mes del presupuesto, 1 a 12 (obligatorio)"`
}

func (t *Toolset) monthView(ctx context.Context, _ *mcp.CallToolRequest, input monthViewInput) (*mcp.CallToolResult, any, error) {
	if result, output, invalid := validatePeriod(input.Year, input.Month); invalid {
		return result, output, nil
	}
	view, err := t.client.MonthView(ctx, input.Year, input.Month)
	if err != nil {
		return toolkit.APIError(err)
	}
	return toolkit.JSONResult(view)
}

func validatePeriod(year, month int) (*mcp.CallToolResult, any, bool) {
	if year < 1900 || year > 3000 {
		result, output, _ := toolkit.ToolError("invalid_request", "year debe estar entre 1900 y 3000", "")
		return result, output, true
	}
	if month < 1 || month > 12 {
		result, output, _ := toolkit.ToolError("invalid_request", "month debe estar entre 1 y 12", "")
		return result, output, true
	}
	return nil, nil, false
}
