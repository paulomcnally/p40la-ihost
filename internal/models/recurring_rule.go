package models

import "time"

// RecurringRule representa una regla recurrente mensual de asignación.
// SPEC-093. Afecta desde start_month en adelante; no toca meses pasados.
type RecurringRule struct {
	ID         int64     `json:"id"`
	CategoryID int64     `json:"category_id"`
	Amount     float64   `json:"amount"`
	Frequency  string    `json:"frequency"`
	StartMonth string    `json:"start_month"`
	EndMonth   string    `json:"end_month"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}