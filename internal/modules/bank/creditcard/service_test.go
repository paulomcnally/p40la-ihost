package creditcard

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/modules"
)

type fakeImporter struct {
	parsed *models.ParsedStatement
	err    error
}

func (f *fakeImporter) Info() modules.Info {
	return modules.Info{ID: "bank.test.ni.creditcard.statement.import", Version: "1.0.0", Kind: "provider"}
}

func (f *fakeImporter) CanParse(filename, mimeType string) bool { return true }

func (f *fakeImporter) Parse(ctx context.Context, r io.Reader) (*models.ParsedStatement, error) {
	return f.parsed, f.err
}

type fakeLookup struct {
	importer modules.StatementImporter
}

func (l fakeLookup) FindImporter(filename, mimeType string) (modules.StatementImporter, bool) {
	if l.importer == nil {
		return nil, false
	}
	return l.importer, true
}

func (l fakeLookup) ParseStatements(ctx context.Context, filename, mimeType string, data []byte) (*models.ParsedStatement, modules.StatementImporter, error) {
	if l.importer == nil {
		return nil, nil, modules.ErrUnsupportedStatement
	}
	parsed, err := l.importer.Parse(ctx, bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	return parsed, l.importer, nil
}

func setupTest(t *testing.T) (*Service, *sql.DB, int64) {
	t.Helper()

	database, err := db.OpenDB(":memory:", "../../../../migrations")
	if err != nil {
		t.Fatalf("abrir db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	result, err := database.Exec(`INSERT INTO banks (name, short_code, country) VALUES ('Banco Demo', 'BDEMO', 'NI')`)
	if err != nil {
		t.Fatalf("crear banco: %v", err)
	}
	bankID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de banco: %v", err)
	}

	service := NewService(NewStorage(database), nil)
	return service, database, bankID
}

func currencyIDs(t *testing.T, database *sql.DB) (int64, int64) {
	t.Helper()
	var nio, usd int64
	if err := database.QueryRow(`SELECT id FROM currencies WHERE code = 'NIO'`).Scan(&nio); err != nil {
		t.Fatalf("moneda NIO: %v", err)
	}
	if err := database.QueryRow(`SELECT id FROM currencies WHERE code = 'USD'`).Scan(&usd); err != nil {
		t.Fatalf("moneda USD: %v", err)
	}
	return nio, usd
}

func validCard(bankID, nio, usd int64) *CreditCard {
	secondary := usd
	return &CreditCard{
		BankID:                 bankID,
		Alias:                  "Tarjeta Demo",
		Network:                NetworkVisa,
		Last4:                  "0000",
		Holder:                 "Titular Demo",
		CurrencyID:             nio,
		SecondaryCurrencyID:    &secondary,
		CreditLimit:            50000,
		CreditLimitUSD:         1400,
		StatementDay:           15,
		PaymentDueDay:          14,
		MinPaymentPct:          4,
		InterestCurrentRate:    24,
		InterestLateRate:       36,
		InterestCurrentRateUSD: 18,
		InterestLateRateUSD:    30,
		Color:                  "blue",
		Icon:                   "credit",
		Active:                 true,
	}
}

func TestCardCRUDAndValidation(t *testing.T) {
	service, database, bankID := setupTest(t)
	nio, usd := currencyIDs(t, database)
	ctx := context.Background()

	card, err := service.CreateCard(ctx, validCard(bankID, nio, usd))
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if card.ID == 0 || card.BankName != "Banco Demo" || card.CurrencyCode != "NIO" {
		t.Fatalf("tarjeta inesperada: %+v", card)
	}
	if card.AvailableLocal != 50000 || card.AvailableUSD != 1400 {
		t.Fatalf("disponible inicial inesperado: %+v", card)
	}

	if _, err := service.CreateCard(ctx, validCard(bankID, nio, usd)); !errors.Is(err, ErrDuplicateCard) {
		t.Fatalf("duplicado = %v, quiero ErrDuplicateCard", err)
	}

	invalid := validCard(bankID, nio, usd)
	invalid.Alias = "Otra"
	invalid.Last4 = "12"
	if _, err := service.CreateCard(ctx, invalid); err == nil {
		t.Fatal("esperaba error por last4 inválido")
	}

	cards, err := service.ListCards(ctx, bankID)
	if err != nil || len(cards) != 1 {
		t.Fatalf("ListCards = %d, %v", len(cards), err)
	}

	card.Alias = "Tarjeta Editada"
	updated, err := service.UpdateCard(ctx, card)
	if err != nil {
		t.Fatalf("UpdateCard: %v", err)
	}
	if updated.Alias != "Tarjeta Editada" {
		t.Fatalf("alias no actualizado: %+v", updated)
	}

	if err := service.DeleteCard(ctx, card.ID); err != nil {
		t.Fatalf("DeleteCard: %v", err)
	}
	if _, err := service.GetCard(ctx, card.ID); !errors.Is(err, ErrCardNotFound) {
		t.Fatalf("GetCard tras borrar = %v", err)
	}
}

func TestStatementAndTransactions(t *testing.T) {
	service, database, bankID := setupTest(t)
	nio, usd := currencyIDs(t, database)
	ctx := context.Background()

	card, err := service.CreateCard(ctx, validCard(bankID, nio, usd))
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	statement, err := service.CreateStatement(ctx, &Statement{
		CardID:             card.ID,
		PeriodEnd:          "2026-03-15",
		MinPaymentDueDate:  "2026-04-14",
		FullPaymentDueDate: "2026-04-07",
		PreviousBalance:    1000,
		PreviousBalanceUSD: 100,
		MinPayment:         200,
		MinPaymentUSD:      20,
		NewBalance:         1124.56,
		NewBalanceUSD:      51.11,
	})
	if err != nil {
		t.Fatalf("CreateStatement: %v", err)
	}
	if statement.Source != SourceManual {
		t.Fatalf("origen = %q", statement.Source)
	}
	if _, err := service.CreateStatement(ctx, &Statement{
		CardID:             card.ID,
		PeriodEnd:          "2026-03-15",
		MinPaymentDueDate:  "2026-04-14",
		FullPaymentDueDate: "2026-04-07",
	}); !errors.Is(err, ErrStatementExists) {
		t.Fatalf("duplicado = %v, quiero ErrStatementExists", err)
	}

	if _, err := service.CreateTransaction(ctx, statement.ID, &Transaction{
		PostedAt:         "2026-03-02",
		Concept:          "Comercio Demo",
		Type:             TypePurchase,
		AmountLocal:      123.45,
		CurrencyOriginal: "NIO",
	}); err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if _, err := service.CreateTransaction(ctx, statement.ID, &Transaction{
		PostedAt:    "2026-03-05",
		Concept:     "Pago Demo",
		Type:        TypePayment,
		AmountLocal: 50,
	}); err == nil {
		t.Fatal("esperaba error de signo en pago positivo")
	}

	detail, err := service.GetStatement(ctx, statement.ID)
	if err != nil {
		t.Fatalf("GetStatement: %v", err)
	}
	if detail.Reconciliation == nil || detail.Reconciliation.DifferenceBalance != -1.11 {
		t.Fatalf("conciliación inesperada: %+v", detail.Reconciliation)
	}
	if len(detail.Summary) != 1 || detail.Summary[0].Type != TypePurchase {
		t.Fatalf("resumen inesperado: %+v", detail.Summary)
	}

	transactions, err := service.ListTransactions(ctx, statement.ID, TypePurchase, "demo")
	if err != nil || len(transactions) != 1 {
		t.Fatalf("ListTransactions = %d, %v", len(transactions), err)
	}
}

func TestImportFlow(t *testing.T) {
	service, database, bankID := setupTest(t)
	nio, usd := currencyIDs(t, database)
	ctx := context.Background()

	card, err := service.CreateCard(ctx, validCard(bankID, nio, usd))
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	parsed := &models.ParsedStatement{
		ModuleID:           "bank.test.ni.creditcard.statement.import",
		ModuleVersion:      "1.0.0",
		Last4:              "0000",
		PeriodEnd:          "2026-03-15",
		MinPaymentDue:      "2026-04-14",
		FullPaymentDue:     "2026-04-07",
		PreviousBalance:    1000,
		PreviousBalanceUSD: 100,
		MinPayment:         200,
		MinPaymentUSD:      20,
		TotalPayment:       1134.56,
		TotalPaymentUSD:    51.11,
		NewBalance:         1134.56,
		NewBalanceUSD:      51.11,
		InterestCurrent:    11.11,
		InterestCurrentUSD: 1.11,
		Transactions: []models.ParsedTransaction{
			{PostedAt: "2026-03-02", Concept: "Comercio Demo", Type: TypePurchase, AmountLocal: 123.45, CurrencyOriginal: "NIO"},
			{PostedAt: "2026-03-05", Concept: "SU PAGO RECIBIDO GRACIAS", Type: TypePayment, AmountUSD: -50, CurrencyOriginal: "USD"},
			{PostedAt: "2026-03-15", Concept: "Intereses corrientes del mes", Type: TypeInterestCurrent, AmountLocal: 11.11, AmountUSD: 1.11, CurrencyOriginal: "USD"},
		},
	}
	service.importers = fakeLookup{importer: &fakeImporter{parsed: parsed}}

	preview, hash, err := service.PreviewImport(ctx, card.ID, "estado.pdf", "application/pdf", nil)
	if err != nil {
		t.Fatalf("PreviewImport: %v", err)
	}
	if hash == "" || preview.Last4 != "0000" {
		t.Fatalf("preview inesperado: %+v hash=%q", preview, hash)
	}

	statement, err := service.ConfirmImport(ctx, card.ID, preview, hash)
	if err != nil {
		t.Fatalf("ConfirmImport: %v", err)
	}
	if statement.Source != SourceImport || statement.SourceModuleID != parsed.ModuleID {
		t.Fatalf("estado importado inesperado: %+v", statement)
	}
	if statement.TransactionCount != 3 {
		t.Fatalf("movimientos importados = %d", statement.TransactionCount)
	}

	detail, err := service.GetStatement(ctx, statement.ID)
	if err != nil {
		t.Fatalf("GetStatement: %v", err)
	}
	if !detail.Reconciliation.Balanced {
		t.Fatalf("conciliación no balanceada: %+v", detail.Reconciliation)
	}

	after, err := service.GetCard(ctx, card.ID)
	if err != nil {
		t.Fatalf("GetCard: %v", err)
	}
	if after.BalanceLocal != 1134.56 || after.BalanceUSD != 51.11 {
		t.Fatalf("saldo derivado inesperado: %+v", after)
	}
	if after.AvailableLocal != 48865.44 {
		t.Fatalf("disponible inesperado: %v", after.AvailableLocal)
	}

	if _, err := service.ConfirmImport(ctx, card.ID, preview, hash); !errors.Is(err, ErrStatementExists) {
		t.Fatalf("reimportar = %v, quiero ErrStatementExists", err)
	}

	// El hash cambia si el payload cambia.
	preview.Transactions[0].AmountLocal = 999
	if _, err := service.ConfirmImport(ctx, card.ID, preview, hash); !errors.Is(err, ErrPreviewMismatch) {
		t.Fatalf("hash alterado = %v, quiero ErrPreviewMismatch", err)
	}
}

func TestImportCardMismatch(t *testing.T) {
	service, database, bankID := setupTest(t)
	nio, usd := currencyIDs(t, database)
	ctx := context.Background()

	card, err := service.CreateCard(ctx, validCard(bankID, nio, usd))
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	parsed := &models.ParsedStatement{
		Last4:          "9999",
		PeriodEnd:      "2026-03-15",
		MinPaymentDue:  "2026-04-14",
		FullPaymentDue: "2026-04-07",
		Transactions: []models.ParsedTransaction{
			{PostedAt: "2026-03-02", Concept: "Comercio Demo", Type: TypePurchase, AmountLocal: 10, CurrencyOriginal: "NIO"},
		},
	}
	service.importers = fakeLookup{importer: &fakeImporter{parsed: parsed}}

	if _, _, err := service.PreviewImport(ctx, card.ID, "estado.pdf", "application/pdf", nil); !errors.Is(err, ErrCardMismatch) {
		t.Fatalf("preview = %v, quiero ErrCardMismatch", err)
	}
}

// ---------------------------------------------------------------- SPEC-138

func validationCode(t *testing.T, err error) string {
	t.Helper()
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("esperaba ValidationError, obtuve %v", err)
	}
	return validation.Code
}

func TestDeriveLimitFromUSD(t *testing.T) {
	service, database, bankID := setupTest(t)
	nio, usd := currencyIDs(t, database)
	ctx := context.Background()

	card := validCard(bankID, nio, usd)
	card.LimitCurrency = LimitCurrencyUSD
	card.CreditLimit = 0
	card.CreditLimitUSD = 1000
	card.ExchangeRate = 36.5

	created, err := service.CreateCard(ctx, card)
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if created.CreditLimit != 36500 {
		t.Fatalf("límite local derivado = %v, esperaba 36500", created.CreditLimit)
	}
	if created.LateTotalRate != 60 || created.LateTotalRateUSD != 48 {
		t.Fatalf("moratoria total = %v/%v, esperaba 60/48", created.LateTotalRate, created.LateTotalRateUSD)
	}
}

func TestDeriveLimitFromLocal(t *testing.T) {
	service, database, bankID := setupTest(t)
	nio, usd := currencyIDs(t, database)
	ctx := context.Background()

	card := validCard(bankID, nio, usd)
	card.LimitCurrency = LimitCurrencyLocal
	card.CreditLimit = 36500
	card.CreditLimitUSD = 0
	card.ExchangeRate = 36.5

	created, err := service.CreateCard(ctx, card)
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if created.CreditLimitUSD != 1000 {
		t.Fatalf("límite USD derivado = %v, esperaba 1000", created.CreditLimitUSD)
	}
}

func TestDeriveLimitKeepsManualValues(t *testing.T) {
	service, database, bankID := setupTest(t)
	nio, usd := currencyIDs(t, database)
	ctx := context.Background()

	card := validCard(bankID, nio, usd)
	card.LimitCurrency = LimitCurrencyUSD
	card.CreditLimit = 30000
	card.CreditLimitUSD = 1000
	card.ExchangeRate = 36.5

	created, err := service.CreateCard(ctx, card)
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if created.CreditLimit != 30000 {
		t.Fatalf("el límite manual se sobrescribió: %v", created.CreditLimit)
	}
}

func TestLimitWithoutExchangeRateStaysUndefined(t *testing.T) {
	service, database, bankID := setupTest(t)
	nio, usd := currencyIDs(t, database)
	ctx := context.Background()

	card := validCard(bankID, nio, usd)
	card.LimitCurrency = LimitCurrencyUSD
	card.CreditLimit = 0
	card.CreditLimitUSD = 1000
	card.ExchangeRate = 0

	created, err := service.CreateCard(ctx, card)
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	if created.CreditLimit != 0 {
		t.Fatalf("sin tipo de cambio el límite local debe quedar en 0, obtuve %v", created.CreditLimit)
	}
}

func TestBonusDayValidation(t *testing.T) {
	service, database, bankID := setupTest(t)
	nio, usd := currencyIDs(t, database)
	ctx := context.Background()

	equal := validCard(bankID, nio, usd)
	equal.BonusDueDay = equal.PaymentDueDay
	if _, err := service.CreateCard(ctx, equal); validationCode(t, err) != "invalid_bonus_day" {
		t.Fatalf("bonus = payment debe fallar con invalid_bonus_day")
	}

	outOfRange := validCard(bankID, nio, usd)
	outOfRange.BonusDueDay = 32
	if _, err := service.CreateCard(ctx, outOfRange); validationCode(t, err) != "invalid_bonus_day" {
		t.Fatalf("bonus 32 debe fallar con invalid_bonus_day")
	}

	valid := validCard(bankID, nio, usd)
	valid.BonusDueDay = 7
	created, err := service.CreateCard(ctx, valid)
	if err != nil {
		t.Fatalf("CreateCard con bonus válido: %v", err)
	}
	if created.BonusDueDay != 7 {
		t.Fatalf("bonus_due_day = %d, esperaba 7", created.BonusDueDay)
	}
}

func TestLimitCurrencyAndExchangeRateValidation(t *testing.T) {
	service, database, bankID := setupTest(t)
	nio, usd := currencyIDs(t, database)
	ctx := context.Background()

	badCurrency := validCard(bankID, nio, usd)
	badCurrency.LimitCurrency = "EUR"
	if _, err := service.CreateCard(ctx, badCurrency); validationCode(t, err) != "invalid_limit_currency" {
		t.Fatalf("moneda EUR debe fallar con invalid_limit_currency")
	}

	badRate := validCard(bankID, nio, usd)
	badRate.ExchangeRate = -1
	if _, err := service.CreateCard(ctx, badRate); validationCode(t, err) != "invalid_exchange_rate" {
		t.Fatalf("tipo de cambio negativo debe fallar con invalid_exchange_rate")
	}

	badCategory := validCard(bankID, nio, usd)
	badCategory.Category = "inventada"
	if _, err := service.CreateCard(ctx, badCategory); validationCode(t, err) != "validation_error" {
		t.Fatalf("categoría inválida debe fallar")
	}

	withCategory := validCard(bankID, nio, usd)
	withCategory.Category = "gold"
	created, err := service.CreateCard(ctx, withCategory)
	if err != nil {
		t.Fatalf("CreateCard con categoría: %v", err)
	}
	if created.Category != "gold" {
		t.Fatalf("categoría = %q, esperaba gold", created.Category)
	}
}

// ---------------------------------------------------------------- SPEC-139

func TestReconciliationTwoBases(t *testing.T) {
	service, database, bankID := setupTest(t)
	nio, usd := currencyIDs(t, database)
	ctx := context.Background()

	card, err := service.CreateCard(ctx, validCard(bankID, nio, usd))
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}

	statement, err := service.CreateStatement(ctx, &Statement{
		CardID:             card.ID,
		PeriodEnd:          "2026-11-17",
		MinPaymentDueDate:  "2026-12-14",
		FullPaymentDueDate: "2026-11-17",
		PreviousBalance:    1000,
		TotalPayment:       1050,
		NewBalance:         1080,
		InterestBonifiable: 30,
		InterestCurrent:    30,
		MinPayment:         100,
		MoraAmount:         200,
		ArrearsCount:       1,
		OverLimitAmount:    50,
	})
	if err != nil {
		t.Fatalf("CreateStatement: %v", err)
	}
	if statement.InterestBonifiable != 30 || statement.MoraAmount != 200 || statement.ArrearsCount != 1 {
		t.Fatalf("agregados no persistidos: %+v", statement)
	}

	if _, err := service.CreateTransaction(ctx, statement.ID, &Transaction{
		PostedAt: "2026-11-02", Concept: "Comercio Demo", Type: TypePurchase, AmountLocal: 100,
	}); err != nil {
		t.Fatalf("compra: %v", err)
	}
	if _, err := service.CreateTransaction(ctx, statement.ID, &Transaction{
		PostedAt: "2026-11-05", Concept: "Pago Demo", Type: TypePayment, AmountLocal: -50,
	}); err != nil {
		t.Fatalf("pago: %v", err)
	}

	detail, err := service.GetStatement(ctx, statement.ID)
	if err != nil {
		t.Fatalf("GetStatement: %v", err)
	}
	recon := detail.Reconciliation
	if recon == nil {
		t.Fatal("sin conciliación")
	}
	if recon.DifferenceCash != 0 || recon.DifferenceBalance != 0 || !recon.Balanced {
		t.Fatalf("conciliación inesperada: %+v", recon)
	}
	if !recon.CashApplicable || !recon.BalanceApplicable {
		t.Fatalf("ambas bases deberían aplicar: %+v", recon)
	}
}

