package models

import "time"

// BudgetMonth representa un mes calendario del presupuesto.
// SPEC-093. Se crea on-demand al materializar un mes.
type BudgetMonth struct {
	ID        int64     `json:"id"`
	Year      int       `json:"year"`
	Month     int       `json:"month"`
	CreatedAt time.Time `json:"created_at"`
}