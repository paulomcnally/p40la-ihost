package models

// ParsedTransaction es una fila de movimiento extraída de un estado de cuenta
// por un importador. Es un contrato entre módulos (ARCH-MODULES).
type ParsedTransaction struct {
	PostedAt         string  `json:"posted_at"`
	Concept          string  `json:"concept"`
	Type             string  `json:"type"`
	AmountLocal      float64 `json:"amount_local"`
	AmountUSD        float64 `json:"amount_usd"`
	CurrencyOriginal string  `json:"currency_original"`
	ExternalRef      string  `json:"external_ref"`
}

// ParseWarning describe una inconsistencia detectada durante el parseo.
type ParseWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ParsedStatement es el resultado de un importador antes de persistirse.
type ParsedStatement struct {
	ModuleID              string              `json:"module_id"`
	ModuleVersion         string              `json:"module_version"`
	Last4                 string              `json:"last4"`
	PeriodEnd             string              `json:"period_end"`
	MinPaymentDue         string              `json:"min_payment_due_date"`
	FullPaymentDue        string              `json:"full_payment_due_date"`
	PreviousBalance       float64             `json:"previous_balance"`
	PreviousBalanceUSD    float64             `json:"previous_balance_usd"`
	InterestCurrent       float64             `json:"interest_current"`
	InterestCurrentUSD    float64             `json:"interest_current_usd"`
	InterestBonifiable    float64             `json:"interest_bonifiable"`
	InterestBonifiableUSD float64             `json:"interest_bonifiable_usd"`
	MoraAmount            float64             `json:"mora_amount"`
	MoraAmountUSD         float64             `json:"mora_amount_usd"`
	ArrearsCount          int                 `json:"arrears_count"`
	OverLimitAmount       float64             `json:"over_limit_amount"`
	OverLimitAmountUSD    float64             `json:"over_limit_amount_usd"`
	PointsPrevious        int                 `json:"points_previous"`
	PointsEarned          int                 `json:"points_earned"`
	PointsUsed            int                 `json:"points_used"`
	PointsAvailable       int                 `json:"points_available"`
	MinPayment            float64             `json:"min_payment"`
	MinPaymentUSD         float64             `json:"min_payment_usd"`
	TotalPayment          float64             `json:"total_payment"`
	TotalPaymentUSD       float64             `json:"total_payment_usd"`
	NewBalance            float64             `json:"new_balance"`
	NewBalanceUSD         float64             `json:"new_balance_usd"`
	Transactions          []ParsedTransaction `json:"transactions"`
	Warnings              []ParseWarning      `json:"warnings,omitempty"`
}
