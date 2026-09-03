package models

import "time"

// MonthClosing representa un mes cerrado (SPEC-049).
// La existencia de una fila (year, month) significa que el mes está cerrado.
type MonthClosing struct {
	ID       int64     `json:"id"`
	Year     int       `json:"year"`
	Month    int       `json:"month"`
	ClosedAt time.Time `json:"closed_at"`
}
