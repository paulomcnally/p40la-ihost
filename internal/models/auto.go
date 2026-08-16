package models

import "time"

// Auto representa un vehículo registrado.
type Auto struct {
	ID        int64     `json:"id"`
	Year      int64     `json:"year"`
	Model     string    `json:"model"`
	Brand     string    `json:"brand"`
	Color     string    `json:"color"`
	Icon      string    `json:"icon"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
