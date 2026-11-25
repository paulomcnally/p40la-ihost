package budget

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/tools/toolkitest"
)

type fakeAPI struct {
	groups    []api.CategoryGroup
	groupsErr error

	accounts    []api.Account
	accountsErr error

	transactions    []api.Transaction
	transactionsErr error

	view    *api.MonthView
	viewErr error

	createResult *api.Transaction
	createErr    error

	createCalls         int
	lastCreate          api.TransactionRequest
	listCalls           int
	lastYear, lastMonth int
	viewCalls           int
	lastViewYear        int
	lastViewMonth       int
}

func (f *fakeAPI) ListCategoryGroups(context.Context) ([]api.CategoryGroup, error) {
	return f.groups, f.groupsErr
}

func (f *fakeAPI) ListAccounts(context.Context) ([]api.Account, error) {
	return f.accounts, f.accountsErr
}

func (f *fakeAPI) ListTransactions(_ context.Context, year, month int) ([]api.Transaction, error) {
	f.listCalls++
	f.lastYear = year
	f.lastMonth = month
	return f.transactions, f.transactionsErr
}

func (f *fakeAPI) MonthView(_ context.Context, year, month int) (*api.MonthView, error) {
	f.viewCalls++
	f.lastViewYear = year
	f.lastViewMonth = month
	if f.view == nil {
		return &api.MonthView{Year: year, Month: month}, f.viewErr
	}
	return f.view, f.viewErr
}

func (f *fakeAPI) CreateTransaction(_ context.Context, req api.TransactionRequest) (*api.Transaction, error) {
	f.createCalls++
	f.lastCreate = req
	if f.createResult != nil {
		return f.createResult, f.createErr
	}
	return &api.Transaction{
		ID:           10,
		AccountID:    req.AccountID,
		CategoryID:   req.CategoryID,
		CurrencyID:   req.CurrencyID,
		Date:         req.Date,
		Payee:        req.Payee,
		Memo:         req.Memo,
		Outflow:      req.Outflow,
		Inflow:       req.Inflow,
		Cleared:      req.Cleared,
		AccountName:  "Cuenta Demo",
		CategoryName: "Comida",
		CurrencyCode: "NIO",
	}, f.createErr
}

func demoGroups() []api.CategoryGroup {
	checking := int64(3)
	savings := int64(4)
	return []api.CategoryGroup{
		{
			ID: 2, Name: "Necesidades", Icon: "cart", SortOrder: 1,
			Categories: []api.Category{
				{ID: 5, CategoryGroupID: 2, Name: "Comida", Icon: "food", AccountID: &checking},
				{ID: 6, CategoryGroupID: 2, Name: "Transporte", Icon: "car"},
			},
		},
		{
			ID: 3, Name: "Ahorros", Icon: "piggy", SortOrder: 2,
			Categories: []api.Category{
				{ID: 7, CategoryGroupID: 3, Name: "Ahorro", Icon: "wallet", AccountID: &savings},
			},
		},
	}
}

func demoAccounts() []api.Account {
	return []api.Account{
		{ID: 3, Name: "Cuenta Demo", Type: "checking", CurrencyID: 1, CurrencyCode: "NIO", StartingBalance: 1000, Balance: 850},
		{ID: 4, Name: "Ahorro Demo", Type: "savings", CurrencyID: 2, CurrencyCode: "USD", StartingBalance: 100, Balance: 100},
	}
}

func newTestSession(t *testing.T, fake *fakeAPI, cfg config.Config) *mcp.ClientSession {
	t.Helper()
	cfg.Location = time.UTC
	toolset := New(fake, cfg)
	toolset.now = func() time.Time { return time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC) }
	return toolkitest.NewServer(t, toolset.Register)
}

func readPayload(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	if result == nil || len(result.Content) == 0 {
		t.Fatal("resultado sin contenido")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("contenido inesperado: %T", result.Content[0])
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Fatalf("decodificar payload %q: %v", text.Text, err)
	}
	return payload
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (map[string]any, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("llamar %s: %v", name, err)
	}
	return readPayload(t, result), result.IsError
}

func TestToolsetRegistersFiveTools(t *testing.T) {
	session := newTestSession(t, &fakeAPI{}, config.Config{})
	names := toolkitest.ToolNames(t, session)
	for _, name := range []string{
		"budget_categories_list",
		"budget_accounts_list",
		"budget_transactions_create",
		"budget_transactions_list",
		"budget_month_view",
	} {
		if !names[name] {
			t.Fatalf("falta la tool %s en %v", name, names)
		}
	}
	if len(names) != 5 {
		t.Fatalf("tools = %d, quiero 5: %v", len(names), names)
	}
}

