package models

import "time"

// Home representa una casa o propiedad que agrupa servicios.
type Home struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Address   string     `json:"address,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
