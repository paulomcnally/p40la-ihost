package models

import "time"

// Home representa una casa o propiedad que agrupa servicios.
type Home struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	Address        string     `json:"address,omitempty"`
	SortOrder      int64      `json:"sort_order"`
	ListingEnabled bool       `json:"listing_enabled"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
