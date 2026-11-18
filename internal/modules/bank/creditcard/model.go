package creditcard

import "time"

// Identificador canónico del módulo core (ARCH-MODULES).
const (
	ModuleID      = "bank.creditcard"
	ModuleVersion = "1.1.0"
)

// Marcas de tarjeta soportadas.
const (
	NetworkVisa       = "visa"
	NetworkMastercard = "mastercard"
	NetworkAmex       = "amex"
	NetworkDiners     = "diners"
	NetworkOther      = "other"
)

// Networks contiene las marcas válidas.
var Networks = []string{NetworkVisa, NetworkMastercard, NetworkAmex, NetworkDiners, NetworkOther}

// Monedas en las que se puede expresar el límite de crédito (SPEC-138).
const (
	LimitCurrencyLocal = "NIO"
	LimitCurrencyUSD   = "USD"
)

// LimitCurrencies contiene las monedas válidas para el límite.
var LimitCurrencies = []string{LimitCurrencyLocal, LimitCurrencyUSD}

// Categorías de producto de la tarjeta (SPEC-138). "" = no definida.
var Categories = []string{"", "classic", "gold", "platinum", "black", "business", "other"}

// Tipos de movimiento.
const (
	TypePurchase        = "purchase"
	TypePayment         = "payment"
	TypePaymentBonus    = "payment_bonus"
	TypeCashAdvance     = "cash_advance"
	TypeCashAdvanceFee  = "cash_advance_fee"
	TypeInterestCurrent = "interest_current"
	TypeInterestLate    = "interest_late"
	TypeFee             = "fee"
	TypeTax             = "tax"
	TypeBonus           = "bonus"
	TypeRefund          = "refund"
	TypeAdjustment      = "adjustment"
	TypeOther           = "other"
)

// TransactionTypes contiene los tipos válidos.
var TransactionTypes = []string{
	TypePurchase, TypePayment, TypePaymentBonus, TypeCashAdvance, TypeCashAdvanceFee,
	TypeInterestCurrent, TypeInterestLate, TypeFee, TypeTax, TypeBonus, TypeRefund,
	TypeAdjustment, TypeOther,
}

// Estados de un estado de cuenta.
const (
	StatementUnpaid  = "unpaid"
	StatementPartial = "partial"
	StatementPaid    = "paid"
)

// StatementStatuses contiene los estados válidos.
var StatementStatuses = []string{StatementUnpaid, StatementPartial, StatementPaid}

// Orígenes de un estado de cuenta.
const (
	SourceManual = "manual"
	SourceImport = "import"
)

// Estados de un movimiento.
const (
	TransactionPosted  = "posted"
	TransactionPending = "pending"
)

// TransactionStatuses contiene los estados válidos.
var TransactionStatuses = []string{TransactionPosted, TransactionPending}

// Colors es la paleta canónica de tarjetas. Debe coincidir con
// frontend/src/constants/cardColors.ts.
var Colors = []string{
	"white", "black", "gray", "silver", "red", "blue", "green", "yellow",
	"orange", "brown", "beige", "gold", "burgundy", "light_blue", "pink", "purple",
}

