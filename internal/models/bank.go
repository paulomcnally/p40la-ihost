package models

import "time"

// Tipos de cuenta bancaria soportados (SPEC-133).
const (
	BankAccountTypeSavings  = "savings"
	BankAccountTypeChecking = "checking"
)

// BankAccountTypes es la lista canónica de tipos de cuenta.
var BankAccountTypes = []string{BankAccountTypeSavings, BankAccountTypeChecking}

// Tipos de formato de transferencia soportados (SPEC-133).
const (
	BankTransferKindTransfer = "transfer"
	BankTransferKindACH      = "ach"
)

// BankTransferKinds es la lista canónica de tipos de formato.
var BankTransferKinds = []string{BankTransferKindTransfer, BankTransferKindACH}

// Bank representa un banco con sus cuentas (SPEC-133).
type Bank struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	ShortCode    string    `json:"short_code"`
	Country      string    `json:"country"`
	Notes        string    `json:"notes"`
	Active       bool      `json:"active"`
	AccountCount int       `json:"account_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BankAccount representa una cuenta bancaria de un banco. Formats se incluye
// solo en el detalle; en los listados va vacío.
type BankAccount struct {
	ID             int64                `json:"id"`
	BankID         int64                `json:"bank_id"`
	BankName       string               `json:"bank_name,omitempty"`
	Alias          string               `json:"alias"`
	AccountNumber  string               `json:"account_number"`
	AccountType    string               `json:"account_type"`
	CurrencyID     int64                `json:"currency_id"`
	CurrencyCode   string               `json:"currency_code,omitempty"`
	CurrencySymbol string               `json:"currency_symbol,omitempty"`
	Beneficiary    string               `json:"beneficiary"`
	Notes          string               `json:"notes"`
	Active         bool                 `json:"active"`
	Formats        []BankTransferFormat `json:"formats,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

// BankTransferFormat es un formato de transferencia (Transferencia o ACH) de
// una cuenta. Fields se persiste como JSON de pares llave/valor.
type BankTransferFormat struct {
	ID            int64             `json:"id"`
	BankAccountID int64             `json:"bank_account_id"`
	Kind          string            `json:"kind"`
	Fields        map[string]string `json:"fields"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}
