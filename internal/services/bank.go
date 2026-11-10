package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

const (
	maxBankName          = 100
	maxBankShortCode     = 20
	maxBankCountry       = 80
	maxBankNotes         = 500
	maxBankAccountAlias  = 100
	maxBankAccountNumber = 50
	maxBankBeneficiary   = 150
	maxBankAccountNotes  = 500
	maxBankFormatFields  = 30
	maxBankFormatKey     = 40
	maxBankFormatValue   = 500
	maxBankFormatJSON    = 16 * 1024
)

var (
	// ErrBankNotFound indica que el banco no existe.
	ErrBankNotFound = storage.ErrBankNotFound
	// ErrBankNameExists indica un nombre de banco duplicado.
	ErrBankNameExists = storage.ErrBankNameExists
	// ErrBankHasAccounts indica que el banco tiene cuentas.
	ErrBankHasAccounts = storage.ErrBankHasAccounts
	// ErrBankAccountNotFound indica que la cuenta no existe.
	ErrBankAccountNotFound = storage.ErrBankAccountNotFound
	// ErrBankAccountDuplicate indica un número de cuenta duplicado.
	ErrBankAccountDuplicate = storage.ErrBankAccountDuplicate
	// ErrBankTransferFormatNotFound indica que el formato no existe.
	ErrBankTransferFormatNotFound = storage.ErrBankTransferFormatNotFound
)

var bankFormatKeyRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)

// BankValidationError es un error de validación del módulo bancario.
type BankValidationError struct {
	Code    string
	Message string
}

func (e *BankValidationError) Error() string { return e.Message }

func newBankValidationError(message string) error {
	return &BankValidationError{Code: "validation_error", Message: message}
}

func newBankInvalidVariableError(message string) error {
	return &BankValidationError{Code: "invalid_variable", Message: message}
}

// bankTransferVariables son las variables propias del contexto bancario.
var bankTransferVariables = []models.DebtInformationVariable{
	{Key: "bank_name", Label: "Nombre del banco", Example: "Banco Demo", Description: "Nombre del banco de la cuenta"},
	{Key: "bank_short_code", Label: "Código corto del banco", Example: "BDEMO", Description: "Código corto del banco"},
	{Key: "account_alias", Label: "Alias de la cuenta", Example: "Cuenta principal", Description: "Alias o etiqueta de la cuenta"},
	{Key: "account_number", Label: "Número de cuenta", Example: "0000000001", Description: "Número de la cuenta bancaria"},
	{Key: "account_type", Label: "Tipo de cuenta", Example: "Ahorro", Description: "Ahorro o Corriente, según el idioma"},
	{Key: "account_beneficiary", Label: "Beneficiario de la cuenta", Example: "Beneficiario Demo", Description: "Nombre del beneficiario de la cuenta"},
	{Key: "account_currency", Label: "Moneda de la cuenta", Example: "COR", Description: "Código de la moneda de la cuenta"},
	{Key: "account_currency_code", Label: "Código de moneda de la cuenta", Example: "COR", Description: "Alias de account_currency"},
	{Key: "account_currency_symbol", Label: "Símbolo de la moneda de la cuenta", Example: "C$", Description: "Símbolo de la moneda de la cuenta"},
	{Key: "account_notes", Label: "Notas de la cuenta", Example: "Cuenta para pagos", Description: "Notas de la cuenta bancaria"},
}

var bankTransferVariableSet = func() map[string]bool {
	set := make(map[string]bool, len(debtInformationValueVariables)+len(bankTransferVariables))
	for _, variable := range debtInformationValueVariables {
		set[variable.Key] = true
	}
	for _, variable := range bankTransferVariables {
		set[variable.Key] = true
	}
	return set
}()

// BankService contiene la lógica de bancos, cuentas y formatos (SPEC-133).
type BankService struct {
	banks      *storage.BankStorage
	currencies *storage.CurrencyStorage
}

// NewBankService crea el servicio.
func NewBankService(banks *storage.BankStorage, currencies *storage.CurrencyStorage) *BankService {
	return &BankService{banks: banks, currencies: currencies}
}

// Variables devuelve el catálogo de variables para formatos de transferencia
// (contexto bancario + variables de deuda/cuota de SPEC-132).
func (s *BankService) Variables() []models.DebtInformationVariable {
	result := make([]models.DebtInformationVariable, 0, len(bankTransferVariables)+len(debtInformationValueVariables))
	result = append(result, bankTransferVariables...)
	result = append(result, debtInformationValueVariables...)
	return result
}

// ListBanks lista los bancos.
func (s *BankService) ListBanks(ctx context.Context) ([]models.Bank, error) {
	return s.banks.ListBanks(ctx)
}

// GetBank devuelve un banco o nil.
func (s *BankService) GetBank(ctx context.Context, id int64) (*models.Bank, error) {
	return s.banks.GetBankByID(ctx, id)
}

