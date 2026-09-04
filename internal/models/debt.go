package models

import "time"

// Debt representa una deuda con su plan de cuotas (SPEC-054).
// Status: activa (genera cuotas) / inactiva / finalizada.
type Debt struct {
	ID                int64      `json:"id"`
	InstitutionID     int64      `json:"institution_id"`
	InstitutionName   string     `json:"institution_name,omitempty"`
	Identifier        string     `json:"identifier"`
	Description       string     `json:"description"`
	Total             float64    `json:"total"`
	Principal         float64    `json:"principal"`
	CurrencyID        int64      `json:"currency_id"`
	CurrencyCode      string     `json:"currency_code,omitempty"`
	InstallmentsTotal int        `json:"installments_total"`
	InstallmentAmount float64    `json:"installment_amount"`
	InterestRate      float64    `json:"interest_rate"`
	PaymentDay        int        `json:"payment_day"`
	StartDate         string     `json:"start_date"`
	Status            string     `json:"status"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
