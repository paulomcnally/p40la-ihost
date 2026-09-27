package models

import "time"

// CategoryGroup representa un agrupador visual de categorías de presupuesto
// (ej. "Necesidades", "Gustos"). SPEC-093.
type CategoryGroup struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}