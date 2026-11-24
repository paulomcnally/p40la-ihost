package api

import (
	"context"
	"fmt"
)

// AutosAPI es el contrato de autos que consumen las tools. Permite usar una
// API falsa en tests sin arrastrar los métodos de facturas.
type AutosAPI interface {
	ListAutos(ctx context.Context) ([]Auto, error)
	GetAuto(ctx context.Context, id int64) (*Auto, error)
	CreateAuto(ctx context.Context, req AutoRequest) (*Auto, error)
	UpdateAuto(ctx context.Context, id int64, req AutoRequest) (*Auto, error)
	ListAvailableServices(ctx context.Context, autoID int64) ([]AvailableService, error)
	CreateMaintenance(ctx context.Context, autoID int64, req MaintenanceRequest) (*Maintenance, error)
	GetMaintenance(ctx context.Context, autoID, maintenanceID int64) (*Maintenance, error)
	CreateMaintenanceItem(ctx context.Context, autoID, maintenanceID int64, req MaintenanceItemRequest) (*MaintenanceItem, error)
	CreateInspection(ctx context.Context, autoID int64, req InspectionRequest) (*Inspection, error)
	ListInspections(ctx context.Context, autoID int64) (*InspectionList, error)
	ListInsurances(ctx context.Context, autoID int64) ([]InsuranceDetail, error)
	CreateInsurance(ctx context.Context, autoID int64, req InsuranceRequest) (*Insurance, error)
}

// FullAPI es la unión de contratos que monta el servidor MCP.
type FullAPI interface {
	API
	AutosAPI
	ReadAPI
}

// Auto es el subconjunto de un vehículo que necesita el MCP.
type Auto struct {
	ID     int64  `json:"id"`
	Year   int64  `json:"year"`
	Model  string `json:"model"`
	Brand  string `json:"brand"`
	Color  string `json:"color"`
	Icon   string `json:"icon"`
	Motor  string `json:"motor"`
	Chasis string `json:"chasis"`
	VIN    string `json:"vin"`
	Placa  string `json:"placa"`
}

// AutoRequest es el contrato de POST /api/autos y PUT /api/autos/{id}.
type AutoRequest struct {
	Year   int64  `json:"year"`
	Model  string `json:"model"`
	Brand  string `json:"brand"`
	Color  string `json:"color"`
	Icon   string `json:"icon"`
	Motor  string `json:"motor"`
	Chasis string `json:"chasis"`
	VIN    string `json:"vin"`
	Placa  string `json:"placa"`
}

// MaintenanceRequest es el contrato de POST /api/autos/{id}/maintenances.
type MaintenanceRequest struct {
	PerformedAt   string `json:"performed_at"`
	Mileage       int64  `json:"mileage"`
	InstitutionID int64  `json:"institution_id"`
	Description   string `json:"description"`
	CurrencyID    int64  `json:"currency_id"`
}

