package models

import "time"

// PensionCategory representa una categoría de gastos del módulo de Pensión Alimenticia.
type PensionCategory struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	AutoGenerate bool      `json:"auto_generate"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}