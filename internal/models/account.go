package models

import "time"

// Account representa una cuenta de dinero (banco, tarjeta, efectivo, ahorro).
// SPEC-093. Independiente de instituciones.
type Account struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	CurrencyID      int64      `json:"currency_id"`
	CurrencyCode    string     `json:"currency_code,omitempty"`
	StartingBalance float64    `json:"starting_balance"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}