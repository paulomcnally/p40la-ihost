package api

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// HomeMaintenanceAPI es el contrato del módulo home.maintenance que consumen
// las tools MCP (lectura y escritura).
type HomeMaintenanceAPI interface {
	ListHomeMaintenances(ctx context.Context, homeID int64, filter HomeMaintenanceFilter) (*HomeMaintenanceList, error)
	GetHomeMaintenance(ctx context.Context, homeID, maintenanceID int64) (*HomeMaintenance, error)
	CreateHomeMaintenance(ctx context.Context, homeID int64, req HomeMaintenanceRequest) (*HomeMaintenance, error)
	UpdateHomeMaintenance(ctx context.Context, homeID, maintenanceID int64, req HomeMaintenanceUpdateRequest) (*HomeMaintenance, error)
	CreateHomeMaintenanceItem(ctx context.Context, homeID, maintenanceID int64, req HomeMaintenanceItemRequest) (*HomeMaintenanceItem, error)
	UpdateHomeMaintenanceItem(ctx context.Context, homeID, maintenanceID, itemID int64, req HomeMaintenanceItemRequest) (*HomeMaintenanceItem, error)
	DeleteHomeMaintenanceItem(ctx context.Context, homeID, maintenanceID, itemID int64) error
	MoveHomeMaintenanceItems(ctx context.Context, homeID, maintenanceID int64, req HomeMaintenanceMoveRequest) (*HomeMaintenanceMoveResult, error)
}

// HomeMaintenanceFilter acota el histórico de una vivienda.
type HomeMaintenanceFilter struct {
	Year         int
	Kind         string
	Date         string
	ItemType     string
	Query        string
	IncludeItems bool
}

// HomeMaintenance es una reparación o mantenimiento con sus líneas.
type HomeMaintenance struct {
	ID               int64                          `json:"id"`
	HomeID           int64                          `json:"home_id"`
	PerformedAt      string                         `json:"performed_at"`
	Kind             string                         `json:"kind"`
	Area             string                         `json:"area"`
	InstitutionID    *int64                         `json:"institution_id"`
	InstitutionName  string                         `json:"institution_name"`
	Description      string                         `json:"description"`
	TotalsByCurrency []HomeMaintenanceCurrencyTotal `json:"totals_by_currency"`
	Items            []HomeMaintenanceItem          `json:"items"`
	CreatedAt        time.Time                      `json:"created_at"`
	UpdatedAt        time.Time                      `json:"updated_at"`
}