// CreateBank valida y crea un banco.
func (s *BankService) CreateBank(ctx context.Context, bank *models.Bank) (*models.Bank, error) {
	if err := prepareBank(bank); err != nil {
		return nil, err
	}
	return s.banks.CreateBank(ctx, bank)
}

// UpdateBank valida y actualiza un banco.
func (s *BankService) UpdateBank(ctx context.Context, bank *models.Bank) (*models.Bank, error) {
	if bank.ID <= 0 {
		return nil, newBankValidationError("El ID del banco es obligatorio.")
	}
	if err := prepareBank(bank); err != nil {
		return nil, err
	}
	return s.banks.UpdateBank(ctx, bank)
}

// DeleteBank elimina un banco sin cuentas.
func (s *BankService) DeleteBank(ctx context.Context, id int64) error {
	return s.banks.DeleteBank(ctx, id)
}

// ListAccounts lista las cuentas de un banco.
func (s *BankService) ListAccounts(ctx context.Context, bankID int64) ([]models.BankAccount, error) {
	bank, err := s.banks.GetBankByID(ctx, bankID)
	if err != nil {
		return nil, err
	}
	if bank == nil {
		return nil, ErrBankNotFound
	}
	return s.banks.ListAccountsByBank(ctx, bankID)
}

// GetAccount devuelve una cuenta con sus formatos o nil.
func (s *BankService) GetAccount(ctx context.Context, id int64) (*models.BankAccount, error) {
	return s.banks.GetAccountByID(ctx, id)
}

// CreateAccount valida y crea una cuenta.
func (s *BankService) CreateAccount(ctx context.Context, account *models.BankAccount) (*models.BankAccount, error) {
	if err := s.prepareAccount(ctx, account); err != nil {
		return nil, err
	}
	return s.banks.CreateAccount(ctx, account)
}

// UpdateAccount valida y actualiza una cuenta.
func (s *BankService) UpdateAccount(ctx context.Context, account *models.BankAccount) (*models.BankAccount, error) {
	if account.ID <= 0 {
		return nil, newBankValidationError("El ID de la cuenta es obligatorio.")
	}
	existing, err := s.banks.GetAccountByID(ctx, account.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrBankAccountNotFound
	}
	// El banco de una cuenta no cambia desde la edición de la cuenta.
	account.BankID = existing.BankID
	if err := s.prepareAccount(ctx, account); err != nil {
		return nil, err
	}
	return s.banks.UpdateAccount(ctx, account)
}

// DeleteAccount elimina una cuenta y sus formatos.
func (s *BankService) DeleteAccount(ctx context.Context, id int64) error {
	return s.banks.DeleteAccount(ctx, id)
}

// ListFormats lista los formatos de una cuenta.
func (s *BankService) ListFormats(ctx context.Context, accountID int64) ([]models.BankTransferFormat, error) {
	account, err := s.banks.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrBankAccountNotFound
	}
	return s.banks.ListFormats(ctx, accountID)
}

// UpsertFormat valida y guarda un formato de transferencia.
func (s *BankService) UpsertFormat(ctx context.Context, accountID int64, kind string, fields map[string]string) (*models.BankTransferFormat, error) {
	if !isBankTransferKind(kind) {
		return nil, newBankValidationError("El tipo de formato de transferencia no es válido.")
	}
	account, err := s.banks.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrBankAccountNotFound
	}
	cleanFields, err := validateBankTransferFields(fields)
	if err != nil {
		return nil, err
	}
	return s.banks.UpsertFormat(ctx, accountID, kind, cleanFields)
}

// DeleteFormat elimina un formato de una cuenta.
func (s *BankService) DeleteFormat(ctx context.Context, accountID int64, kind string) error {
	if !isBankTransferKind(kind) {
		return newBankValidationError("El tipo de formato de transferencia no es válido.")
	}
	account, err := s.banks.GetAccountByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account == nil {
		return ErrBankAccountNotFound
	}
	return s.banks.DeleteFormat(ctx, accountID, kind)
}

func prepareBank(bank *models.Bank) error {
	bank.Name = strings.TrimSpace(bank.Name)
	bank.ShortCode = strings.TrimSpace(bank.ShortCode)
	bank.Country = strings.TrimSpace(bank.Country)
	bank.Notes = strings.TrimSpace(bank.Notes)

	if bank.Name == "" {
		return newBankValidationError("El nombre del banco es obligatorio.")
	}
	if utf8.RuneCountInString(bank.Name) > maxBankName {
		return newBankValidationError(fmt.Sprintf("El nombre del banco no puede superar los %d caracteres.", maxBankName))
	}
	if utf8.RuneCountInString(bank.ShortCode) > maxBankShortCode {
		return newBankValidationError(fmt.Sprintf("El código corto no puede superar los %d caracteres.", maxBankShortCode))
	}
	if utf8.RuneCountInString(bank.Country) > maxBankCountry {
		return newBankValidationError(fmt.Sprintf("El país no puede superar los %d caracteres.", maxBankCountry))
	}
	if utf8.RuneCountInString(bank.Notes) > maxBankNotes {
		return newBankValidationError(fmt.Sprintf("Las notas no pueden superar los %d caracteres.", maxBankNotes))
	}
	return nil
}