func TestCategoriesList(t *testing.T) {
	session := newTestSession(t, &fakeAPI{groups: demoGroups()}, config.Config{})
	payload, isError := callTool(t, session, "budget_categories_list", map[string]any{})
	if isError {
		t.Fatalf("error inesperado: %v", payload)
	}
	if payload["count"] != float64(2) {
		t.Fatalf("count = %v", payload["count"])
	}
	groups, ok := payload["groups"].([]any)
	if !ok || len(groups) != 2 {
		t.Fatalf("grupos inesperados: %v", payload["groups"])
	}
	first := groups[0].(map[string]any)
	categories := first["categories"].([]any)
	category := categories[0].(map[string]any)
	if category["name"] != "Comida" || category["account_id"] != float64(3) {
		t.Fatalf("categoría inesperada: %v", category)
	}
}

func TestCategoriesListEmptyIsArray(t *testing.T) {
	session := newTestSession(t, &fakeAPI{}, config.Config{})
	payload, isError := callTool(t, session, "budget_categories_list", map[string]any{})
	if isError {
		t.Fatalf("error inesperado: %v", payload)
	}
	groups, ok := payload["groups"].([]any)
	if !ok || len(groups) != 0 {
		t.Fatalf("grupos = %v, quiero []", payload["groups"])
	}
}

func TestAccountsList(t *testing.T) {
	session := newTestSession(t, &fakeAPI{accounts: demoAccounts()}, config.Config{})
	payload, isError := callTool(t, session, "budget_accounts_list", map[string]any{})
	if isError {
		t.Fatalf("error inesperado: %v", payload)
	}
	if payload["count"] != float64(2) {
		t.Fatalf("count = %v", payload["count"])
	}
	accounts := payload["accounts"].([]any)
	first := accounts[0].(map[string]any)
	if first["name"] != "Cuenta Demo" || first["currency_code"] != "NIO" || first["balance"] != float64(850) {
		t.Fatalf("cuenta inesperada: %v", first)
	}
}

func TestTransactionsList(t *testing.T) {
	categoryID := int64(5)
	fake := &fakeAPI{transactions: []api.Transaction{{
		ID: 10, AccountID: 3, CategoryID: &categoryID, CurrencyID: 1,
		CurrencyCode: "NIO", Date: "2026-03-10", Payee: "Proveedor Demo", Outflow: 1200,
	}}}
	session := newTestSession(t, fake, config.Config{})
	payload, isError := callTool(t, session, "budget_transactions_list", map[string]any{"year": 2026, "month": 3})
	if isError {
		t.Fatalf("error inesperado: %v", payload)
	}
	if fake.lastYear != 2026 || fake.lastMonth != 3 {
		t.Fatalf("período = %d-%d", fake.lastYear, fake.lastMonth)
	}
	if payload["count"] != float64(1) || payload["year"] != float64(2026) || payload["month"] != float64(3) {
		t.Fatalf("payload inesperado: %v", payload)
	}
}

func TestTransactionsListInvalidPeriod(t *testing.T) {
	session := newTestSession(t, &fakeAPI{}, config.Config{})
	for _, args := range []map[string]any{
		{"year": 2026, "month": 13},
		{"year": 2026, "month": 0},
		{"year": 1800, "month": 3},
	} {
		payload, isError := callTool(t, session, "budget_transactions_list", args)
		if !isError || payload["error"] != "invalid_request" {
			t.Fatalf("args %v: esperaba invalid_request, obtuve %v", args, payload)
		}
	}
}

