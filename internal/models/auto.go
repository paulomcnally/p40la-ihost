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
	Motor     string    `json:"motor"`
	Chasis    string    `json:"chasis"`
	VIN       string    `json:"vin"`
	Placa     string    `json:"placa"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AlertTypeNoInsurance y AlertTypeExpired son los tipos de alerta de autos.
const (
	AlertTypeNoInsurance = "no_insurance"
	AlertTypeExpired     = "expired"
)

// AutoAlert representa un auto que requiere atención en el email de alerta.
type AutoAlert struct {
	AutoID      int64  `json:"id"`
	Year        int64  `json:"year"`
	Model       string `json:"model"`
	Brand       string `json:"brand"`
	Color       string `json:"color"`
	Icon        string `json:"icon"`
	Placa       string `json:"placa"`
	AlertType   string `json:"alert_type"`
	EndDate     string `json:"end_date,omitempty"`
	ServiceName string `json:"service_name,omitempty"`
}
