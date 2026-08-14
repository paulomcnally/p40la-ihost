package models

import "time"

// Currency representa una moneda disponible en el sistema.
type Currency struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	Symbol    string     `json:"symbol"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
