package models

import "time"

// Category representa una categoría de presupuesto dentro de un CategoryGroup.
// SPEC-093. Se archiva (soft delete) si tiene transacciones/assignments.
// SPEC-094: puede vincularse a un servicio del sistema para generar
// transacciones automáticas al pagar.
// SPEC-096: una categoría puede vincularse a N servicios (tabla de enlace
// category_service_links); ServiceIDs/ServiceNames son solo lectura (join).
type Category struct {
	ID              int64      `json:"id"`
	CategoryGroupID int64      `json:"category_group_id"`
	Name            string     `json:"name"`
	Icon            string     `json:"icon"`
	SortOrder       int        `json:"sort_order"`
	TargetAmount    *float64   `json:"target_amount,omitempty"`
	TargetType      string     `json:"target_type,omitempty"`
	TargetDate      string     `json:"target_date,omitempty"`
	ServiceIDs      []int64    `json:"service_ids,omitempty"`
	ServiceNames    []string   `json:"service_names,omitempty"`
	AccountID       *int64     `json:"account_id,omitempty"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
