package models

import "time"

// SupportRecord representa un registro mensual de manutención (SPEC-049).
// child_name y category_name se resuelven con JOIN al listar, para que el
// frontend no necesite joins.
type SupportRecord struct {
	ID                int64      `json:"id"`
	ChildID           int64      `json:"child_id"`
	ChildName         string     `json:"child_name,omitempty"`
	PensionCategoryID int64      `json:"pension_category_id"`
	CategoryName      string     `json:"category_name,omitempty"`
	Year              int        `json:"year"`
	Month             int        `json:"month"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	PaymentMethod     *string    `json:"payment_method,omitempty"`
	PaymentReference  *string    `json:"payment_reference,omitempty"`
	EvidenceNotes     *string    `json:"evidence_notes,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	ProofFileName     *string    `json:"proof_file_name,omitempty"`
	OriginalAmount    *float64   `json:"original_amount,omitempty"`
	OriginalCurrency  *string    `json:"original_currency,omitempty"`
	ExchangeRate      *float64   `json:"exchange_rate,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
