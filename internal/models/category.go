package models

import "time"

// Category representa una categoría de presupuesto dentro de un CategoryGroup.
// SPEC-093. Se archiva (soft delete) si tiene transacciones/assignments.
type Category struct {
	ID              int64      `json:"id"`
	CategoryGroupID int64      `json:"category_group_id"`
	Name            string     `json:"name"`
	Icon            string     `json:"icon"`
	SortOrder       int        `json:"sort_order"`
	TargetAmount    *float64   `json:"target_amount,omitempty"`
	TargetType      string     `json:"target_type,omitempty"`
	TargetDate      string     `json:"target_date,omitempty"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}