func (s *BankService) prepareAccount(ctx context.Context, account *models.BankAccount) error {
	account.Alias = strings.TrimSpace(account.Alias)
	account.AccountNumber = strings.TrimSpace(account.AccountNumber)
	account.AccountType = strings.TrimSpace(strings.ToLower(account.AccountType))
	account.Beneficiary = strings.TrimSpace(account.Beneficiary)
	account.Notes = strings.TrimSpace(account.Notes)

	if account.BankID <= 0 {
		return newBankValidationError("El banco es obligatorio.")
	}
	bank, err := s.banks.GetBankByID(ctx, account.BankID)
	if err != nil {
		return err
	}
	if bank == nil {
		return ErrBankNotFound
	}
	if account.Alias == "" {
		return newBankValidationError("El alias de la cuenta es obligatorio.")
	}
	if utf8.RuneCountInString(account.Alias) > maxBankAccountAlias {
		return newBankValidationError(fmt.Sprintf("El alias no puede superar los %d caracteres.", maxBankAccountAlias))
	}
	if account.AccountNumber == "" {
		return newBankValidationError("El número de cuenta es obligatorio.")
	}
	if utf8.RuneCountInString(account.AccountNumber) > maxBankAccountNumber {
		return newBankValidationError(fmt.Sprintf("El número de cuenta no puede superar los %d caracteres.", maxBankAccountNumber))
	}
	if !isBankAccountType(account.AccountType) {
		return newBankValidationError("El tipo de cuenta no es válido.")
	}
	if account.CurrencyID <= 0 {
		return newBankValidationError("La moneda de la cuenta es obligatoria.")
	}
	currency, err := s.currencies.GetByID(ctx, account.CurrencyID)
	if err != nil {
		return err
	}
	if currency == nil {
		return newBankValidationError("La moneda de la cuenta no existe.")
	}
	if account.Beneficiary == "" {
		return newBankValidationError("El beneficiario de la cuenta es obligatorio.")
	}
	if utf8.RuneCountInString(account.Beneficiary) > maxBankBeneficiary {
		return newBankValidationError(fmt.Sprintf("El beneficiario no puede superar los %d caracteres.", maxBankBeneficiary))
	}
	if utf8.RuneCountInString(account.Notes) > maxBankAccountNotes {
		return newBankValidationError(fmt.Sprintf("Las notas de la cuenta no pueden superar los %d caracteres.", maxBankAccountNotes))
	}
	return nil
}

func validateBankTransferFields(fields map[string]string) (map[string]string, error) {
	if fields == nil {
		return map[string]string{}, nil
	}
	if len(fields) > maxBankFormatFields {
		return nil, newBankValidationError(fmt.Sprintf("El formato no puede tener más de %d campos.", maxBankFormatFields))
	}
	clean := make(map[string]string, len(fields))
	for rawKey, rawValue := range fields {
		key := strings.TrimSpace(rawKey)
		value := strings.TrimSpace(rawValue)
		if key == "" {
			return nil, newBankValidationError("El formato tiene una llave vacía.")
		}
		if !bankFormatKeyRegex.MatchString(key) {
			return nil, newBankValidationError(fmt.Sprintf("La llave '%s' no es válida. Usá minúsculas, números y guion bajo, empezando por una letra.", key))
		}
		if utf8.RuneCountInString(key) > maxBankFormatKey {
			return nil, newBankValidationError(fmt.Sprintf("La llave '%s' no puede superar los %d caracteres.", key, maxBankFormatKey))
		}
		if utf8.RuneCountInString(value) > maxBankFormatValue {
			return nil, newBankValidationError(fmt.Sprintf("El valor de la llave '%s' no puede superar los %d caracteres.", key, maxBankFormatValue))
		}
		if err := validateVariableText(value, bankTransferVariableSet); err != nil {
			var validation *DebtInformationValidationError
			if errors.As(err, &validation) {
				return nil, newBankInvalidVariableError(fmt.Sprintf("%s (llave '%s')", validation.Message, key))
			}
			return nil, err
		}
		clean[key] = value
	}
	encoded, err := json.Marshal(clean)
	if err != nil {
		return nil, newBankValidationError("No se pudo serializar el formato de transferencia.")
	}
	if len(encoded) > maxBankFormatJSON {
		return nil, newBankValidationError("El formato de transferencia es demasiado grande.")
	}
	return clean, nil
}

func isBankAccountType(value string) bool {
	for _, accountType := range models.BankAccountTypes {
		if accountType == value {
			return true
		}
	}
	return false
}

func isBankTransferKind(value string) bool {
	for _, kind := range models.BankTransferKinds {
		if kind == value {
			return true
		}
	}
	return false
}
