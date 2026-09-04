package models

import "time"

// DebtBill representa una cuota de una deuda (SPEC-054).
// Estados idénticos a Bill: pending / paid.
type DebtBill struct {
	ID                int64      `json:"id"`
	DebtID            int64      `json:"debt_id"`
	DebtDescription   string     `json:"debt_description,omitempty"`
	InstitutionName   string     `json:"institution_name,omitempty"`
	CurrencyCode      string     `json:"currency_code,omitempty"`
	InstallmentNumber int        `json:"installment_number"`
	DueDate           string     `json:"due_date"`
	Amount            float64    `json:"amount"`
	Status            string     `json:"status"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	PaymentReference  string     `json:"payment_reference,omitempty"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
