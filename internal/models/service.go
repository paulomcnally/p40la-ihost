package models

import "time"

// Service representa un servicio recurrente de pago asociado a un hogar.
type Service struct {
	ID              int64      `json:"id"`
	HomeID          int64      `json:"home_id"`
	Name            string     `json:"name"`
	Institution     string     `json:"institution,omitempty"`
	CurrencyID      int64      `json:"currency_id"`
	Frequency       string     `json:"frequency"`
	SuggestedAmount float64    `json:"suggested_amount"`
	Active          bool       `json:"active"`
	IconKey         string     `json:"icon_key"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