// HomeMaintenanceItem es una línea de mano de obra o insumos.
type HomeMaintenanceItem struct {
	ID              int64     `json:"id"`
	MaintenanceID   int64     `json:"maintenance_id"`
	ItemType        string    `json:"item_type"`
	Description     string    `json:"description"`
	Quantity        float64   `json:"quantity"`
	Unit            string    `json:"unit"`
	Amount          float64   `json:"amount"`
	CurrencyID      int64     `json:"currency_id"`
	CurrencyCode    string    `json:"currency_code"`
	InstitutionID   *int64    `json:"institution_id"`
	InstitutionName string    `json:"institution_name"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// HomeMaintenanceCurrencyTotal es el total de una moneda con desglose.
type HomeMaintenanceCurrencyTotal struct {
	CurrencyID   int64   `json:"currency_id"`
	CurrencyCode string  `json:"currency_code"`
	LaborCost    float64 `json:"labor_cost"`
	SuppliesCost float64 `json:"supplies_cost"`
	Total        float64 `json:"total"`
}

// HomeMaintenanceList es la respuesta de GET /api/homes/{id}/maintenances.
type HomeMaintenanceList struct {
	Items            []HomeMaintenance              `json:"items"`
	TotalsByCurrency []HomeMaintenanceCurrencyTotal `json:"totals_by_currency"`
	Truncated        bool                           `json:"truncated"`
}

// HomeMaintenanceRequest es el cuerpo de POST /api/homes/{id}/maintenances.
// Item es opcional: si viene, el backend crea mantenimiento + línea en una
// transacción.
type HomeMaintenanceRequest struct {
	PerformedAt   string                      `json:"performed_at"`
	Kind          string                      `json:"kind,omitempty"`
	Area          string                      `json:"area,omitempty"`
	InstitutionID *int64                      `json:"institution_id,omitempty"`
	Description   string                      `json:"description"`
	Item          *HomeMaintenanceItemRequest `json:"item,omitempty"`
}

// HomeMaintenanceUpdateRequest es el cuerpo de PUT /api/homes/{id}/maintenances/{mid}.
// El backend reemplaza la cabecera completa: la tool debe enviar los cinco campos.
type HomeMaintenanceUpdateRequest struct {
	PerformedAt   string `json:"performed_at"`
	Kind          string `json:"kind"`
	Area          string `json:"area"`
	InstitutionID *int64 `json:"institution_id"`
	Description   string `json:"description"`
}

// HomeMaintenanceItemRequest es el cuerpo de POST .../items y la línea
// opcional del alta combinada.
type HomeMaintenanceItemRequest struct {
	ItemType      string  `json:"item_type"`
	Description   string  `json:"description"`
	Quantity      float64 `json:"quantity,omitempty"`
	Unit          string  `json:"unit,omitempty"`
	Amount        float64 `json:"amount"`
	CurrencyID    int64   `json:"currency_id"`
	InstitutionID *int64  `json:"institution_id,omitempty"`
}

// HomeMaintenanceTargetRequest es el destino nuevo de un movimiento.
type HomeMaintenanceTargetRequest struct {
	PerformedAt   string `json:"performed_at"`
	Kind          string `json:"kind,omitempty"`
	Area          string `json:"area,omitempty"`
	InstitutionID *int64 `json:"institution_id,omitempty"`
	Description   string `json:"description"`
}

// HomeMaintenanceMoveRequest es el cuerpo de POST .../items/move. Exactamente
// uno de TargetMaintenanceID o Target debe venir informado.
type HomeMaintenanceMoveRequest struct {
	ItemIDs             []int64                       `json:"item_ids"`
	ItemType            string                        `json:"item_type,omitempty"`
	TargetMaintenanceID *int64                        `json:"target_maintenance_id,omitempty"`
	Target              *HomeMaintenanceTargetRequest `json:"target,omitempty"`
	DeleteSourceIfEmpty bool                          `json:"delete_source_if_empty,omitempty"`
}

// HomeMaintenanceMoveResult describe el movimiento de líneas.
type HomeMaintenanceMoveResult struct {
	SourceMaintenanceID int64                 `json:"source_maintenance_id"`
	TargetMaintenanceID int64                 `json:"target_maintenance_id"`
	TargetCreated       bool                  `json:"target_created"`
	Target              *HomeMaintenance      `json:"target"`
	MovedCount          int                   `json:"moved_count"`
	MovedItems          []HomeMaintenanceItem `json:"moved_items"`
	SourceItemCount     int                   `json:"source_item_count"`
	SourceDeleted       bool                  `json:"source_deleted"`
}

// ListHomeMaintenances devuelve el histórico de una vivienda con filtros.
func (c *Client) ListHomeMaintenances(ctx context.Context, homeID int64, filter HomeMaintenanceFilter) (*HomeMaintenanceList, error) {
	query := url.Values{}
	if filter.Year > 0 {
		query.Set("year", fmt.Sprintf("%d", filter.Year))
	}
	if filter.Kind != "" {
		query.Set("kind", filter.Kind)
	}
	if filter.Date != "" {
		query.Set("date", filter.Date)
	}
	if filter.ItemType != "" {
		query.Set("item_type", filter.ItemType)
	}
	if filter.Query != "" {
		query.Set("q", filter.Query)
	}
	if filter.IncludeItems {
		query.Set("include_items", "1")
	}
	path := fmt.Sprintf("/api/homes/%d/maintenances", homeID)
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var list HomeMaintenanceList
	if err := c.doJSON(ctx, "GET", path, nil, &list, nil); err != nil {
		return nil, err
	}
	return &list, nil
}

// GetHomeMaintenance devuelve una reparación con sus líneas.
func (c *Client) GetHomeMaintenance(ctx context.Context, homeID, maintenanceID int64) (*HomeMaintenance, error) {
	var maintenance HomeMaintenance
	path := fmt.Sprintf("/api/homes/%d/maintenances/%d", homeID, maintenanceID)
	if err := c.doJSON(ctx, "GET", path, nil, &maintenance, nil); err != nil {
		return nil, err
	}
	return &maintenance, nil
}

// CreateHomeMaintenance registra una reparación y, si req.Item no es nil, su
// primera línea en una transacción.
func (c *Client) CreateHomeMaintenance(ctx context.Context, homeID int64, req HomeMaintenanceRequest) (*HomeMaintenance, error) {
	var maintenance HomeMaintenance
	path := fmt.Sprintf("/api/homes/%d/maintenances", homeID)
	if err := c.doJSON(ctx, "POST", path, req, &maintenance, sourceHeaders()); err != nil {
		return nil, err
	}
	return &maintenance, nil
}

// UpdateHomeMaintenance reemplaza la cabecera de una reparación existente.
func (c *Client) UpdateHomeMaintenance(ctx context.Context, homeID, maintenanceID int64, req HomeMaintenanceUpdateRequest) (*HomeMaintenance, error) {
	var maintenance HomeMaintenance
	path := fmt.Sprintf("/api/homes/%d/maintenances/%d", homeID, maintenanceID)
	if err := c.doJSON(ctx, "PUT", path, req, &maintenance, sourceHeaders()); err != nil {
		return nil, err
	}
	return &maintenance, nil
}

// CreateHomeMaintenanceItem agrega una línea a una reparación existente.
func (c *Client) CreateHomeMaintenanceItem(ctx context.Context, homeID, maintenanceID int64, req HomeMaintenanceItemRequest) (*HomeMaintenanceItem, error) {
	var item HomeMaintenanceItem
	path := fmt.Sprintf("/api/homes/%d/maintenances/%d/items", homeID, maintenanceID)
	if err := c.doJSON(ctx, "POST", path, req, &item, sourceHeaders()); err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateHomeMaintenanceItem reemplaza una línea de una reparación existente.
func (c *Client) UpdateHomeMaintenanceItem(ctx context.Context, homeID, maintenanceID, itemID int64, req HomeMaintenanceItemRequest) (*HomeMaintenanceItem, error) {
	var item HomeMaintenanceItem
	path := fmt.Sprintf("/api/homes/%d/maintenances/%d/items/%d", homeID, maintenanceID, itemID)
	if err := c.doJSON(ctx, "PUT", path, req, &item, sourceHeaders()); err != nil {
		return nil, err
	}
	return &item, nil
}

// DeleteHomeMaintenanceItem borra una línea; el backend responde 204.
func (c *Client) DeleteHomeMaintenanceItem(ctx context.Context, homeID, maintenanceID, itemID int64) error {
	path := fmt.Sprintf("/api/homes/%d/maintenances/%d/items/%d", homeID, maintenanceID, itemID)
	return c.doJSON(ctx, "DELETE", path, nil, nil, sourceHeaders())
}

// MoveHomeMaintenanceItems mueve líneas a otra reparación del mismo hogar,
// creando el destino si req.Target viene informado.
func (c *Client) MoveHomeMaintenanceItems(ctx context.Context, homeID, maintenanceID int64, req HomeMaintenanceMoveRequest) (*HomeMaintenanceMoveResult, error) {
	var result HomeMaintenanceMoveResult
	path := fmt.Sprintf("/api/homes/%d/maintenances/%d/items/move", homeID, maintenanceID)
	if err := c.doJSON(ctx, "POST", path, req, &result, sourceHeaders()); err != nil {
		return nil, err
	}
	return &result, nil
}
