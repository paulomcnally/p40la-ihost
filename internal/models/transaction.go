package models

import "time"

// Transaction representa un movimiento real de dinero asociado a una cuenta y
// opcionalmente a una categoría. SPEC-093. null category = "Sin categorizar".
type Transaction struct {
	ID           int64      `json:"id"`
	AccountID    int64      `json:"account_id"`
	AccountName  string     `json:"account_name,omitempty"`
	CategoryID   *int64     `json:"category_id,omitempty"`
	CategoryName string     `json:"category_name,omitempty"`
	CurrencyID   int64      `json:"currency_id"`
	CurrencyCode string     `json:"currency_code,omitempty"`
	Date         string     `json:"date"`
	Payee        string     `json:"payee"`
	Memo         string     `json:"memo"`
	Outflow      float64    `json:"outflow"`
	Inflow       float64    `json:"inflow"`
	Cleared      bool       `json:"cleared"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}