// MaintenanceItemRequest es el contrato de POST
// /api/autos/{id}/maintenances/{maintenance_id}/items.
type MaintenanceItemRequest struct {
	ItemType    string  `json:"item_type"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

// MaintenanceItem es una línea de mano de obra o insumos.
type MaintenanceItem struct {
	ID            int64   `json:"id"`
	MaintenanceID int64   `json:"maintenance_id"`
	ItemType      string  `json:"item_type"`
	Description   string  `json:"description"`
	Amount        float64 `json:"amount"`
}

// Maintenance es el detalle devuelto por la API al crear un mantenimiento.
type Maintenance struct {
	ID              int64             `json:"id"`
	AutoID          int64             `json:"auto_id"`
	PerformedAt     string            `json:"performed_at"`
	Mileage         int64             `json:"mileage"`
	InstitutionID   int64             `json:"institution_id"`
	InstitutionName string            `json:"institution_name,omitempty"`
	Description     string            `json:"description"`
	CurrencyID      int64             `json:"currency_id"`
	CurrencyCode    string            `json:"currency_code,omitempty"`
	LaborCost       float64           `json:"labor_cost"`
	SuppliesCost    float64           `json:"supplies_cost"`
	TotalCost       float64           `json:"total_cost"`
	Items           []MaintenanceItem `json:"items"`
}

// InspectionRequest es el contrato de POST /api/autos/{id}/inspections.
type InspectionRequest struct {
	PerformedAt    string `json:"performed_at"`
	InspectionYear int64  `json:"inspection_year"`
	InstitutionID  int64  `json:"institution_id"`
	Result         string `json:"result"`
	Notes          string `json:"notes"`
}

// Inspection es una inspección mecánica registrada.
type Inspection struct {
	ID              int64  `json:"id"`
	AutoID          int64  `json:"auto_id"`
	PerformedAt     string `json:"performed_at"`
	InspectionYear  int64  `json:"inspection_year"`
	InstitutionID   int64  `json:"institution_id"`
	InstitutionName string `json:"institution_name,omitempty"`
	Result          string `json:"result"`
	Notes           string `json:"notes"`
}

// InspectionSchedule es el cronograma calculado por el backend.
type InspectionSchedule struct {
	FirstDueYear  int64   `json:"first_due_year"`
	NextDueYear   int64   `json:"next_due_year"`
	NextDueDate   string  `json:"next_due_date"`
	Status        string  `json:"status"`
	DaysUntilDue  int64   `json:"days_until_due"`
	ApprovedYears []int64 `json:"approved_years"`
	RejectedYears []int64 `json:"rejected_years"`
}

// InspectionList es la respuesta de GET /api/autos/{id}/inspections.
type InspectionList struct {
	Items    []Inspection       `json:"items"`
	Schedule InspectionSchedule `json:"schedule"`
}

// InsuranceRequest es el contrato de POST /api/autos/{id}/services.
type InsuranceRequest struct {
	ServiceID     int64  `json:"service_id"`
	CoverageType  string `json:"coverage_type"`
	PolicyNumber  string `json:"policy_number"`
	Certificate   string `json:"certificate"`
	InsurerNumber string `json:"insurer_number"`
}

// Insurance es la asociación entre un auto y un servicio de seguro.
type Insurance struct {
	ID            int64   `json:"id"`
	AutoID        int64   `json:"auto_id"`
	ServiceID     int64   `json:"service_id"`
	CoverageType  string  `json:"coverage_type"`
	PolicyNumber  string  `json:"policy_number"`
	Certificate   *string `json:"certificate,omitempty"`
	InsurerNumber string  `json:"insurer_number"`
}

// InsuranceDetail agrega los datos del servicio a la asociación.
type InsuranceDetail struct {
	Insurance
	ServiceName     string  `json:"service_name"`
	InstitutionName string  `json:"institution_name"`
	CurrencyID      int64   `json:"currency_id"`
	SuggestedAmount float64 `json:"suggested_amount"`
	Frequency       string  `json:"frequency"`
	Active          bool    `json:"active"`
	StartDate       *string `json:"start_date,omitempty"`
	EndDate         *string `json:"end_date,omitempty"`
}

// AvailableService es el subconjunto de un servicio que puede asociarse como
// seguro a un auto.
type AvailableService struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Institution     string  `json:"institution,omitempty"`
	CurrencyID      int64   `json:"currency_id"`
	SuggestedAmount float64 `json:"suggested_amount"`
	Frequency       string  `json:"frequency"`
	Active          bool    `json:"active"`
	IsInsurance     bool    `json:"is_insurance"`
}

// ListAutos devuelve todos los autos.
func (c *Client) ListAutos(ctx context.Context) ([]Auto, error) {
	var autos []Auto
	if err := c.doJSON(ctx, "GET", "/api/autos", nil, &autos, nil); err != nil {
		return nil, err
	}
	return autos, nil
}

// GetAuto devuelve un auto por ID.
func (c *Client) GetAuto(ctx context.Context, id int64) (*Auto, error) {
	var auto Auto
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/autos/%d", id), nil, &auto, nil); err != nil {
		return nil, err
	}
	return &auto, nil
}

// CreateAuto crea un auto.
func (c *Client) CreateAuto(ctx context.Context, req AutoRequest) (*Auto, error) {
	var auto Auto
	if err := c.doJSON(ctx, "POST", "/api/autos", req, &auto, sourceHeaders()); err != nil {
		return nil, err
	}
	return &auto, nil
}

// UpdateAuto actualiza un auto existente.
func (c *Client) UpdateAuto(ctx context.Context, id int64, req AutoRequest) (*Auto, error) {
	var auto Auto
	if err := c.doJSON(ctx, "PUT", fmt.Sprintf("/api/autos/%d", id), req, &auto, sourceHeaders()); err != nil {
		return nil, err
	}
	return &auto, nil
}

// ListAvailableServices devuelve los servicios que pueden asociarse al auto.
func (c *Client) ListAvailableServices(ctx context.Context, autoID int64) ([]AvailableService, error) {
	var services []AvailableService
	if err := c.doJSON(ctx, "GET", fmt.Sprintf("/api/autos/%d/available-services", autoID), nil, &services, nil); err != nil {
		return nil, err
	}
	return services, nil
}

// CreateMaintenance registra un mantenimiento.
func (c *Client) CreateMaintenance(ctx context.Context, autoID int64, req MaintenanceRequest) (*Maintenance, error) {
	var maintenance Maintenance
	path := fmt.Sprintf("/api/autos/%d/maintenances", autoID)
	if err := c.doJSON(ctx, "POST", path, req, &maintenance, sourceHeaders()); err != nil {
		return nil, err
	}
	return &maintenance, nil
}

// GetMaintenance devuelve el detalle de un mantenimiento.
func (c *Client) GetMaintenance(ctx context.Context, autoID, maintenanceID int64) (*Maintenance, error) {
	var maintenance Maintenance
	path := fmt.Sprintf("/api/autos/%d/maintenances/%d", autoID, maintenanceID)
	if err := c.doJSON(ctx, "GET", path, nil, &maintenance, nil); err != nil {
		return nil, err
	}
	return &maintenance, nil
}

// CreateMaintenanceItem agrega una línea de mano de obra o insumos.
func (c *Client) CreateMaintenanceItem(ctx context.Context, autoID, maintenanceID int64, req MaintenanceItemRequest) (*MaintenanceItem, error) {
	var item MaintenanceItem
	path := fmt.Sprintf("/api/autos/%d/maintenances/%d/items", autoID, maintenanceID)
	if err := c.doJSON(ctx, "POST", path, req, &item, sourceHeaders()); err != nil {
		return nil, err
	}
	return &item, nil
}

// CreateInspection registra una inspección mecánica.
func (c *Client) CreateInspection(ctx context.Context, autoID int64, req InspectionRequest) (*Inspection, error) {
	var inspection Inspection
	path := fmt.Sprintf("/api/autos/%d/inspections", autoID)
	if err := c.doJSON(ctx, "POST", path, req, &inspection, sourceHeaders()); err != nil {
		return nil, err
	}
	return &inspection, nil
}

// ListInspections devuelve las inspecciones y el cronograma de un auto.
func (c *Client) ListInspections(ctx context.Context, autoID int64) (*InspectionList, error) {
	var list InspectionList
	path := fmt.Sprintf("/api/autos/%d/inspections", autoID)
	if err := c.doJSON(ctx, "GET", path, nil, &list, nil); err != nil {
		return nil, err
	}
	return &list, nil
}

// ListInsurances devuelve los seguros asociados a un auto.
func (c *Client) ListInsurances(ctx context.Context, autoID int64) ([]InsuranceDetail, error) {
	var insurances []InsuranceDetail
	path := fmt.Sprintf("/api/autos/%d/services", autoID)
	if err := c.doJSON(ctx, "GET", path, nil, &insurances, nil); err != nil {
		return nil, err
	}
	return insurances, nil
}

// CreateInsurance asocia un servicio como seguro a un auto.
func (c *Client) CreateInsurance(ctx context.Context, autoID int64, req InsuranceRequest) (*Insurance, error) {
	var insurance Insurance
	path := fmt.Sprintf("/api/autos/%d/services", autoID)
	if err := c.doJSON(ctx, "POST", path, req, &insurance, sourceHeaders()); err != nil {
		return nil, err
	}
	return &insurance, nil
}

func sourceHeaders() map[string]string {
	return map[string]string{sourceHeader: sourceMCP}
}
