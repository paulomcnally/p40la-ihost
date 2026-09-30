package models

import "time"

// Transfer representa un movimiento de dinero entre dos cuentas del presupuesto
// (SPEC-099). Es neutro: no afecta la actividad por categoría ni los ingresos;
// solo el balance de las cuentas origen (resta) y destino (suma).
type Transfer struct {
	ID              int64      `json:"id"`
	FromAccountID   int64      `json:"from_account_id"`
	FromAccountName string     `json:"from_account_name,omitempty"`
	ToAccountID     int64      `json:"to_account_id"`
	ToAccountName   string     `json:"to_account_name,omitempty"`
	CurrencyID      int64      `json:"currency_id"`
	CurrencyCode    string     `json:"currency_code,omitempty"`
	Date            string     `json:"date"`
	Payee           string     `json:"payee"`
	Memo            string     `json:"memo"`
	Amount          float64    `json:"amount"`
	Cleared         bool       `json:"cleared"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
