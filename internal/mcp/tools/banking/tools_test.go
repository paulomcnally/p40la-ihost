package banking

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

type fakeAPI struct {
	banks        []api.BankRef
	currencies   []api.CurrencyRef
	accounts     map[int64][]api.BankAccount
	cards        map[int64][]api.CreditCard
	getAccount   *api.BankAccount
	getCard      *api.CreditCard
	createdBank  int64
	createdAcc   api.BankAccountInput
	updatedAcc   api.BankAccountInput
	updatedAccID int64
	createdCard  api.CreditCardInput
	updatedCard  api.CreditCardInput
	updatedCardI int64
	err          error
	calls        int
}

func (f *fakeAPI) GetService(context.Context, int64) (*api.Service, error) { return nil, nil }
func (f *fakeAPI) ListBills(context.Context, int64) ([]api.Bill, error)    { return nil, nil }
func (f *fakeAPI) GenerateBills(context.Context, int64, api.GenerateRequest) (*api.GenerateResult, error) {
	return nil, nil
}
func (f *fakeAPI) PayBills(context.Context, int64, api.PayRequest) (*api.PayResult, error) {
	return nil, nil
}

func (f *fakeAPI) ListBankRefs(context.Context) ([]api.BankRef, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.banks, nil
}

func (f *fakeAPI) ListCurrencyRefs(context.Context) ([]api.CurrencyRef, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.currencies, nil
}

func (f *fakeAPI) ListBankAccounts(_ context.Context, bankID int64) ([]api.BankAccount, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.accounts[bankID], nil
}

func (f *fakeAPI) GetBankAccount(context.Context, int64) (*api.BankAccount, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.getAccount, nil
}

func (f *fakeAPI) CreateBankAccount(_ context.Context, bankID int64, input api.BankAccountInput) (*api.BankAccount, error) {
	f.calls++
	f.createdBank = bankID
	f.createdAcc = input
	return &api.BankAccount{
		ID:            99,
		BankID:        bankID,
		BankName:      "Banco Demo",
		Alias:         input.Alias,
		AccountNumber: input.AccountNumber,
		AccountType:   input.AccountType,
		CurrencyID:    input.CurrencyID,
		CurrencyCode:  "NIO",
		Beneficiary:   input.Beneficiary,
		Notes:         input.Notes,
		Active:        input.Active,
	}, nil
}

func (f *fakeAPI) UpdateBankAccount(_ context.Context, id int64, input api.BankAccountInput) (*api.BankAccount, error) {
	f.calls++
	f.updatedAccID = id
	f.updatedAcc = input
	return &api.BankAccount{
		ID:            id,
		BankID:        1,
		BankName:      "Banco Demo",
		Alias:         input.Alias,
		AccountNumber: input.AccountNumber,
		AccountType:   input.AccountType,
		CurrencyID:    input.CurrencyID,
		CurrencyCode:  "USD",
		Beneficiary:   input.Beneficiary,
		Notes:         input.Notes,
		Active:        input.Active,
	}, nil
}

func (f *fakeAPI) ListCreditCards(_ context.Context, bankID int64) ([]api.CreditCard, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.cards[bankID], nil
}

func (f *fakeAPI) GetCreditCard(context.Context, int64) (*api.CreditCard, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.getCard, nil
}

func (f *fakeAPI) CreateCreditCard(_ context.Context, bankID int64, input api.CreditCardInput) (*api.CreditCard, error) {
	f.calls++
	f.createdBank = bankID
	f.createdCard = input
	return &api.CreditCard{
		ID:                  77,
		BankID:              bankID,
		BankName:            "Banco Demo",
		Alias:               input.Alias,
		Network:             input.Network,
		Last4:               input.Last4,
		Holder:              input.Holder,
		CurrencyID:          input.CurrencyID,
		CurrencyCode:        "NIO",
		Category:            input.Category,
		LimitCurrency:       input.LimitCurrency,
		CreditLimit:         input.CreditLimit,
		CreditLimitUSD:      input.CreditLimitUSD,
		StatementDay:        input.StatementDay,
		PaymentDueDay:       input.PaymentDueDay,
		BonusDueDay:         input.BonusDueDay,
		MinPaymentPct:       input.MinPaymentPct,
		Color:               input.Color,
		Icon:                input.Icon,
		Active:              input.Active,
		Notes:               input.Notes,
		SecondaryCurrencyID: input.SecondaryCurrencyID,
	}, nil
}

