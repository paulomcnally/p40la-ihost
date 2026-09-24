package models

import "time"

// AutoService representa la asociación entre un auto y un servicio (seguro).
type AutoService struct {
	ID            int64     `json:"id"`
	AutoID        int64     `json:"auto_id"`
	ServiceID     int64     `json:"service_id"`
	CoverageType  string    `json:"coverage_type"`
	PolicyNumber  string    `json:"policy_number"`
	Certificate   *string   `json:"certificate,omitempty"`
	InsurerNumber string    `json:"insurer_number"`
	CreatedAt     time.Time `json:"created_at"`
}

// AutoServiceDetail incluye info del servicio e institución para la API.
type AutoServiceDetail struct {
	ID              int64   `json:"id"`
	AutoID          int64   `json:"auto_id"`
	ServiceID       int64   `json:"service_id"`
	CoverageType    string  `json:"coverage_type"`
	PolicyNumber    string  `json:"policy_number"`
	Certificate     *string `json:"certificate,omitempty"`
	InsurerNumber   string  `json:"insurer_number"`
	ServiceName     string  `json:"service_name"`
	InstitutionName string  `json:"institution_name"`
	InstitutionID   *int64  `json:"institution_id,omitempty"`
	CurrencyID      int64   `json:"currency_id"`
	SuggestedAmount float64 `json:"suggested_amount"`
	Frequency       string  `json:"frequency"`
	IconKey         string  `json:"icon_key"`
	Active          bool    `json:"active"`
	StartDate       *string `json:"start_date,omitempty"`
	EndDate         *string `json:"end_date,omitempty"`
	IsRecurring     bool    `json:"is_recurring"`
	CreatedAt       string  `json:"created_at"`
}

// ServiceAutoDetail incluye info del auto asociado a un servicio (SPEC-091).
// Es el reverse lookup de AutoServiceDetail: se consulta desde service_id.
type ServiceAutoDetail struct {
	AutoID        int64   `json:"auto_id"`
	Brand         string  `json:"brand"`
	Model         string  `json:"model"`
	Year          int64   `json:"year"`
	Color         string  `json:"color"`
	Icon          string  `json:"icon"`
	Placa         string  `json:"placa"`
	CoverageType  string  `json:"coverage_type"`
	PolicyNumber  string  `json:"policy_number"`
	Certificate   *string `json:"certificate,omitempty"`
	InsurerNumber string  `json:"insurer_number"`
	CreatedAt     string  `json:"created_at"`
}
