package models

import "time"

const (
	// AutoMaintenanceItemLabor identifica una línea de mano de obra.
	AutoMaintenanceItemLabor = "labor"
	// AutoMaintenanceItemSupplies identifica una línea de insumos.
	AutoMaintenanceItemSupplies = "supplies"
)

// AutoMaintenance representa un mantenimiento realizado a un auto, medido por
// kilometraje (SPEC-124). Los costos no se persisten en esta tabla: se calculan
// a partir de las líneas de detalle.
type AutoMaintenance struct {
	ID            int64     `json:"id"`
	AutoID        int64     `json:"auto_id"`
	PerformedAt   string    `json:"performed_at"`
	Mileage       int64     `json:"mileage"`
	InstitutionID int64     `json:"institution_id"`
	Description   string    `json:"description"`
	CurrencyID    int64     `json:"currency_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AutoMaintenanceItem es una línea de mano de obra o de insumos.
type AutoMaintenanceItem struct {
	ID            int64     `json:"id"`
	MaintenanceID int64     `json:"maintenance_id"`
	ItemType      string    `json:"item_type"`
	Description   string    `json:"description"`
	Amount        float64   `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AutoMaintenanceItemInput es el payload de una línea de detalle.
type AutoMaintenanceItemInput struct {
	ItemType    string  `json:"item_type"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

// AutoMaintenanceDetail agrega los datos calculados que se muestran en la UI.
type AutoMaintenanceDetail struct {
	AutoMaintenance
	InstitutionName string                `json:"institution_name,omitempty"`
	CurrencyCode    string                `json:"currency_code"`
	LaborCost       float64               `json:"labor_cost"`
	SuppliesCost    float64               `json:"supplies_cost"`
	TotalCost       float64               `json:"total_cost"`
	Items           []AutoMaintenanceItem `json:"items"`
	Attachments     []AutoAttachment      `json:"attachments"`
}

// AutoMaintenanceInput es el payload de creación/edición.
type AutoMaintenanceInput struct {
	PerformedAt   string                     `json:"performed_at"`
	Mileage       int64                      `json:"mileage"`
	InstitutionID int64                      `json:"institution_id"`
	Description   string                     `json:"description"`
	CurrencyID    int64                      `json:"currency_id"`
	Items         []AutoMaintenanceItemInput `json:"items"`
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
