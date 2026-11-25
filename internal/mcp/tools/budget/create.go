package budget

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkit"
)

// createInput es la entrada de budget_transactions_create.
type createInput struct {
	Date           string  `json:"date" jsonschema:"Fecha del gasto YYYY-MM-DD (obligatorio); determina el mes y no puede ser futura"`
	Amount         float64 `json:"amount" jsonschema:"Monto del gasto en la moneda de la cuenta (mayor que cero)"`
	CategoryID     *int64  `json:"category_id,omitempty" jsonschema:"ID de la categoría del presupuesto (alternativa a category)"`
	Category       *string `json:"category,omitempty" jsonschema:"Nombre exacto de la categoría activa (alternativa a category_id)"`
	AccountID      *int64  `json:"account_id,omitempty" jsonschema:"ID de la cuenta del presupuesto; si falta, se usa la cuenta vinculada a la categoría o la primera cuenta activa"`
	Payee          *string `json:"payee,omitempty" jsonschema:"Beneficiario o comercio; se compara para detectar duplicados"`
	Memo           *string `json:"memo,omitempty" jsonschema:"Nota libre"`
	DryRun         *bool   `json:"dry_run,omitempty" jsonschema:"Si es true solo previsualiza; por defecto true"`
	AllowDuplicate *bool   `json:"allow_duplicate,omitempty" jsonschema:"Si es true permite un gasto equivalente el mismo día; por defecto false"`
}

func (t *Toolset) registerCreate(server *mcp.Server) {
	destructive := false
	notIdempotent := false
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name: "budget_transactions_create",
		Description: "Registra un gasto (outflow) en el presupuesto del mes al que pertenece la fecha. " +
			"Resuelve la categoría por category_id o por nombre exacto y la cuenta por account_id, " +
			"por la cuenta vinculada a la categoría o por la primera cuenta activa; la moneda sale de la cuenta. " +
			"Antes de escribir busca un gasto equivalente (misma fecha, categoría, monto y payee) y devuelve " +
			"status=duplicate sin crear otra fila; usá allow_duplicate=true solo para un registro distinto. " +
			"dry_run viene activado por defecto; para escribir se requiere P40LA_MCP_ALLOW_WRITES=true.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive, IdempotentHint: notIdempotent, OpenWorldHint: &openWorld},
	}, t.transactionsCreate)
}

