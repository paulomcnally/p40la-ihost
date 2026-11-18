package creditcard

import (
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
	if detail.Reconciliation == nil || detail.Reconciliation.DifferenceLocal != -1.11 {
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
