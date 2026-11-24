package models

import "time"

// Usos de una vivienda (SPEC-143). El uso es manual; el estado de alquiler se
// deriva de contratos y reservas en el módulo realestate.rental.
const (
	HomeUsageRental    = "rental"
	HomeUsageInhabited = "inhabited"
	HomeUsageUnused    = "unused"
)

// HomeUsages contiene los usos válidos.
var HomeUsages = []string{HomeUsageRental, HomeUsageInhabited, HomeUsageUnused}

// Home representa una casa o propiedad que agrupa servicios.
type Home struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Address   string     `json:"address,omitempty"`
	SortOrder int64      `json:"sort_order"`
	Usage     string     `json:"usage"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