func TestMonthView(t *testing.T) {
	fake := &fakeAPI{view: &api.MonthView{
		Year: 2026, Month: 3,
		CurrencyTotals: []api.MonthCurrencyTotal{{CurrencyID: 1, Code: "NIO", Symbol: "C$"}},
		Groups: []api.MonthGroup{{
			ID: 2, Name: "Necesidades",
			Categories: []api.MonthCategory{{
				Category:  api.Category{ID: 5, Name: "Comida"},
				Assigned:  map[int64]float64{1: 5000},
				Activity:  map[int64]float64{1: 1200},
				Available: map[int64]float64{1: 3800},
			}},
		}},
	}}
	session := newTestSession(t, fake, config.Config{})
	payload, isError := callTool(t, session, "budget_month_view", map[string]any{"year": 2026, "month": 3})
	if isError {
		t.Fatalf("error inesperado: %v", payload)
	}
	if fake.lastViewYear != 2026 || fake.lastViewMonth != 3 {
		t.Fatalf("período = %d-%d", fake.lastViewYear, fake.lastViewMonth)
	}
	groups := payload["groups"].([]any)
	group := groups[0].(map[string]any)
	category := group["categories"].([]any)[0].(map[string]any)
	available := category["available"].(map[string]any)
	if available["1"] != float64(3800) {
		t.Fatalf("available inesperado: %v", category)
	}
}

func TestCreatePlannedByDefault(t *testing.T) {
	fake := &fakeAPI{groups: demoGroups(), accounts: demoAccounts()}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200, "category_id": 5, "payee": "Proveedor Demo",
	})
	if isError {
		t.Fatalf("error inesperado: %v", payload)
	}
	if payload["status"] != "planned" || payload["dry_run"] != true {
		t.Fatalf("payload inesperado: %v", payload)
	}
	if fake.createCalls != 0 {
		t.Fatalf("createCalls = %d, quiero 0", fake.createCalls)
	}
	if fake.listCalls != 1 {
		t.Fatalf("listCalls = %d, quiero 1", fake.listCalls)
	}
	plan := payload["plan"].(map[string]any)
	if plan["account_id"] != float64(3) || plan["category_id"] != float64(5) ||
		plan["currency_code"] != "NIO" || plan["amount"] != float64(1200) ||
		plan["date"] != "2026-03-10" || plan["category_name"] != "Comida" {
		t.Fatalf("plan inesperado: %v", plan)
	}
}

func TestCreateWritesDisabled(t *testing.T) {
	fake := &fakeAPI{groups: demoGroups(), accounts: demoAccounts()}
	session := newTestSession(t, fake, config.Config{})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200, "category_id": 5, "dry_run": false,
	})
	if !isError || payload["error"] != "writes_disabled" {
		t.Fatalf("esperaba writes_disabled: %v", payload)
	}
	if fake.createCalls != 0 {
		t.Fatalf("createCalls = %d, quiero 0", fake.createCalls)
	}
}

func TestCreateSuccess(t *testing.T) {
	fake := &fakeAPI{groups: demoGroups(), accounts: demoAccounts()}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200, "category": "Comida",
		"payee": "  Proveedor Demo  ", "memo": "  Compra demo  ", "dry_run": false,
	})
	if isError {
		t.Fatalf("error inesperado: %v", payload)
	}
	if payload["status"] != "created" || payload["dry_run"] != false {
		t.Fatalf("payload inesperado: %v", payload)
	}
	if fake.createCalls != 1 {
		t.Fatalf("createCalls = %d, quiero 1", fake.createCalls)
	}
	req := fake.lastCreate
	if req.AccountID != 3 || req.CategoryID == nil || *req.CategoryID != 5 ||
		req.CurrencyID != 1 || req.Date != "2026-03-10" || req.Payee != "Proveedor Demo" ||
		req.Memo != "Compra demo" || req.Outflow != 1200 || req.Inflow != 0 || req.Cleared {
		t.Fatalf("request inesperado: %+v", req)
	}
	if fake.lastYear != 2026 || fake.lastMonth != 3 {
		t.Fatalf("dedup consultó %d-%d", fake.lastYear, fake.lastMonth)
	}
	transaction := payload["transaction"].(map[string]any)
	if transaction["id"] != float64(10) || transaction["outflow"] != float64(1200) {
		t.Fatalf("transacción inesperada: %v", transaction)
	}
}

func TestCreateDuplicate(t *testing.T) {
	categoryID := int64(5)
	fake := &fakeAPI{
		groups:   demoGroups(),
		accounts: demoAccounts(),
		transactions: []api.Transaction{{
			ID: 9, AccountID: 3, AccountName: "Cuenta Demo", CategoryID: &categoryID,
			CategoryName: "Comida", CurrencyID: 1, Date: "2026-03-10",
			Payee: "Proveedor Demo", Outflow: 1200,
		}},
	}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200, "category_id": 5, "payee": "Proveedor Demo", "dry_run": false,
	})
	if isError {
		t.Fatalf("error inesperado: %v", payload)
	}
	if payload["status"] != "duplicate" {
		t.Fatalf("status = %v, quiero duplicate", payload["status"])
	}
	if fake.createCalls != 0 {
		t.Fatalf("createCalls = %d, quiero 0", fake.createCalls)
	}
	existing := payload["existing"].(map[string]any)
	if existing["id"] != float64(9) || existing["outflow"] != float64(1200) {
		t.Fatalf("existing inesperado: %v", existing)
	}
	if payload["hint"] == "" {
		t.Fatal("falta el hint del duplicado")
	}
}

