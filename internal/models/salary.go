package models

import "time"

// Salary representa un salario/ingreso registrado en el módulo de Pensión Alimenticia.
type Salary struct {
	ID         int64     `json:"id"`
	Employer   string    `json:"employer"`
	Amount     float64   `json:"amount"`
	CurrencyID int64     `json:"currency_id"`
	PaymentDay int       `json:"payment_day"`
	Active     bool      `json:"active"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}