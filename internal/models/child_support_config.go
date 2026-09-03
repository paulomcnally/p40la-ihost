package models

import "time"

// ChildSupportConfig define el monto/moneda de una categoría para un hijo,
// usado para la generación automática de registros (SPEC-051).
type ChildSupportConfig struct {
	ID                int64     `json:"id"`
	ChildID           int64     `json:"child_id"`
	ChildName         string    `json:"child_name,omitempty"`
	PensionCategoryID int64     `json:"pension_category_id"`
	CategoryName      string    `json:"category_name,omitempty"`
	Amount            float64   `json:"amount"`
	Currency          string    `json:"currency"`
	IsActive          bool      `json:"is_active"`
	AutoGenerate      bool      `json:"auto_generate"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