func TestCreateDuplicateNormalizesPayee(t *testing.T) {
	categoryID := int64(5)
	fake := &fakeAPI{
		groups:   demoGroups(),
		accounts: demoAccounts(),
		transactions: []api.Transaction{{
			ID: 9, CategoryID: &categoryID, Outflow: 1200, Date: "2026-03-10",
			Payee: "  PROVEEDOR   demo ",
		}},
	}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200, "category_id": 5, "payee": "Proveedor Demo", "dry_run": false,
	})
	if isError || payload["status"] != "duplicate" {
		t.Fatalf("esperaba duplicate: %v", payload)
	}

	// Un monto distinto no es duplicado.
	fake.transactions[0].Outflow = 1300
	payload, isError = callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200, "category_id": 5, "payee": "Proveedor Demo", "dry_run": false,
	})
	if isError || payload["status"] != "created" {
		t.Fatalf("esperaba created con monto distinto: %v", payload)
	}
}

func TestCreateAllowDuplicate(t *testing.T) {
	categoryID := int64(5)
	fake := &fakeAPI{
		groups:   demoGroups(),
		accounts: demoAccounts(),
		transactions: []api.Transaction{{
			ID: 9, CategoryID: &categoryID, Outflow: 1200, Date: "2026-03-10", Payee: "Proveedor Demo",
		}},
	}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200, "category_id": 5, "payee": "Proveedor Demo",
		"dry_run": false, "allow_duplicate": true,
	})
	if isError {
		t.Fatalf("error inesperado: %v", payload)
	}
	if payload["status"] != "created" {
		t.Fatalf("status = %v, quiero created", payload["status"])
	}
	if fake.createCalls != 1 {
		t.Fatalf("createCalls = %d, quiero 1", fake.createCalls)
	}
	warnings, ok := payload["warnings"].([]any)
	if !ok || len(warnings) != 1 || warnings[0] != "possible_duplicate" {
		t.Fatalf("warnings inesperados: %v", payload["warnings"])
	}
}

func TestCreateAmbiguousCategory(t *testing.T) {
	other := int64(5)
	fake := &fakeAPI{
		groups: []api.CategoryGroup{
			{ID: 2, Name: "Necesidades", Categories: []api.Category{
				{ID: 5, CategoryGroupID: 2, Name: "Comida"},
			}},
			{ID: 3, Name: "Gustos", Categories: []api.Category{
				{ID: 8, CategoryGroupID: 3, Name: "Comida"},
			}},
		},
		accounts:     demoAccounts(),
		transactions: []api.Transaction{{ID: 9, CategoryID: &other, Outflow: 1, Date: "2026-03-10"}},
	}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200, "category": "comida",
	})
	if !isError || payload["error"] != "ambiguous_category" {
		t.Fatalf("esperaba ambiguous_category: %v", payload)
	}
	candidates, ok := payload["candidates"].([]any)
	if !ok || len(candidates) != 2 {
		t.Fatalf("candidatas inesperadas: %v", payload["candidates"])
	}
	first := candidates[0].(map[string]any)
	if first["id"] != float64(5) || first["group_name"] != "Necesidades" {
		t.Fatalf("candidata inesperada: %v", first)
	}
	if fake.createCalls != 0 {
		t.Fatalf("createCalls = %d, quiero 0", fake.createCalls)
	}
}

func TestCreateCategoryNotFound(t *testing.T) {
	fake := &fakeAPI{groups: demoGroups(), accounts: demoAccounts()}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200, "category": "Inexistente",
	})
	if !isError || payload["error"] != "category_not_found" {
		t.Fatalf("esperaba category_not_found: %v", payload)
	}
	if fake.createCalls != 0 {
		t.Fatalf("createCalls = %d, quiero 0", fake.createCalls)
	}
}

func TestCreateMissingCategory(t *testing.T) {
	fake := &fakeAPI{groups: demoGroups(), accounts: demoAccounts()}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200,
	})
	if !isError || payload["error"] != "invalid_request" {
		t.Fatalf("esperaba invalid_request: %v", payload)
	}
}