func (t *Toolset) transactionsCreate(ctx context.Context, _ *mcp.CallToolRequest, input createInput) (*mcp.CallToolResult, any, error) {
	dryRun, disabled, output := t.writeMode(input.DryRun)
	if disabled != nil {
		return disabled, output, nil
	}

	date, err := t.validateDate(input.Date)
	if err != nil {
		return toolkit.ToolError("invalid_date", err.Error(), "")
	}
	amount, err := validateAmount(input.Amount)
	if err != nil {
		return toolkit.ToolError("invalid_amount", err.Error(), "")
	}

	payee := stringValue(input.Payee)
	memo := stringValue(input.Memo)

	categoryID := input.CategoryID
	if categoryID != nil && *categoryID <= 0 {
		categoryID = nil
	}
	categoryName := strings.TrimSpace(stringValue(input.Category))
	if categoryID == nil && categoryName == "" {
		return toolkit.ToolError("invalid_request",
			"indicá category_id o category para asociar el gasto",
			"Consultá budget_categories_list para obtener el id o el nombre exacto de la categoría.")
	}

	groups, err := t.client.ListCategoryGroups(ctx)
	if err != nil {
		return toolkit.APIError(err)
	}

	category, code, candidates := resolveCategory(groups, categoryID, categoryName)
	switch code {
	case "":
	case "ambiguous_category":
		return ambiguousCategoryResult(candidates, groups, dryRun)
	case "invalid_request":
		return toolkit.ToolError("invalid_request",
			"category y category_id apuntan a categorías distintas",
			"Pasá solo uno de los dos, o repetí con el nombre exacto de la categoría elegida.")
	default:
		return toolkit.ToolError("category_not_found",
			"no se encontró una categoría activa con el id o el nombre indicado",
			"Consultá budget_categories_list para ver las categorías activas.")
	}

	accounts, err := t.client.ListAccounts(ctx)
	if err != nil {
		return toolkit.APIError(err)
	}
	account, ok := resolveAccount(accounts, input.AccountID, category)
	if !ok {
		return toolkit.ToolError("account_not_found",
			"no hay una cuenta activa del presupuesto para registrar el gasto",
			"Pasá account_id de budget_accounts_list o creá una cuenta en Presupuesto.")
	}
	if account.CurrencyID == 0 {
		return toolkit.ToolError("invalid_request",
			"la cuenta resuelta no tiene moneda configurada",
			"Revisá la cuenta en Presupuesto.")
	}

	year, month := periodOf(date)
	monthTransactions, err := t.client.ListTransactions(ctx, year, month)
	if err != nil {
		return toolkit.APIError(err)
	}
	duplicate := findDuplicate(monthTransactions, date, category.ID, amount, payee)
	allowDuplicate := input.AllowDuplicate != nil && *input.AllowDuplicate

	if duplicate != nil && !allowDuplicate {
		return toolkit.JSONResult(map[string]any{
			"status":   "duplicate",
			"dry_run":  dryRun,
			"existing": duplicateMap(duplicate),
			"hint":     "Ya existe un gasto equivalente el mismo día en esa categoría. Usá allow_duplicate=true solo si es un registro distinto.",
		})
	}

	plan := map[string]any{
		"account_id":    account.ID,
		"account_name":  account.Name,
		"category_id":   category.ID,
		"category_name": category.Name,
		"currency_id":   account.CurrencyID,
		"currency_code": account.CurrencyCode,
		"date":          date,
		"amount":        amount,
		"payee":         payee,
		"memo":          memo,
	}

	if dryRun {
		result := map[string]any{
			"status":  "planned",
			"dry_run": true,
			"plan":    plan,
		}
		if duplicate != nil {
			result["duplicate"] = duplicateMap(duplicate)
			result["warnings"] = []string{"possible_duplicate"}
		}
		return toolkit.JSONResult(result)
	}

	created, err := t.client.CreateTransaction(ctx, api.TransactionRequest{
		AccountID:  account.ID,
		CategoryID: &category.ID,
		CurrencyID: account.CurrencyID,
		Date:       date,
		Payee:      payee,
		Memo:       memo,
		Outflow:    amount,
		Inflow:     0,
		Cleared:    false,
	})
	if err != nil {
		return toolkit.APIError(err)
	}

	result := map[string]any{
		"status":      "created",
		"dry_run":     false,
		"transaction": created,
	}
	if duplicate != nil {
		result["warnings"] = []string{"possible_duplicate"}
	}
	return toolkit.JSONResult(result)
}

// resolveCategory resuelve la categoría por id, por nombre exacto o por ambos
// (verificando coherencia). Devuelve un código vacío cuando resuelve.
func resolveCategory(groups []api.CategoryGroup, categoryID *int64, name string) (*api.Category, string, []api.Category) {
	categories := categoryIndex(groups)
	if categoryID != nil {
		for i := range categories {
			if categories[i].ID != *categoryID {
				continue
			}
			if name == "" {
				return &categories[i], "", nil
			}
			matches := findByName(categories, name)
			switch {
			case len(matches) == 0:
				return nil, "category_not_found", nil
			case len(matches) > 1:
				return nil, "ambiguous_category", matches
			case matches[0].ID != *categoryID:
				return nil, "invalid_request", nil
			}
			return &categories[i], "", nil
		}
		return nil, "category_not_found", nil
	}
	matches := findByName(categories, name)
	switch len(matches) {
	case 0:
		return nil, "category_not_found", nil
	case 1:
		return &matches[0], "", nil
	default:
		return nil, "ambiguous_category", matches
	}
}

func findByName(categories []api.Category, name string) []api.Category {
	target := normalizeName(name)
	matches := make([]api.Category, 0, 1)
	for _, category := range categories {
		if normalizeName(category.Name) == target {
			matches = append(matches, category)
		}
	}
	return matches
}

