package api

import (
	"context"
	"net/http"
	"strconv"
)

// Home es el subconjunto de un hogar que necesita el MCP.
type Home struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address,omitempty"`
	Usage   string `json:"usage"`
}

// Institution es el subconjunto de una institución que necesita el MCP.
type Institution struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	CategoryID *int64 `json:"category_id,omitempty"`
}

// Bank es el subconjunto de un banco que necesita el MCP.
type Bank struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ShortCode string `json:"short_code,omitempty"`
	Country   string `json:"country,omitempty"`
	Notes     string `json:"notes,omitempty"`
	Active    bool   `json:"active"`
}

// Currency es una moneda del sistema.
type Currency struct {
	ID     int64  `json:"id"`
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol,omitempty"`
}

// InstitutionCategory es una categoría de institución.
type InstitutionCategory struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// HomeCreateRequest es el contrato de POST /api/homes.
type HomeCreateRequest struct {
	Name    string `json:"name"`
	Address string `json:"address,omitempty"`
	Usage   string `json:"usage,omitempty"`
}

// InstitutionCreateRequest es el contrato de POST /api/institutions.
type InstitutionCreateRequest struct {
	Name       string `json:"name"`
	CategoryID *int64 `json:"category_id,omitempty"`
}

// BankCreateRequest es el contrato de POST /api/banks.
type BankCreateRequest struct {
	Name      string `json:"name"`
	ShortCode string `json:"short_code,omitempty"`
	Country   string `json:"country,omitempty"`
	Notes     string `json:"notes,omitempty"`
	Active    *bool  `json:"active,omitempty"`
}

// ServiceCreateRequest es el contrato de POST /api/services.
type ServiceCreateRequest struct {
	HomeID          int64   `json:"home_id"`
	Name            string  `json:"name"`
	Institution     string  `json:"institution,omitempty"`
	CurrencyID      int64   `json:"currency_id"`
	Frequency       string  `json:"frequency,omitempty"`
	SuggestedAmount float64 `json:"suggested_amount,omitempty"`
	Active          bool    `json:"active"`
	IconKey         string  `json:"icon_key,omitempty"`
	BillingType     string  `json:"billing_type,omitempty"`
	BillingDay      *int    `json:"billing_day,omitempty"`
	AutoGenerate    bool    `json:"auto_generate,omitempty"`
	InstitutionID   *int64  `json:"institution_id,omitempty"`
	StartDate       *string `json:"start_date,omitempty"`
	EndDate         *string `json:"end_date,omitempty"`
	IsRecurring     bool    `json:"is_recurring,omitempty"`
}

// ListHomes devuelve todos los hogares.
func (c *Client) ListHomes(ctx context.Context) ([]Home, error) {
	var homes []Home
	if err := c.doJSON(ctx, http.MethodGet, "/api/homes", nil, &homes, nil); err != nil {
		return nil, err
	}
	return homes, nil
}

// CreateHome crea un hogar.
func (c *Client) CreateHome(ctx context.Context, req HomeCreateRequest) (*Home, error) {
	var home Home
	if err := c.doJSON(ctx, http.MethodPost, "/api/homes", req, &home, nil); err != nil {
		return nil, err
	}
	return &home, nil
}

// ListInstitutions devuelve todas las instituciones.
func (c *Client) ListInstitutions(ctx context.Context) ([]Institution, error) {
	var items []Institution
	if err := c.doJSON(ctx, http.MethodGet, "/api/institutions", nil, &items, nil); err != nil {
		return nil, err
	}
	return items, nil
}

// CreateInstitution crea una institución.
func (c *Client) CreateInstitution(ctx context.Context, req InstitutionCreateRequest) (*Institution, error) {
	var item Institution
	if err := c.doJSON(ctx, http.MethodPost, "/api/institutions", req, &item, nil); err != nil {
		return nil, err
	}
	return &item, nil
}

// ListBanks devuelve todos los bancos.
func (c *Client) ListBanks(ctx context.Context) ([]Bank, error) {
	var items []Bank
	if err := c.doJSON(ctx, http.MethodGet, "/api/banks", nil, &items, nil); err != nil {
		return nil, err
	}
	return items, nil
}

// CreateBank crea un banco.
func (c *Client) CreateBank(ctx context.Context, req BankCreateRequest) (*Bank, error) {
	var item Bank
	if err := c.doJSON(ctx, http.MethodPost, "/api/banks", req, &item, nil); err != nil {
		return nil, err
	}
	return &item, nil
}

// ListCurrencies devuelve las monedas del sistema.
func (c *Client) ListCurrencies(ctx context.Context) ([]Currency, error) {
	var items []Currency
	if err := c.doJSON(ctx, http.MethodGet, "/api/currencies", nil, &items, nil); err != nil {
		return nil, err
	}
	return items, nil
}

// ListInstitutionCategories devuelve las categorías de institución.
func (c *Client) ListInstitutionCategories(ctx context.Context) ([]InstitutionCategory, error) {
	var items []InstitutionCategory
	if err := c.doJSON(ctx, http.MethodGet, "/api/institution-categories", nil, &items, nil); err != nil {
		return nil, err
	}
	return items, nil
}

// ListServices devuelve todos los servicios (opcionalmente por hogar).
func (c *Client) ListServices(ctx context.Context, homeID *int64) ([]Service, error) {
	path := "/api/services"
	if homeID != nil {
		path += "?home_id=" + strconv.FormatInt(*homeID, 10)
	}
	var items []Service
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &items, nil); err != nil {
		return nil, err
	}
	return items, nil
}

// CreateService crea un servicio.
func (c *Client) CreateService(ctx context.Context, req ServiceCreateRequest) (*Service, error) {
	var item Service
	if err := c.doJSON(ctx, http.MethodPost, "/api/services", req, &item, nil); err != nil {
		return nil, err
	}
	return &item, nil
}