func (f *fakeAPI) UpdateCreditCard(_ context.Context, id int64, input api.CreditCardInput) (*api.CreditCard, error) {
	f.calls++
	f.updatedCardI = id
	f.updatedCard = input
	return &api.CreditCard{
		ID:            id,
		BankID:        1,
		BankName:      "Banco Demo",
		Alias:         input.Alias,
		Network:       input.Network,
		Last4:         input.Last4,
		Holder:        input.Holder,
		CurrencyID:    input.CurrencyID,
		CurrencyCode:  "NIO",
		LimitCurrency: input.LimitCurrency,
		CreditLimit:   input.CreditLimit,
		StatementDay:  input.StatementDay,
		PaymentDueDay: input.PaymentDueDay,
		MinPaymentPct: input.MinPaymentPct,
		Color:         input.Color,
		Icon:          input.Icon,
		Active:        input.Active,
		Notes:         input.Notes,
	}, nil
}

func (f *fakeAPI) ListHomes(context.Context) ([]api.Home, error) { return nil, nil }
func (f *fakeAPI) CreateHome(context.Context, api.HomeCreateRequest) (*api.Home, error) {
	return nil, nil
}
func (f *fakeAPI) UpdateHome(context.Context, int64, api.HomeUpdateRequest) (*api.Home, error) {
	return nil, nil
}
func (f *fakeAPI) ListInstitutions(context.Context) ([]api.Institution, error) { return nil, nil }
func (f *fakeAPI) CreateInstitution(context.Context, api.InstitutionCreateRequest) (*api.Institution, error) {
	return nil, nil
}
func (f *fakeAPI) ListBanks(context.Context) ([]api.Bank, error) { return nil, nil }
func (f *fakeAPI) CreateBank(context.Context, api.BankCreateRequest) (*api.Bank, error) {
	return nil, nil
}
func (f *fakeAPI) ListCurrencies(context.Context) ([]api.Currency, error) { return nil, nil }
func (f *fakeAPI) ListInstitutionCategories(context.Context) ([]api.InstitutionCategory, error) {
	return nil, nil
}
func (f *fakeAPI) ListServices(context.Context, *int64) ([]api.Service, error) { return nil, nil }
func (f *fakeAPI) CreateService(context.Context, api.ServiceCreateRequest) (*api.Service, error) {
	return nil, nil
}

func testToolset(allowWrites bool) (*Toolset, *fakeAPI) {
	fake := &fakeAPI{
		banks: []api.BankRef{
			{ID: 1, Name: "Banco Demo", ShortCode: "BD", Active: true},
			{ID: 2, Name: "Banco Otro", ShortCode: "BO", Active: true},
		},
		currencies: []api.CurrencyRef{
			{ID: 1, Code: "NIO", Name: "Córdoba"},
			{ID: 2, Code: "USD", Name: "Dólar"},
		},
		accounts: map[int64][]api.BankAccount{},
		cards:    map[int64][]api.CreditCard{},
	}
	return New(fake, config.Config{AllowWrites: allowWrites, Timezone: "UTC"}), fake
}

func strPtr(value string) *string       { return &value }
func boolPtrOf(value bool) *bool        { return &value }
func intPtrOf(value int) *int           { return &value }
func int64PtrOf(value int64) *int64     { return &value }
func floatPtrOf(value float64) *float64 { return &value }

func resultErrorCode(t *testing.T, result any) string {
	t.Helper()
	payload, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("resultado de error inesperado: %#v", result)
	}
	code, _ := payload["error"].(string)
	return code
}

func TestFoldNormalizesAccents(t *testing.T) {
	if got := fold("  Banco DÉMO  "); got != "banco demo" {
		t.Fatalf("fold: got %q", got)
	}
}