// resolveAccount aplica el orden account_id → cuenta de la categoría → primera
// cuenta activa (el backend devuelve las cuentas ordenadas por nombre).
func resolveAccount(accounts []api.Account, accountID *int64, category *api.Category) (*api.Account, bool) {
	if accountID != nil && *accountID > 0 {
		for i := range accounts {
			if accounts[i].ID == *accountID {
				return &accounts[i], true
			}
		}
		return nil, false
	}
	if category != nil && category.AccountID != nil {
		for i := range accounts {
			if accounts[i].ID == *category.AccountID {
				return &accounts[i], true
			}
		}
	}
	if len(accounts) == 0 {
		return nil, false
	}
	return &accounts[0], true
}

// findDuplicate busca un gasto equivalente en el mes: misma fecha, categoría,
// monto y payee normalizado (vacío en ambos lados equivale a coincidencia).
func findDuplicate(transactions []api.Transaction, date string, categoryID int64, amount float64, payee string) *api.Transaction {
	targetPayee := normalizeName(payee)
	for i := range transactions {
		transaction := &transactions[i]
		if dateOnly(transaction.Date) != date {
			continue
		}
		if transaction.CategoryID == nil || *transaction.CategoryID != categoryID {
			continue
		}
		if transaction.Outflow <= 0 || round2(transaction.Outflow) != amount {
			continue
		}
		if normalizeName(transaction.Payee) != targetPayee {
			continue
		}
		return transaction
	}
	return nil
}

func duplicateMap(transaction *api.Transaction) map[string]any {
	if transaction == nil {
		return nil
	}
	result := map[string]any{
		"id":            transaction.ID,
		"date":          dateOnly(transaction.Date),
		"category_id":   transaction.CategoryID,
		"category_name": transaction.CategoryName,
		"outflow":       transaction.Outflow,
		"payee":         transaction.Payee,
	}
	if transaction.AccountID != 0 {
		result["account_id"] = transaction.AccountID
	}
	if transaction.AccountName != "" {
		result["account_name"] = transaction.AccountName
	}
	return result
}

func ambiguousCategoryResult(candidates []api.Category, groups []api.CategoryGroup, dryRun bool) (*mcp.CallToolResult, any, error) {
	list := make([]map[string]any, 0, len(candidates))
	for _, candidate := range candidates {
		item := map[string]any{
			"id":                candidate.ID,
			"name":              candidate.Name,
			"category_group_id": candidate.CategoryGroupID,
		}
		if name := groupName(groups, candidate.ID); name != "" {
			item["group_name"] = name
		}
		list = append(list, item)
	}
	payload := map[string]any{
		"error":            "ambiguous_category",
		"message":          "Más de una categoría activa coincide con el nombre indicado.",
		"status":           "ambiguous_category",
		"dry_run":          dryRun,
		"candidates":       list,
		"hint":             "Pasá category_id para elegir una de las candidatas; no se escribió nada.",
		"category_id_hint": "budget_categories_list",
	}
	encoded, _ := json.Marshal(payload)
	return &mcp.CallToolResult{
		IsError:           true,
		Content:           []mcp.Content{&mcp.TextContent{Text: string(encoded)}},
		StructuredContent: payload,
	}, payload, nil
}

func (t *Toolset) validateDate(value string) (string, error) {
	parsed, err := time.Parse(dateLayout, strings.TrimSpace(value))
	if err != nil {
		return "", errors.New("la fecha debe tener formato YYYY-MM-DD")
	}
	today, err := time.Parse(dateLayout, t.today())
	if err != nil {
		return "", errors.New("no se pudo resolver la fecha actual")
	}
	if parsed.After(today) {
		return "", errors.New("la fecha no puede ser futura")
	}
	return parsed.Format(dateLayout), nil
}

func validateAmount(value float64) (float64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, errors.New("amount no es válido")
	}
	if value <= 0 {
		return 0, errors.New("amount debe ser mayor que cero")
	}
	return round2(value), nil
}

func periodOf(date string) (int, int) {
	year, _ := strconv.Atoi(date[0:4])
	month, _ := strconv.Atoi(date[5:7])
	return year, month
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