func TestCreateCategoryCoherence(t *testing.T) {
	fake := &fakeAPI{groups: demoGroups(), accounts: demoAccounts()}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200, "category_id": 5, "category": "Transporte",
	})
	if !isError || payload["error"] != "invalid_request" {
		t.Fatalf("esperaba invalid_request por incoherencia: %v", payload)
	}
	if fake.createCalls != 0 {
		t.Fatalf("createCalls = %d, quiero 0", fake.createCalls)
	}
}

func TestCreateAccountFromCategory(t *testing.T) {
	fake := &fakeAPI{groups: demoGroups(), accounts: demoAccounts()}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 50, "category_id": 7, "dry_run": false,
	})
	if isError {
		t.Fatalf("error inesperado: %v", payload)
	}
	if fake.lastCreate.AccountID != 4 || fake.lastCreate.CurrencyID != 2 {
		t.Fatalf("cuenta/moneda inesperadas: %+v", fake.lastCreate)
	}
}

func TestCreateAccountDefaultsToFirstActive(t *testing.T) {
	fake := &fakeAPI{groups: demoGroups(), accounts: demoAccounts()}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 50, "category_id": 6, "dry_run": false,
	})
	if isError {
		t.Fatalf("error inesperado: %v", payload)
	}
	if fake.lastCreate.AccountID != 3 || fake.lastCreate.CurrencyID != 1 {
		t.Fatalf("cuenta/moneda inesperadas: %+v", fake.lastCreate)
	}
}

func TestCreateAccountNotFound(t *testing.T) {
	fake := &fakeAPI{groups: demoGroups()}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 50, "category_id": 5, "dry_run": false,
	})
	if !isError || payload["error"] != "account_not_found" {
		t.Fatalf("esperaba account_not_found: %v", payload)
	}
	if payload["hint"] == "" {
		t.Fatal("falta hint para crear una cuenta")
	}
	if fake.createCalls != 0 {
		t.Fatalf("createCalls = %d, quiero 0", fake.createCalls)
	}
}

func TestCreateInvalidDate(t *testing.T) {
	fake := &fakeAPI{groups: demoGroups(), accounts: demoAccounts()}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	for _, date := range []string{"2026-3-1", "2026-03-10T10:00:00", "2026-04-01"} {
		payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
			"date": date, "amount": 1200, "category_id": 5,
		})
		if !isError || payload["error"] != "invalid_date" {
			t.Fatalf("fecha %q: esperaba invalid_date, obtuve %v", date, payload)
		}
	}
	if fake.createCalls != 0 {
		t.Fatalf("createCalls = %d, quiero 0", fake.createCalls)
	}
}

func TestCreateInvalidAmount(t *testing.T) {
	fake := &fakeAPI{groups: demoGroups(), accounts: demoAccounts()}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	for _, amount := range []float64{0, -5} {
		payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
			"date": "2026-03-10", "amount": amount, "category_id": 5,
		})
		if !isError || payload["error"] != "invalid_amount" {
			t.Fatalf("monto %v: esperaba invalid_amount, obtuve %v", amount, payload)
		}
	}
}

func TestCreateRoundsAmount(t *testing.T) {
	fake := &fakeAPI{groups: demoGroups(), accounts: demoAccounts()}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200.456, "category_id": 5, "dry_run": false,
	})
	if isError {
		t.Fatalf("error inesperado: %v", payload)
	}
	if fake.lastCreate.Outflow != 1200.46 {
		t.Fatalf("outflow = %v, quiero 1200.46", fake.lastCreate.Outflow)
	}
}

func TestCreateEmptyPayeeMatchesEmptyPayee(t *testing.T) {
	categoryID := int64(5)
	fake := &fakeAPI{
		groups:   demoGroups(),
		accounts: demoAccounts(),
		transactions: []api.Transaction{{
			ID: 9, CategoryID: &categoryID, Outflow: 1200, Date: "2026-03-10", Payee: "",
		}},
	}
	session := newTestSession(t, fake, config.Config{AllowWrites: true})

	payload, isError := callTool(t, session, "budget_transactions_create", map[string]any{
		"date": "2026-03-10", "amount": 1200, "category_id": 5,
	})
	if isError || payload["status"] != "duplicate" {
		t.Fatalf("esperaba duplicate sin payee: %v", payload)
	}
}
