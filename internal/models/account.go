package models

import "time"

// Account representa una cuenta de dinero (banco, tarjeta, efectivo, ahorro).
// SPEC-093. Independiente de instituciones.
// SPEC-099: Balance es el saldo calculado (starting_balance + transacciones +
// transferencias); no se persiste en la tabla.
type Account struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	CurrencyID      int64      `json:"currency_id"`
	CurrencyCode    string     `json:"currency_code,omitempty"`
	StartingBalance float64    `json:"starting_balance"`
	Balance         float64    `json:"balance"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
