package models

import "time"

// AutoMaintenance representa un mantenimiento realizado a un auto, medido por
// kilometraje (SPEC-124).
type AutoMaintenance struct {
	ID            int64     `json:"id"`
	AutoID        int64     `json:"auto_id"`
	PerformedAt   string    `json:"performed_at"`
	Mileage       int64     `json:"mileage"`
	InstitutionID *int64    `json:"institution_id,omitempty"`
	WorkshopName  string    `json:"workshop_name"`
	Description   string    `json:"description"`
	LaborCost     float64   `json:"labor_cost"`
	SuppliesCost  float64   `json:"supplies_cost"`
	CurrencyID    int64     `json:"currency_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AutoMaintenanceDetail agrega los datos calculados que se muestran en la UI.
type AutoMaintenanceDetail struct {
	AutoMaintenance
	InstitutionName string           `json:"institution_name,omitempty"`
	CurrencyCode    string           `json:"currency_code"`
	TotalCost       float64          `json:"total_cost"`
	Attachments     []AutoAttachment `json:"attachments"`
}

// AutoMaintenanceInput es el payload de creación/edición.
type AutoMaintenanceInput struct {
	PerformedAt   string  `json:"performed_at"`
	Mileage       int64   `json:"mileage"`
	InstitutionID *int64  `json:"institution_id,omitempty"`
	WorkshopName  string  `json:"workshop_name"`
	Description   string  `json:"description"`
	LaborCost     float64 `json:"labor_cost"`
	SuppliesCost  float64 `json:"supplies_cost"`
	CurrencyID    int64   `json:"currency_id"`
}

// AutoMaintenanceCurrencyTotal es el total de mantenimientos por moneda.
type AutoMaintenanceCurrencyTotal struct {
	CurrencyID   int64   `json:"currency_id"`
	CurrencyCode string  `json:"currency_code"`
	Total        float64 `json:"total"`
}

// AutoMaintenanceList es la respuesta de GET /api/autos/{id}/maintenances.
type AutoMaintenanceList struct {
	Items            []AutoMaintenanceDetail        `json:"items"`
	LatestMileage    int64                          `json:"latest_mileage"`
	TotalsByCurrency []AutoMaintenanceCurrencyTotal `json:"totals_by_currency"`
}
