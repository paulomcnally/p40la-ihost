package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// BudgetAPI es el contrato de presupuesto que consumen las tools (SPEC-163).
// Permite usar una API falsa en tests sin arrastrar el resto de los dominios.
type BudgetAPI interface {
	ListCategoryGroups(ctx context.Context) ([]CategoryGroup, error)
	ListAccounts(ctx context.Context) ([]Account, error)
	ListTransactions(ctx context.Context, year, month int) ([]Transaction, error)
	MonthView(ctx context.Context, year, month int) (*MonthView, error)
	CreateTransaction(ctx context.Context, req TransactionRequest) (*Transaction, error)
}

// CategoryGroup es un grupo de categorías del presupuesto con sus categorías
// activas (GET /api/budget/category-groups).
type CategoryGroup struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Icon       string     `json:"icon"`
	SortOrder  int        `json:"sort_order"`
	Categories []Category `json:"categories"`
}

// Category es una categoría del presupuesto. AccountID es la cuenta vinculada
// (opcional); SPEC-097 la usa para derivar la cuenta de una transacción.
type Category struct {
	ID              int64    `json:"id"`
	CategoryGroupID int64    `json:"category_group_id"`
	Name            string   `json:"name"`
	Icon            string   `json:"icon"`
	SortOrder       int      `json:"sort_order"`
	TargetAmount    *float64 `json:"target_amount,omitempty"`
	TargetType      string   `json:"target_type,omitempty"`
	TargetDate      string   `json:"target_date,omitempty"`
	AccountID       *int64   `json:"account_id,omitempty"`
	ServiceIDs      []int64  `json:"service_ids,omitempty"`
	ServiceNames    []string `json:"service_names,omitempty"`
}

// Account es una cuenta del presupuesto (tabla accounts, SPEC-093), distinta de
// la cuenta bancaria de bank_accounts. Balance es calculado por el backend.
type Account struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	CurrencyID      int64   `json:"currency_id"`
	CurrencyCode    string  `json:"currency_code,omitempty"`
	StartingBalance float64 `json:"starting_balance"`
	Balance         float64 `json:"balance"`
}

// Transaction es un movimiento del presupuesto. El mes no es un campo: se
// deriva de Date (ADR-001).
type Transaction struct {
	ID           int64   `json:"id"`
	AccountID    int64   `json:"account_id"`
	AccountName  string  `json:"account_name,omitempty"`
	CategoryID   *int64  `json:"category_id,omitempty"`
	CategoryName string  `json:"category_name,omitempty"`
	CurrencyID   int64   `json:"currency_id"`
	CurrencyCode string  `json:"currency_code,omitempty"`
	Date         string  `json:"date"`
	Payee        string  `json:"payee"`
	Memo         string  `json:"memo"`
	Outflow      float64 `json:"outflow"`
	Inflow       float64 `json:"inflow"`
	Cleared      bool    `json:"cleared"`
	SourceBillID *int64  `json:"source_bill_id,omitempty"`
}

// TransactionRequest es el contrato de POST /api/budget/transactions.
type TransactionRequest struct {
	AccountID  int64   `json:"account_id"`
	CategoryID *int64  `json:"category_id"`
	CurrencyID int64   `json:"currency_id"`
	Date       string  `json:"date"`
	Payee      string  `json:"payee"`
	Memo       string  `json:"memo"`
	Outflow    float64 `json:"outflow"`
	Inflow     float64 `json:"inflow"`
	Cleared    bool    `json:"cleared"`
}

// MonthView es la vista mensual del presupuesto (GET
// /api/budget/months/{year}/{month}).
type MonthView struct {
	Year           int                  `json:"year"`
	Month          int                  `json:"month"`
	CurrencyTotals []MonthCurrencyTotal `json:"currency_totals"`
	Groups         []MonthGroup         `json:"groups"`
}

// MonthCurrencyTotal es un total agregado por moneda.
type MonthCurrencyTotal struct {
	CurrencyID    int64   `json:"currency_id"`
	Code          string  `json:"code"`
	Symbol        string  `json:"symbol"`
	TotalIncome   float64 `json:"total_income"`
	TotalAssigned float64 `json:"total_assigned"`
	Unassigned    float64 `json:"unassigned"`
}

// MonthGroup agrupa las filas de la vista mensual por grupo de categorías.
type MonthGroup struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	Icon       string          `json:"icon"`
	SortOrder  int             `json:"sort_order"`
	Categories []MonthCategory `json:"categories"`
}

// MonthCategory es una categoría de la vista mensual. Assigned, Activity y
// Available son mapas currency_id → monto.
type MonthCategory struct {
	Category
	Assigned      map[int64]float64 `json:"assigned"`
	Activity      map[int64]float64 `json:"activity"`
	Available     map[int64]float64 `json:"available"`
	RecurringRule any               `json:"recurring_rule,omitempty"`
}

// ListCategoryGroups devuelve los grupos con sus categorías activas.
func (c *Client) ListCategoryGroups(ctx context.Context) ([]CategoryGroup, error) {
	var groups []CategoryGroup
	if err := c.doJSON(ctx, http.MethodGet, "/api/budget/category-groups", nil, &groups, nil); err != nil {
		return nil, err
	}
	return groups, nil
}

// ListAccounts devuelve las cuentas activas del presupuesto.
func (c *Client) ListAccounts(ctx context.Context) ([]Account, error) {
	var accounts []Account
	if err := c.doJSON(ctx, http.MethodGet, "/api/budget/accounts", nil, &accounts, nil); err != nil {
		return nil, err
	}
	return accounts, nil
}

// ListTransactions devuelve las transacciones de un mes.
func (c *Client) ListTransactions(ctx context.Context, year, month int) ([]Transaction, error) {
	var transactions []Transaction
	query := url.Values{}
	query.Set("year", fmt.Sprintf("%d", year))
	query.Set("month", fmt.Sprintf("%d", month))
	path := "/api/budget/transactions?" + query.Encode()
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &transactions, nil); err != nil {
		return nil, err
	}
	return transactions, nil
}

// MonthView devuelve la vista mensual completa del presupuesto.
func (c *Client) MonthView(ctx context.Context, year, month int) (*MonthView, error) {
	var view MonthView
	path := fmt.Sprintf("/api/budget/months/%d/%d", year, month)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &view, nil); err != nil {
		return nil, err
	}
	return &view, nil
}

// CreateTransaction registra un movimiento del presupuesto. La escritura se
// marca con origen mcp.
func (c *Client) CreateTransaction(ctx context.Context, req TransactionRequest) (*Transaction, error) {
	var transaction Transaction
	if err := c.doJSON(ctx, http.MethodPost, "/api/budget/transactions", req, &transaction, sourceHeaders()); err != nil {
		return nil, err
	}
	return &transaction, nil
}
