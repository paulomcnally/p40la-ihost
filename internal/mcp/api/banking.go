package api

// BankRef es el subconjunto de un banco que necesita el MCP para resolver
// referencias y mostrar resultados.
type BankRef struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ShortCode string `json:"short_code"`
	Country   string `json:"country,omitempty"`
	Active    bool   `json:"active"`
}

// CurrencyRef es el subconjunto de una moneda que necesita el MCP.
type CurrencyRef struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name,omitempty"`
}

// BankAccount es una cuenta bancaria tal como la devuelve la API.
type BankAccount struct {
	ID             int64  `json:"id"`
	BankID         int64  `json:"bank_id"`
	BankName       string `json:"bank_name,omitempty"`
	Alias          string `json:"alias"`
	AccountNumber  string `json:"account_number"`
	AccountType    string `json:"account_type"`
	CurrencyID     int64  `json:"currency_id"`
	CurrencyCode   string `json:"currency_code,omitempty"`
	CurrencySymbol string `json:"currency_symbol,omitempty"`
	Beneficiary    string `json:"beneficiary"`
	Notes          string `json:"notes"`
	Active         bool   `json:"active"`
}

// BankAccountInput es el cuerpo de alta y edición de una cuenta bancaria.
// El backend usa PUT con reemplazo completo, así que siempre se envía completo.
type BankAccountInput struct {
	Alias         string `json:"alias"`
	AccountNumber string `json:"account_number"`
	AccountType   string `json:"account_type"`
	CurrencyID    int64  `json:"currency_id"`
	Beneficiary   string `json:"beneficiary"`
	Notes         string `json:"notes"`
	Active        bool   `json:"active"`
}

// CreditCard es una tarjeta de crédito tal como la devuelve la API.
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
}

// CreditCardInput es el cuerpo de alta y edición de una tarjeta. Solo incluye
// campos escribibles; los derivados (saldos, disponible) no se envían.
type CreditCardInput struct {
	Alias                  string  `json:"alias"`
	Network                string  `json:"network"`
	Last4                  string  `json:"last4"`
	Holder                 string  `json:"holder"`
	CurrencyID             int64   `json:"currency_id"`
	SecondaryCurrencyID    *int64  `json:"secondary_currency_id"`
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
}