func TestAccountSearchOrdersFiltersAndTruncates(t *testing.T) {
	toolset, fake := testToolset(false)
	fake.accounts[1] = []api.BankAccount{
		{ID: 1, BankID: 1, BankName: "Banco Demo", Alias: "Cuenta Demo", AccountNumber: "demo-account-001", Beneficiary: "Titular Demo", Active: true},
		{ID: 2, BankID: 1, BankName: "Banco Demo", Alias: "Otra Cuenta", AccountNumber: "demo-account-002", Beneficiary: "Titular Demo", Active: false},
	}
	fake.accounts[2] = []api.BankAccount{
		{ID: 3, BankID: 2, BankName: "Banco Otro", Alias: "Cuenta Demo Dos", AccountNumber: "demo-account-003", Beneficiary: "Titular Demo", Active: false},
	}

	_, output, err := toolset.accountsSearch(context.Background(), nil, AccountSearchInput{
		Query: strPtr("cuenta demo"),
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	result := output.(AccountSearchOutput)
	if result.Total != 2 || result.Count != 2 || result.Truncated {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if result.Items[0].ID != 1 || result.Items[1].ID != 3 {
		t.Fatalf("orden inesperado: %+v", result.Items)
	}

	_, output, err = toolset.accountsSearch(context.Background(), nil, AccountSearchInput{
		Query:  strPtr("demo"),
		Active: boolPtrOf(false),
		Limit:  intPtrOf(1),
	})
	if err != nil {
		t.Fatalf("search active: %v", err)
	}
	filtered := output.(AccountSearchOutput)
	if filtered.Count != 1 || !filtered.Truncated || filtered.Total != 2 || filtered.Items[0].ID != 3 {
		t.Fatalf("filtro inesperado: %+v", filtered)
	}
}

func TestAccountSearchByBankAvoidsOtherBanks(t *testing.T) {
	toolset, fake := testToolset(false)
	fake.accounts[1] = []api.BankAccount{{ID: 1, BankID: 1, Alias: "Cuenta Demo", Active: true}}
	fake.accounts[2] = []api.BankAccount{{ID: 2, BankID: 2, Alias: "Otra", Active: true}}

	_, output, err := toolset.accountsSearch(context.Background(), nil, AccountSearchInput{
		Bank: strPtr("banco otro"),
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	result := output.(AccountSearchOutput)
	if result.Total != 1 || result.Items[0].ID != 2 {
		t.Fatalf("resultado inesperado: %+v", result)
	}
}

func TestSearchRejectsInvalidLimit(t *testing.T) {
	toolset, _ := testToolset(false)
	result, output, err := toolset.accountsSearch(context.Background(), nil, AccountSearchInput{Limit: intPtrOf(0)})
	if err != nil || result == nil {
		t.Fatalf("se esperaba error: %v", err)
	}
	if code := resultErrorCode(t, output); code != "invalid_value" {
		t.Fatalf("código inesperado: %s", code)
	}
}

func TestCreateAccountResolvesReferencesAndDefaults(t *testing.T) {
	toolset, fake := testToolset(true)

	_, output, err := toolset.accountsCreate(context.Background(), nil, AccountCreateInput{
		Bank:          strPtr("Banco Demo"),
		Alias:         "  Cuenta Nueva  ",
		AccountNumber: "  demo-account-001  ",
		Currency:      strPtr("NIO"),
		Beneficiary:   "  Titular Demo  ",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	item := output.(AccountItem)
	if item.ID != 99 || fake.createdBank != 1 {
		t.Fatalf("resultado inesperado: %+v bank=%d", item, fake.createdBank)
	}
	if fake.createdAcc.AccountType != "savings" || !fake.createdAcc.Active {
		t.Fatalf("defaults inesperados: %+v", fake.createdAcc)
	}
	if fake.createdAcc.CurrencyID != 1 || fake.createdAcc.Alias != "Cuenta Nueva" || fake.createdAcc.AccountNumber != "demo-account-001" {
		t.Fatalf("payload inesperado: %+v", fake.createdAcc)
	}
}

func TestCreateAccountRejectsDuplicate(t *testing.T) {
	toolset, fake := testToolset(true)
	fake.accounts[1] = []api.BankAccount{{ID: 5, BankID: 1, Alias: "Cuenta Demo", AccountNumber: "demo-account-001", Active: true}}

	result, output, err := toolset.accountsCreate(context.Background(), nil, AccountCreateInput{
		Bank:          strPtr("Banco Demo"),
		Alias:         "Cuenta Demo",
		AccountNumber: "DEMO-ACCOUNT-001",
		Currency:      strPtr("NIO"),
		Beneficiary:   "Titular Demo",
	})
	if err != nil || result == nil {
		t.Fatalf("se esperaba error: %v", err)
	}
	if code := resultErrorCode(t, output); code != "already_exists" {
		t.Fatalf("código inesperado: %s", code)
	}
}

func TestWritesDisabledBlocksCreate(t *testing.T) {
	toolset, fake := testToolset(false)
	result, output, err := toolset.accountsCreate(context.Background(), nil, AccountCreateInput{
		Bank: strPtr("Banco Demo"), Alias: "Cuenta Demo", AccountNumber: "demo-account-001",
		Currency: strPtr("NIO"), Beneficiary: "Titular Demo",
	})
	if err != nil || result == nil {
		t.Fatalf("se esperaba error: %v", err)
	}
	if code := resultErrorCode(t, output); code != "writes_disabled" {
		t.Fatalf("código inesperado: %s", code)
	}
	if fake.calls != 0 {
		t.Fatalf("no debía llamar a la API: %d", fake.calls)
	}
}

func TestUpdateAccountKeepsUnsentFields(t *testing.T) {
	toolset, fake := testToolset(true)
	fake.getAccount = &api.BankAccount{
		ID: 5, BankID: 1, BankName: "Banco Demo", Alias: "Vieja", AccountNumber: "demo-account-001",
		AccountType: "checking", CurrencyID: 2, CurrencyCode: "USD", Beneficiary: "Titular Demo",
		Notes: "nota", Active: true,
	}

	_, _, err := toolset.accountsUpdate(context.Background(), nil, AccountUpdateInput{
		ID:    5,
		Alias: strPtr("Nueva"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if fake.updatedAccID != 5 {
		t.Fatalf("id inesperado: %d", fake.updatedAccID)
	}
	if fake.updatedAcc.Alias != "Nueva" || fake.updatedAcc.AccountNumber != "demo-account-001" ||
		fake.updatedAcc.AccountType != "checking" || fake.updatedAcc.CurrencyID != 2 ||
		fake.updatedAcc.Beneficiary != "Titular Demo" || !fake.updatedAcc.Active {
		t.Fatalf("merge inesperado: %+v", fake.updatedAcc)
	}
}

func TestUpdateAccountRejectsBankChange(t *testing.T) {
	toolset, fake := testToolset(true)
	result, output, err := toolset.accountsUpdate(context.Background(), nil, AccountUpdateInput{
		ID:     5,
		BankID: int64PtrOf(2),
	})
	if err != nil || result == nil {
		t.Fatalf("se esperaba error: %v", err)
	}
	if code := resultErrorCode(t, output); code != "invalid_value" {
		t.Fatalf("código inesperado: %s", code)
	}
	if fake.calls != 0 {
		t.Fatalf("no debía llamar a la API: %d", fake.calls)
	}
}

func TestCreateCardDefaults(t *testing.T) {
	toolset, fake := testToolset(true)

	_, _, err := toolset.cardsCreate(context.Background(), nil, CardCreateInput{
		Bank:     strPtr("Banco Demo"),
		Alias:    "  Tarjeta Demo  ",
		Network:  "VISA",
		Last4:    " 0000 ",
		Currency: strPtr("NIO"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	payload := fake.createdCard
	if payload.Alias != "Tarjeta Demo" || payload.Network != "visa" || payload.Last4 != "0000" {
		t.Fatalf("normalización inesperada: %+v", payload)
	}
	if payload.StatementDay != 1 || payload.PaymentDueDay != 1 || payload.MinPaymentPct != 4 {
		t.Fatalf("defaults de días inesperados: %+v", payload)
	}
	if payload.LimitCurrency != "NIO" || payload.Color != "blue" || payload.Icon != "credit" || !payload.Active {
		t.Fatalf("defaults visuales inesperados: %+v", payload)
	}
	if payload.SecondaryCurrencyID != nil || payload.CurrencyID != 1 {
		t.Fatalf("monedas inesperadas: %+v", payload)
	}
}

func TestCreateCardValidatesLast4(t *testing.T) {
	toolset, _ := testToolset(true)
	result, output, err := toolset.cardsCreate(context.Background(), nil, CardCreateInput{
		Bank: strPtr("Banco Demo"), Alias: "Tarjeta Demo", Network: "visa", Last4: "12a4", Currency: strPtr("NIO"),
	})
	if err != nil || result == nil {
		t.Fatalf("se esperaba error: %v", err)
	}
	if code := resultErrorCode(t, output); code != "invalid_value" {
		t.Fatalf("código inesperado: %s", code)
	}
}

func TestCardSearchByLast4(t *testing.T) {
	toolset, fake := testToolset(false)
	fake.cards[1] = []api.CreditCard{
		{ID: 1, BankID: 1, BankName: "Banco Demo", Alias: "Tarjeta Demo", Last4: "0000", Network: "visa", Active: true},
		{ID: 2, BankID: 1, BankName: "Banco Demo", Alias: "Otra Tarjeta", Last4: "1111", Network: "mastercard", Active: true},
	}

	_, output, err := toolset.cardsSearch(context.Background(), nil, CardSearchInput{Query: strPtr("0000")})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	result := output.(CardSearchOutput)
	if result.Total != 1 || result.Items[0].ID != 1 {
		t.Fatalf("resultado inesperado: %+v", result)
	}
}

func TestUpdateCardClearsSecondaryCurrency(t *testing.T) {
	toolset, fake := testToolset(true)
	secondary := int64(2)
	fake.getCard = &api.CreditCard{
		ID: 7, BankID: 1, BankName: "Banco Demo", Alias: "Tarjeta Demo", Network: "visa", Last4: "0000",
		CurrencyID: 1, CurrencyCode: "NIO", SecondaryCurrencyID: &secondary, SecondaryCurrencyCode: "USD",
		LimitCurrency: "NIO", StatementDay: 1, PaymentDueDay: 1, MinPaymentPct: 4, Color: "blue", Icon: "credit", Active: true,
	}

	_, _, err := toolset.cardsUpdate(context.Background(), nil, CardUpdateInput{
		ID:                7,
		SecondaryCurrency: strPtr(""),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if fake.updatedCard.SecondaryCurrencyID != nil {
		t.Fatalf("la moneda secundaria debía quedar vacía: %+v", fake.updatedCard)
	}
	if fake.updatedCard.Alias != "Tarjeta Demo" || fake.updatedCard.Last4 != "0000" || fake.updatedCard.StatementDay != 1 {
		t.Fatalf("merge inesperado: %+v", fake.updatedCard)
	}
}

func TestResolveBankAmbiguous(t *testing.T) {
	toolset, fake := testToolset(false)
	fake.banks = []api.BankRef{
		{ID: 1, Name: "Banco del Norte"},
		{ID: 2, Name: "Banco del Sur"},
	}
	_, err := toolset.resolveBank(context.Background(), "Banco del", 0)
	if err == nil {
		t.Fatal("se esperaba ambigüedad")
	}
	lookup, ok := err.(*lookupError)
	if !ok || lookup.Code != "ambiguous_reference" {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestResolveBankExactWins(t *testing.T) {
	toolset, fake := testToolset(false)
	fake.banks = []api.BankRef{
		{ID: 1, Name: "Banco Demo"},
		{ID: 2, Name: "Banco Demo Dos"},
	}
	bank, err := toolset.resolveBank(context.Background(), "banco demo", 0)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if bank.ID != 1 {
		t.Fatalf("banco inesperado: %+v", bank)
	}
}

func TestProtocolToolsRegistered(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "p40la-mcp-test", Version: "0.1.0"}, nil)
	toolset, _ := testToolset(false)
	toolset.Register(server)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("conectar server: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.1.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("conectar cliente: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	names := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{
		"bank_accounts_search", "bank_accounts_create", "bank_accounts_update",
		"credit_cards_search", "credit_cards_create", "credit_cards_update",
	} {
		if !names[want] {
			t.Fatalf("falta la tool %q en %v", want, names)
		}
	}
}