func TestReconciliationDetectsMissingBonifiable(t *testing.T) {
	service, database, bankID := setupTest(t)
	nio, usd := currencyIDs(t, database)
	ctx := context.Background()

	card, err := service.CreateCard(ctx, validCard(bankID, nio, usd))
	if err != nil {
		t.Fatalf("CreateCard: %v", err)
	}
	// El saldo al corte incluye 30 de bonificable, pero el estado no lo declara.
	statement, err := service.CreateStatement(ctx, &Statement{
		CardID:             card.ID,
		PeriodEnd:          "2026-11-17",
		MinPaymentDueDate:  "2026-12-14",
		FullPaymentDueDate: "2026-11-17",
		PreviousBalance:    1000,
		TotalPayment:       1050,
		NewBalance:         1080,
	})
	if err != nil {
		t.Fatalf("CreateStatement: %v", err)
	}
	if _, err := service.CreateTransaction(ctx, statement.ID, &Transaction{
		PostedAt: "2026-11-02", Concept: "Comercio Demo", Type: TypePurchase, AmountLocal: 100,
	}); err != nil {
		t.Fatalf("compra: %v", err)
	}
	if _, err := service.CreateTransaction(ctx, statement.ID, &Transaction{
		PostedAt: "2026-11-05", Concept: "Pago Demo", Type: TypePayment, AmountLocal: -50,
	}); err != nil {
		t.Fatalf("pago: %v", err)
	}

	detail, err := service.GetStatement(ctx, statement.ID)
	if err != nil {
		t.Fatalf("GetStatement: %v", err)
	}
	recon := detail.Reconciliation
	if recon.DifferenceCash != 0 {
		t.Fatalf("la base cash debería cerrar: %+v", recon)
	}
	if recon.DifferenceBalance != -30 || recon.Balanced {
		t.Fatalf("la base balance debería acusar 30: %+v", recon)
	}
}
