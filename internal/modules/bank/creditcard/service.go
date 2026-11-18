package creditcard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/modules"
)

// Límites de validación.
const (
	maxAlias    = 100
	maxHolder   = 150
	maxNotes    = 500
	maxConcept  = 200
	maxIcon     = 50
	maxQuery    = 100
	maxFileSize = 5 * 1024 * 1024
)

var (
	last4Regex = regexp.MustCompile(`^[0-9]{4}$`)
	dateRegex  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

var (
	// ErrUnsupportedStatement indica que ningún importador reconoce el archivo.
	// SPEC-139: un único sentinela compartido con el registry.
	ErrUnsupportedStatement = modules.ErrUnsupportedStatement
	// ErrCardMismatch indica que el PDF pertenece a otra tarjeta.
	ErrCardMismatch = errors.New("el estado de cuenta no corresponde a la tarjeta")
	// ErrPreviewMismatch indica que el preview y el confirm no coinciden.
	ErrPreviewMismatch = errors.New("los datos de la importación cambiaron desde el preview")
	// ErrInvalidFile indica un archivo inválido.
	ErrInvalidFile = errors.New("archivo inválido")
)

// ValidationError describe un error de validación con código estable.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// ImporterLookup encuentra el importador adecuado para un archivo.
type ImporterLookup interface {
	FindImporter(filename, mimeType string) (modules.StatementImporter, bool)
	ParseStatements(ctx context.Context, filename, mimeType string, data []byte) (*models.ParsedStatement, modules.StatementImporter, error)
}

// Service contiene la lógica de negocio del módulo (ARCH-MODULES).
type Service struct {
	storage   *Storage
	importers ImporterLookup
}

// NewService crea el servicio del módulo.
func NewService(storage *Storage, importers ImporterLookup) *Service {
	return &Service{storage: storage, importers: importers}
}

// ---------------------------------------------------------------- tarjetas

// ListCards devuelve las tarjetas de un banco con sus derivados.
func (s *Service) ListCards(ctx context.Context, bankID int64) ([]CreditCard, error) {
	exists, err := s.storage.BankExists(ctx, bankID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrBankNotFound
	}
	cards, err := s.storage.ListCards(ctx, bankID)
	if err != nil {
		return nil, err
	}
	for i := range cards {
		if err := s.withDerived(ctx, &cards[i]); err != nil {
			return nil, err
		}
	}
	return cards, nil
}

// GetCard devuelve una tarjeta con sus derivados.
func (s *Service) GetCard(ctx context.Context, id int64) (*CreditCard, error) {
	card, err := s.storage.GetCard(ctx, id)
	if err != nil {
		return nil, err
	}
	if card == nil {
		return nil, ErrCardNotFound
	}
	if err := s.withDerived(ctx, card); err != nil {
		return nil, err
	}
	return card, nil
}

// CreateCard valida e inserta una tarjeta.
func (s *Service) CreateCard(ctx context.Context, card *CreditCard) (*CreditCard, error) {
	if err := s.prepareCard(ctx, card, true); err != nil {
		return nil, err
	}
	created, err := s.storage.CreateCard(ctx, card)
	if err != nil {
		return nil, err
	}
	if err := s.withDerived(ctx, created); err != nil {
		return nil, err
	}
	return created, nil
}

// UpdateCard valida y actualiza una tarjeta. El banco no cambia.
func (s *Service) UpdateCard(ctx context.Context, card *CreditCard) (*CreditCard, error) {
	existing, err := s.storage.GetCard(ctx, card.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrCardNotFound
	}
	card.BankID = existing.BankID
	if err := s.prepareCard(ctx, card, false); err != nil {
		return nil, err
	}
	updated, err := s.storage.UpdateCard(ctx, card)
	if err != nil {
		return nil, err
	}
	if err := s.withDerived(ctx, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

// DeleteCard elimina una tarjeta y sus dependencias.
func (s *Service) DeleteCard(ctx context.Context, id int64) error {
	return s.storage.DeleteCard(ctx, id)
}

// ------------------------------------------------------- estados de cuenta

// ListStatements devuelve los estados de una tarjeta.
func (s *Service) ListStatements(ctx context.Context, cardID int64) ([]Statement, error) {
	card, err := s.storage.GetCard(ctx, cardID)
	if err != nil {
		return nil, err
	}
	if card == nil {
		return nil, ErrCardNotFound
	}
	return s.storage.ListStatements(ctx, cardID)
}

// GetStatement devuelve el estado con resumen y conciliación.
func (s *Service) GetStatement(ctx context.Context, id int64) (*Statement, error) {
	statement, err := s.storage.GetStatement(ctx, id)
	if err != nil {
		return nil, err
	}
	if statement == nil {
		return nil, ErrStatementNotFound
	}
	summary, err := s.storage.StatementSummary(ctx, id)
	if err != nil {
		return nil, err
	}
	statement.Summary = summary

	var totalLocal, totalUSD float64
	for _, item := range summary {
		totalLocal += item.AmountLocal
		totalUSD += item.AmountUSD
	}
	// SPEC-139: dos bases de conciliación. `cash` valida la suma de movimientos;
	// `balance` agrega el interés bonificable que el banco imprime aparte.
	expectedCashLocal := round2(statement.PreviousBalance + totalLocal)
	expectedCashUSD := round2(statement.PreviousBalanceUSD + totalUSD)
	expectedBalanceLocal := round2(expectedCashLocal + statement.InterestBonifiable)
	expectedBalanceUSD := round2(expectedCashUSD + statement.InterestBonifiableUSD)

	diffCashLocal := round2(expectedCashLocal - statement.TotalPayment)
	diffCashUSD := round2(expectedCashUSD - statement.TotalPaymentUSD)
	diffBalanceLocal := round2(expectedBalanceLocal - statement.NewBalance)
	diffBalanceUSD := round2(expectedBalanceUSD - statement.NewBalanceUSD)

	cashApplicable := statement.TotalPayment != 0 || statement.TotalPaymentUSD != 0
	balanceApplicable := statement.NewBalance != 0 || statement.NewBalanceUSD != 0
	cashOK := !cashApplicable ||
		(math.Abs(diffCashLocal) <= 0.005 && math.Abs(diffCashUSD) <= 0.005)
	balanceOK := !balanceApplicable ||
		(math.Abs(diffBalanceLocal) <= 0.005 && math.Abs(diffBalanceUSD) <= 0.005)

	statement.Reconciliation = &Reconciliation{
		ExpectedCash:         expectedCashLocal,
		PrintedCash:          statement.TotalPayment,
		DifferenceCash:       diffCashLocal,
		ExpectedBalance:      expectedBalanceLocal,
		PrintedBalance:       statement.NewBalance,
		DifferenceBalance:    diffBalanceLocal,
		ExpectedCashUSD:      expectedCashUSD,
		PrintedCashUSD:       statement.TotalPaymentUSD,
		DifferenceCashUSD:    diffCashUSD,
		ExpectedBalanceUSD:   expectedBalanceUSD,
		PrintedBalanceUSD:    statement.NewBalanceUSD,
		DifferenceBalanceUSD: diffBalanceUSD,
		CashApplicable:       cashApplicable,
		BalanceApplicable:    balanceApplicable,
		Balanced:             cashOK && balanceOK,
	}
	return statement, nil
}

// CreateStatement valida e inserta un estado manual.
func (s *Service) CreateStatement(ctx context.Context, statement *Statement) (*Statement, error) {
	if err := s.prepareStatement(ctx, statement, true); err != nil {
		return nil, err
	}
	return s.storage.CreateStatement(ctx, statement)
}

// UpdateStatement valida y actualiza un estado.
func (s *Service) UpdateStatement(ctx context.Context, statement *Statement) (*Statement, error) {
	existing, err := s.storage.GetStatement(ctx, statement.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrStatementNotFound
	}
	statement.CardID = existing.CardID
	statement.Source = existing.Source
	statement.SourceModuleID = existing.SourceModuleID
	statement.ImportHash = existing.ImportHash
	if err := s.prepareStatement(ctx, statement, false); err != nil {
		return nil, err
	}
	return s.storage.UpdateStatement(ctx, statement)
}

// DeleteStatement elimina un estado y sus movimientos.
func (s *Service) DeleteStatement(ctx context.Context, id int64) error {
	return s.storage.DeleteStatement(ctx, id)
}

// ------------------------------------------------------------- movimientos

// ListTransactions devuelve los movimientos de un estado.
func (s *Service) ListTransactions(ctx context.Context, statementID int64, txType, query string) ([]Transaction, error) {
	statement, err := s.storage.GetStatement(ctx, statementID)
	if err != nil {
		return nil, err
	}
	if statement == nil {
		return nil, ErrStatementNotFound
	}
	if txType != "" && !isTransactionType(txType) {
		return nil, &ValidationError{Code: "validation_error", Message: "tipo de movimiento inválido"}
	}
	if len(query) > maxQuery {
		query = query[:maxQuery]
	}
	return s.storage.ListTransactions(ctx, statementID, txType, strings.TrimSpace(query))
}

// CreateTransaction valida e inserta un movimiento.
func (s *Service) CreateTransaction(ctx context.Context, statementID int64, transaction *Transaction) (*Transaction, error) {
	statement, err := s.storage.GetStatement(ctx, statementID)
	if err != nil {
		return nil, err
	}
	if statement == nil {
		return nil, ErrStatementNotFound
	}
	transaction.CardID = statement.CardID
	transaction.StatementID = &statementID
	if err := s.prepareTransaction(ctx, transaction); err != nil {
		return nil, err
	}
	return s.storage.CreateTransaction(ctx, transaction)
}

// UpdateTransaction valida y actualiza un movimiento.
func (s *Service) UpdateTransaction(ctx context.Context, transaction *Transaction) (*Transaction, error) {
	existing, err := s.storage.GetTransaction(ctx, transaction.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrTransactionNotFound
	}
	transaction.CardID = existing.CardID
	transaction.StatementID = existing.StatementID
	transaction.ExternalRef = existing.ExternalRef
	if err := s.prepareTransaction(ctx, transaction); err != nil {
		return nil, err
	}
	return s.storage.UpdateTransaction(ctx, transaction)
}

// DeleteTransaction elimina un movimiento.
func (s *Service) DeleteTransaction(ctx context.Context, id int64) error {
	return s.storage.DeleteTransaction(ctx, id)
}

// --------------------------------------------------------------- importar

// PreviewImport parsea un archivo sin persistirlo y devuelve el hash del payload.
func (s *Service) PreviewImport(ctx context.Context, cardID int64, filename, mimeType string, data []byte) (*models.ParsedStatement, string, error) {
	card, err := s.storage.GetCard(ctx, cardID)
	if err != nil {
		return nil, "", err
	}
	if card == nil {
		return nil, "", ErrCardNotFound
	}
	if s.importers == nil {
		return nil, "", ErrUnsupportedStatement
	}
	parsed, _, err := s.importers.ParseStatements(ctx, filename, mimeType, data)
	if err != nil {
		return nil, "", err
	}
	if err := normalizeParsed(parsed); err != nil {
		return nil, "", err
	}
	if parsed.Last4 != "" && parsed.Last4 != card.Last4 {
		return nil, "", ErrCardMismatch
	}
	exists, err := s.storage.StatementExists(ctx, cardID, parsed.PeriodEnd)
	if err != nil {
		return nil, "", err
	}
	if exists {
		return nil, "", ErrStatementExists
	}
	hash, err := hashParsed(parsed)
	if err != nil {
		return nil, "", err
	}
	return parsed, hash, nil
}

// ConfirmImport persiste el estado y sus movimientos en una transacción SQL.
func (s *Service) ConfirmImport(ctx context.Context, cardID int64, parsed *models.ParsedStatement, previewHash string) (*Statement, error) {
	card, err := s.storage.GetCard(ctx, cardID)
	if err != nil {
		return nil, err
	}
	if card == nil {
		return nil, ErrCardNotFound
	}
	if err := normalizeParsed(parsed); err != nil {
		return nil, err
	}
	hash, err := hashParsed(parsed)
	if err != nil {
		return nil, err
	}
	if hash != previewHash {
		return nil, ErrPreviewMismatch
	}
	if parsed.Last4 != "" && parsed.Last4 != card.Last4 {
		return nil, ErrCardMismatch
	}
	exists, err := s.storage.StatementExists(ctx, cardID, parsed.PeriodEnd)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrStatementExists
	}

	statement := &Statement{
		CardID:             cardID,
		PeriodEnd:          parsed.PeriodEnd,
		MinPaymentDueDate:  parsed.MinPaymentDue,
		FullPaymentDueDate: parsed.FullPaymentDue,
		PreviousBalance:    parsed.PreviousBalance,
		PreviousBalanceUSD: parsed.PreviousBalanceUSD,
		InterestCurrent:    parsed.InterestCurrent,
		InterestCurrentUSD: parsed.InterestCurrentUSD,

		InterestBonifiable:    parsed.InterestBonifiable,
		InterestBonifiableUSD: parsed.InterestBonifiableUSD,
		MoraAmount:            parsed.MoraAmount,
		MoraAmountUSD:         parsed.MoraAmountUSD,
		ArrearsCount:          parsed.ArrearsCount,
		OverLimitAmount:       parsed.OverLimitAmount,
		OverLimitAmountUSD:    parsed.OverLimitAmountUSD,
		PointsPrevious:        parsed.PointsPrevious,
		PointsEarned:          parsed.PointsEarned,
		PointsUsed:            parsed.PointsUsed,
		PointsAvailable:       parsed.PointsAvailable,

		MinPayment:      parsed.MinPayment,
		MinPaymentUSD:   parsed.MinPaymentUSD,
		TotalPayment:    parsed.TotalPayment,
		TotalPaymentUSD: parsed.TotalPaymentUSD,
		NewBalance:      parsed.NewBalance,
		NewBalanceUSD:   parsed.NewBalanceUSD,
		Status:          StatementUnpaid,
		Source:          SourceImport,
		SourceModuleID:  parsed.ModuleID,
		ImportHash:      importHash(cardID, parsed.PeriodEnd, hash),
	}

	transactions := make([]Transaction, 0, len(parsed.Transactions))
	for i, item := range parsed.Transactions {
		ref := item.ExternalRef
		if ref == "" {
			ref = importHash(cardID, parsed.PeriodEnd, fmt.Sprintf("%s|%d", hash, i))[:32]
		}
		transactions = append(transactions, Transaction{
			CardID:           cardID,
			PostedAt:         item.PostedAt,
			Concept:          item.Concept,
			Type:             item.Type,
			AmountLocal:      round2(item.AmountLocal),
			AmountUSD:        round2(item.AmountUSD),
			CurrencyOriginal: item.CurrencyOriginal,
			Status:           TransactionPosted,
			ExternalRef:      ref,
		})
	}
	return s.storage.ImportStatement(ctx, statement, transactions)
}

// --------------------------------------------------------------- helpers

func (s *Service) withDerived(ctx context.Context, card *CreditCard) error {
	id, periodEnd, local, usd, found, err := s.storage.LatestStatementBalance(ctx, card.ID)
	if err != nil {
		return err
	}
	if found {
		sumLocal, sumUSD, err := s.storage.SumTransactionsAfter(ctx, card.ID, periodEnd)
		if err != nil {
			return err
		}
		card.BalanceLocal = round2(local + sumLocal)
		card.BalanceUSD = round2(usd + sumUSD)
		card.LastStatementID = &id
	}
	card.AvailableLocal = math.Max(0, round2(card.CreditLimit-card.BalanceLocal))
	card.AvailableUSD = math.Max(0, round2(card.CreditLimitUSD-card.BalanceUSD))
	return nil
}

func (s *Service) prepareCard(ctx context.Context, card *CreditCard, checkBank bool) error {
	card.Alias = strings.TrimSpace(card.Alias)
	card.Holder = strings.TrimSpace(card.Holder)
	card.Notes = strings.TrimSpace(card.Notes)
	card.Icon = strings.TrimSpace(card.Icon)
	card.Network = strings.ToLower(strings.TrimSpace(card.Network))
	card.Color = strings.ToLower(strings.TrimSpace(card.Color))

	if checkBank {
		if card.BankID <= 0 {
			return &ValidationError{Code: "validation_error", Message: "El banco es obligatorio"}
		}
		exists, err := s.storage.BankExists(ctx, card.BankID)
		if err != nil {
			return err
		}
		if !exists {
			return ErrBankNotFound
		}
	}
	if card.Alias == "" {
		return &ValidationError{Code: "validation_error", Message: "El alias es obligatorio"}
	}
	if len(card.Alias) > maxAlias {
		return &ValidationError{Code: "validation_error", Message: "El alias es demasiado largo"}
	}
	if !isNetwork(card.Network) {
		return &ValidationError{Code: "validation_error", Message: "Marca de tarjeta inválida"}
	}
	if !last4Regex.MatchString(card.Last4) {
		return &ValidationError{Code: "validation_error", Message: "Los últimos 4 dígitos deben ser exactamente 4 números"}
	}
	if len(card.Holder) > maxHolder {
		return &ValidationError{Code: "validation_error", Message: "El titular es demasiado largo"}
	}
	if card.CurrencyID <= 0 {
		return &ValidationError{Code: "validation_error", Message: "La moneda es obligatoria"}
	}
	exists, err := s.storage.CurrencyExists(ctx, card.CurrencyID)
	if err != nil {
		return err
	}
	if !exists {
		return &ValidationError{Code: "validation_error", Message: "La moneda no existe"}
	}
	if card.SecondaryCurrencyID != nil {
		if *card.SecondaryCurrencyID <= 0 {
			return &ValidationError{Code: "validation_error", Message: "La moneda secundaria es inválida"}
		}
		if *card.SecondaryCurrencyID == card.CurrencyID {
			return &ValidationError{Code: "validation_error", Message: "La moneda secundaria debe ser distinta de la principal"}
		}
		exists, err := s.storage.CurrencyExists(ctx, *card.SecondaryCurrencyID)
		if err != nil {
			return err
		}
		if !exists {
			return &ValidationError{Code: "validation_error", Message: "La moneda secundaria no existe"}
		}
	}
	if card.StatementDay < 1 || card.StatementDay > 31 {
		return &ValidationError{Code: "validation_error", Message: "El día de corte debe estar entre 1 y 31"}
	}
	if card.PaymentDueDay < 1 || card.PaymentDueDay > 31 {
		return &ValidationError{Code: "validation_error", Message: "El día de pago debe estar entre 1 y 31"}
	}
	if !validPercent(card.MinPaymentPct) {
		return &ValidationError{Code: "validation_error", Message: "El porcentaje de pago mínimo debe estar entre 0 y 100"}
	}
	for _, value := range []float64{
		card.InterestCurrentRate, card.InterestLateRate,
		card.InterestCurrentRateUSD, card.InterestLateRateUSD,
	} {
		if !validAmount(value) || value < 0 {
			return &ValidationError{Code: "validation_error", Message: "Las tasas deben ser números positivos"}
		}
	}
	if !validAmount(card.CreditLimit) || card.CreditLimit < 0 ||
		!validAmount(card.CreditLimitUSD) || card.CreditLimitUSD < 0 {
		return &ValidationError{Code: "validation_error", Message: "El límite de crédito es inválido"}
	}
	if !isColor(card.Color) {
		return &ValidationError{Code: "validation_error", Message: "Color inválido"}
	}
	if card.Icon == "" {
		card.Icon = "credit"
	}
	if len(card.Icon) > maxIcon {
		return &ValidationError{Code: "validation_error", Message: "Icono inválido"}
	}
	if len(card.Notes) > maxNotes {
		return &ValidationError{Code: "validation_error", Message: "Las notas son demasiado largas"}
	}

	card.CreditLimit = round2(card.CreditLimit)
	card.CreditLimitUSD = round2(card.CreditLimitUSD)
	card.MinPaymentPct = round2(card.MinPaymentPct)
	card.InterestCurrentRate = round2(card.InterestCurrentRate)
	card.InterestLateRate = round2(card.InterestLateRate)
	card.InterestCurrentRateUSD = round2(card.InterestCurrentRateUSD)
	card.InterestLateRateUSD = round2(card.InterestLateRateUSD)
	return nil
}

func (s *Service) prepareStatement(ctx context.Context, statement *Statement, checkCard bool) error {
	if checkCard {
		if statement.CardID <= 0 {
			return &ValidationError{Code: "validation_error", Message: "La tarjeta es obligatoria"}
		}
		card, err := s.storage.GetCard(ctx, statement.CardID)
		if err != nil {
			return err
		}
		if card == nil {
			return ErrCardNotFound
		}
	}
	if statement.PeriodStart != nil {
		value := strings.TrimSpace(*statement.PeriodStart)
		if value == "" {
			statement.PeriodStart = nil
		} else if !validDate(value) {
			return &ValidationError{Code: "validation_error", Message: "La fecha de inicio del periodo es inválida"}
		} else {
			statement.PeriodStart = &value
		}
	}
	statement.PeriodEnd = strings.TrimSpace(statement.PeriodEnd)
	statement.MinPaymentDueDate = strings.TrimSpace(statement.MinPaymentDueDate)
	statement.FullPaymentDueDate = strings.TrimSpace(statement.FullPaymentDueDate)
	statement.Notes = strings.TrimSpace(statement.Notes)

	if !validDate(statement.PeriodEnd) {
		return &ValidationError{Code: "validation_error", Message: "La fecha de corte es obligatoria (YYYY-MM-DD)"}
	}
	if !validDate(statement.MinPaymentDueDate) {
		return &ValidationError{Code: "validation_error", Message: "La fecha límite de pago mínimo es obligatoria"}
	}
	if !validDate(statement.FullPaymentDueDate) {
		return &ValidationError{Code: "validation_error", Message: "La fecha límite de pago de contado es obligatoria"}
	}
	if statement.PeriodStart != nil && *statement.PeriodStart > statement.PeriodEnd {
		return &ValidationError{Code: "validation_error", Message: "El periodo no puede terminar antes de empezar"}
	}
	if statement.Status == "" {
		statement.Status = StatementUnpaid
	}
	if !isStatementStatus(statement.Status) {
		return &ValidationError{Code: "validation_error", Message: "Estado del estado de cuenta inválido"}
	}
	if statement.Source == "" {
		statement.Source = SourceManual
	}
	if statement.Source != SourceManual && statement.Source != SourceImport {
		return &ValidationError{Code: "validation_error", Message: "Origen inválido"}
	}
	for name, value := range map[string]float64{
		"saldo anterior":      statement.PreviousBalance,
		"saldo anterior USD":  statement.PreviousBalanceUSD,
		"intereses":           statement.InterestCurrent,
		"intereses USD":       statement.InterestCurrentUSD,
		"pago mínimo":         statement.MinPayment,
		"pago mínimo USD":     statement.MinPaymentUSD,
		"pago de contado":     statement.TotalPayment,
		"pago de contado USD": statement.TotalPaymentUSD,
		"saldo al corte":      statement.NewBalance,
		"saldo al corte USD":  statement.NewBalanceUSD,

		"interés bonificable":     statement.InterestBonifiable,
		"interés bonificable USD": statement.InterestBonifiableUSD,
		"monto en mora":           statement.MoraAmount,
		"monto en mora USD":       statement.MoraAmountUSD,
		"monto en exceso":         statement.OverLimitAmount,
		"monto en exceso USD":     statement.OverLimitAmountUSD,
	} {
		if !validAmount(value) {
			return &ValidationError{Code: "validation_error", Message: "Monto inválido en " + name}
		}
	}
	if statement.ArrearsCount < 0 || statement.PointsPrevious < 0 || statement.PointsEarned < 0 ||
		statement.PointsUsed < 0 || statement.PointsAvailable < 0 {
		return &ValidationError{Code: "validation_error", Message: "Los contadores no pueden ser negativos"}
	}
	if len(statement.Notes) > maxNotes {
		return &ValidationError{Code: "validation_error", Message: "Las notas son demasiado largas"}
	}
	statement.PreviousBalance = round2(statement.PreviousBalance)
	statement.PreviousBalanceUSD = round2(statement.PreviousBalanceUSD)
	statement.InterestCurrent = round2(statement.InterestCurrent)
	statement.InterestCurrentUSD = round2(statement.InterestCurrentUSD)
	statement.MinPayment = round2(statement.MinPayment)
	statement.MinPaymentUSD = round2(statement.MinPaymentUSD)
	statement.TotalPayment = round2(statement.TotalPayment)
	statement.TotalPaymentUSD = round2(statement.TotalPaymentUSD)
	statement.NewBalance = round2(statement.NewBalance)
	statement.NewBalanceUSD = round2(statement.NewBalanceUSD)
	statement.InterestBonifiable = round2(statement.InterestBonifiable)
	statement.InterestBonifiableUSD = round2(statement.InterestBonifiableUSD)
	statement.MoraAmount = round2(statement.MoraAmount)
	statement.MoraAmountUSD = round2(statement.MoraAmountUSD)
	statement.OverLimitAmount = round2(statement.OverLimitAmount)
	statement.OverLimitAmountUSD = round2(statement.OverLimitAmountUSD)
	return nil
}

func (s *Service) prepareTransaction(ctx context.Context, transaction *Transaction) error {
	transaction.PostedAt = strings.TrimSpace(transaction.PostedAt)
	transaction.Concept = strings.TrimSpace(transaction.Concept)
	transaction.Note = strings.TrimSpace(transaction.Note)
	transaction.Type = strings.ToLower(strings.TrimSpace(transaction.Type))
	transaction.CurrencyOriginal = strings.ToUpper(strings.TrimSpace(transaction.CurrencyOriginal))
	transaction.Status = strings.ToLower(strings.TrimSpace(transaction.Status))

	if !validDate(transaction.PostedAt) {
		return &ValidationError{Code: "validation_error", Message: "La fecha del movimiento es inválida"}
	}
	if transaction.Concept == "" {
		return &ValidationError{Code: "validation_error", Message: "El concepto es obligatorio"}
	}
	if len(transaction.Concept) > maxConcept {
		return &ValidationError{Code: "validation_error", Message: "El concepto es demasiado largo"}
	}
	if !isTransactionType(transaction.Type) {
		return &ValidationError{Code: "validation_error", Message: "Tipo de movimiento inválido"}
	}
	if !validAmount(transaction.AmountLocal) || !validAmount(transaction.AmountUSD) {
		return &ValidationError{Code: "validation_error", Message: "Monto inválido"}
	}
	switch transaction.CurrencyOriginal {
	case "":
		if transaction.AmountUSD != 0 {
			transaction.CurrencyOriginal = "USD"
		} else {
			transaction.CurrencyOriginal = "NIO"
		}
	case "NIO", "USD":
	default:
		return &ValidationError{Code: "validation_error", Message: "Moneda original inválida"}
	}
	if transaction.Status == "" {
		transaction.Status = TransactionPosted
	}
	if !isTransactionStatus(transaction.Status) {
		return &ValidationError{Code: "validation_error", Message: "Estado del movimiento inválido"}
	}
	if transaction.CategoryID != nil {
		if *transaction.CategoryID <= 0 {
			return &ValidationError{Code: "validation_error", Message: "Categoría inválida"}
		}
		exists, err := s.storage.CategoryExists(ctx, *transaction.CategoryID)
		if err != nil {
			return err
		}
		if !exists {
			return &ValidationError{Code: "validation_error", Message: "La categoría no existe"}
		}
	}
	if err := validateSign(transaction.Type, transaction.AmountLocal, "local"); err != nil {
		return err
	}
	if err := validateSign(transaction.Type, transaction.AmountUSD, "USD"); err != nil {
		return err
	}
	if len(transaction.Note) > maxNotes {
		return &ValidationError{Code: "validation_error", Message: "La nota es demasiado larga"}
	}
	transaction.AmountLocal = round2(transaction.AmountLocal)
	transaction.AmountUSD = round2(transaction.AmountUSD)
	return nil
}

func validateSign(txType string, amount float64, currency string) error {
	if amount == 0 {
		return nil
	}
	switch txType {
	case TypePayment, TypePaymentBonus, TypeRefund, TypeBonus:
		if amount > 0 {
			return &ValidationError{Code: "validation_error", Message: "Un abono no puede ser positivo (" + currency + ")"}
		}
	case TypePurchase, TypeCashAdvance, TypeCashAdvanceFee, TypeInterestCurrent, TypeInterestLate, TypeFee, TypeTax:
		if amount < 0 {
			return &ValidationError{Code: "validation_error", Message: "Un cargo no puede ser negativo (" + currency + ")"}
		}
	}
	return nil
}

func normalizeParsed(parsed *models.ParsedStatement) error {
	if parsed == nil {
		return ErrUnsupportedStatement
	}
	parsed.PeriodEnd = strings.TrimSpace(parsed.PeriodEnd)
	parsed.MinPaymentDue = strings.TrimSpace(parsed.MinPaymentDue)
	parsed.FullPaymentDue = strings.TrimSpace(parsed.FullPaymentDue)
	if !validDate(parsed.PeriodEnd) || !validDate(parsed.MinPaymentDue) || !validDate(parsed.FullPaymentDue) {
		return &ValidationError{Code: "unsupported_statement", Message: "El estado de cuenta no tiene fechas reconocibles"}
	}
	if parsed.ModuleID == "" {
		parsed.ModuleID = ModuleID
	}
	for i := range parsed.Transactions {
		item := &parsed.Transactions[i]
		item.PostedAt = strings.TrimSpace(item.PostedAt)
		item.Concept = strings.TrimSpace(item.Concept)
		item.Type = strings.ToLower(strings.TrimSpace(item.Type))
		item.CurrencyOriginal = strings.ToUpper(strings.TrimSpace(item.CurrencyOriginal))
		if !validDate(item.PostedAt) {
			return &ValidationError{Code: "unsupported_statement", Message: "Movimiento con fecha inválida"}
		}
		if item.Concept == "" {
			return &ValidationError{Code: "unsupported_statement", Message: "Movimiento sin concepto"}
		}
		if !isTransactionType(item.Type) {
			item.Type = TypeOther
		}
		if item.CurrencyOriginal == "" {
			if item.AmountUSD != 0 {
				item.CurrencyOriginal = "USD"
			} else {
				item.CurrencyOriginal = "NIO"
			}
		}
		item.AmountLocal = round2(item.AmountLocal)
		item.AmountUSD = round2(item.AmountUSD)
	}
	if parsed.Transactions == nil {
		parsed.Transactions = []models.ParsedTransaction{}
	}
	if parsed.Warnings == nil {
		parsed.Warnings = []models.ParseWarning{}
	}
	return nil
}

func hashParsed(parsed *models.ParsedStatement) (string, error) {
	payload, err := json.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("serializar preview: %w", err)
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func importHash(cardID int64, periodEnd, previewHash string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%s", cardID, periodEnd, previewHash)))
	return hex.EncodeToString(sum[:])
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func validAmount(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func validPercent(value float64) bool {
	return validAmount(value) && value >= 0 && value <= 100
}

func validDate(value string) bool {
	if !dateRegex.MatchString(value) {
		return false
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func isNetwork(value string) bool {
	for _, item := range Networks {
		if item == value {
			return true
		}
	}
	return false
}

func isColor(value string) bool {
	for _, item := range Colors {
		if item == value {
			return true
		}
	}
	return false
}

func isTransactionType(value string) bool {
	for _, item := range TransactionTypes {
		if item == value {
			return true
		}
	}
	return false
}

func isStatementStatus(value string) bool {
	for _, item := range StatementStatuses {
		if item == value {
			return true
		}
	}
	return false
}

func isTransactionStatus(value string) bool {
	for _, item := range TransactionStatuses {
		if item == value {
			return true
		}
	}
	return false
}