// CreditCard es una tarjeta de crédito de un banco.
type CreditCard struct {
	ID                     int64   `json:"id"`
	BankID                 int64   `json:"bank_id"`
	BankName               string  `json:"bank_name,omitempty"`
	Alias                  string  `json:"alias"`
	Network                string  `json:"network"`
	Last4                  string  `json:"last4"`
	Holder                 string  `json:"holder"`
	CurrencyID             int64   `json:"currency_id"`
	CurrencyCode           string  `json:"currency_code,omitempty"`
	SecondaryCurrencyID    *int64  `json:"secondary_currency_id,omitempty"`
	SecondaryCurrencyCode  string  `json:"secondary_currency_code,omitempty"`
	Category               string  `json:"category"`
	LimitCurrency          string  `json:"limit_currency"`
	ExchangeRate           float64 `json:"exchange_rate"`
	CreditLimit            float64 `json:"credit_limit"`
	CreditLimitUSD         float64 `json:"credit_limit_usd"`
	StatementDay           int     `json:"statement_day"`
	PaymentDueDay          int     `json:"payment_due_day"`
	BonusDueDay            int     `json:"bonus_due_day"`
	MinPaymentPct          float64 `json:"min_payment_pct"`
	InterestCurrentRate    float64 `json:"interest_current_rate"`
	InterestLateRate       float64 `json:"interest_late_rate"`
	InterestCurrentRateUSD float64 `json:"interest_current_rate_usd"`
	InterestLateRateUSD    float64 `json:"interest_late_rate_usd"`
	Color                  string  `json:"color"`
	Icon                   string  `json:"icon"`
	Active                 bool    `json:"active"`
	Notes                  string  `json:"notes"`
	StatementCount         int     `json:"statement_count"`

	// Derivados (no se almacenan).
	LateTotalRate    float64 `json:"late_total_rate"`
	LateTotalRateUSD float64 `json:"late_total_rate_usd"`
	BalanceLocal     float64 `json:"balance_local"`
	BalanceUSD       float64 `json:"balance_usd"`
	AvailableLocal   float64 `json:"available_local"`
	AvailableUSD     float64 `json:"available_usd"`
	LastStatementID  *int64  `json:"last_statement_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Statement es el estado de cuenta mensual de una tarjeta.
type Statement struct {
	ID                 int64     `json:"id"`
	CardID             int64     `json:"card_id"`
	PeriodStart        *string   `json:"period_start,omitempty"`
	PeriodEnd          string    `json:"period_end"`
	MinPaymentDueDate  string    `json:"min_payment_due_date"`
	FullPaymentDueDate string    `json:"full_payment_due_date"`
	PreviousBalance    float64   `json:"previous_balance"`
	PreviousBalanceUSD float64   `json:"previous_balance_usd"`
	InterestCurrent    float64   `json:"interest_current"`
	InterestCurrentUSD float64   `json:"interest_current_usd"`
	MinPayment         float64   `json:"min_payment"`
	MinPaymentUSD      float64   `json:"min_payment_usd"`
	TotalPayment       float64   `json:"total_payment"`
	TotalPaymentUSD    float64   `json:"total_payment_usd"`
	NewBalance         float64   `json:"new_balance"`
	NewBalanceUSD      float64   `json:"new_balance_usd"`
	Status             string    `json:"status"`
	Source             string    `json:"source"`
	SourceModuleID     string    `json:"source_module_id,omitempty"`
	ImportHash         string    `json:"-"`
	Notes              string    `json:"notes"`
	TransactionCount   int       `json:"transaction_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	// Derivados solo en el detalle.
	Summary        []StatementSummary `json:"summary,omitempty"`
	Reconciliation *Reconciliation    `json:"reconciliation,omitempty"`
}

// StatementSummary agrega los movimientos por tipo.
type StatementSummary struct {
	Type        string  `json:"type"`
	AmountLocal float64 `json:"amount_local"`
	AmountUSD   float64 `json:"amount_usd"`
	Count       int     `json:"count"`
}

// Reconciliation compara los totales impresos con la suma de movimientos.
type Reconciliation struct {
	ExpectedLocal   float64 `json:"expected_local"`
	ActualLocal     float64 `json:"actual_local"`
	DifferenceLocal float64 `json:"difference_local"`
	ExpectedUSD     float64 `json:"expected_usd"`
	ActualUSD       float64 `json:"actual_usd"`
	DifferenceUSD   float64 `json:"difference_usd"`
	Balanced        bool    `json:"balanced"`
}

// Transaction es un movimiento de la tarjeta.
type Transaction struct {
	ID               int64     `json:"id"`
	CardID           int64     `json:"card_id"`
	StatementID      *int64    `json:"statement_id,omitempty"`
	PostedAt         string    `json:"posted_at"`
	Concept          string    `json:"concept"`
	CategoryID       *int64    `json:"category_id,omitempty"`
	CategoryName     string    `json:"category_name,omitempty"`
	Type             string    `json:"type"`
	AmountLocal      float64   `json:"amount_local"`
	AmountUSD        float64   `json:"amount_usd"`
	CurrencyOriginal string    `json:"currency_original"`
	Status           string    `json:"status"`
	ExternalRef      string    `json:"external_ref,omitempty"`
	Note             string    `json:"note"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
