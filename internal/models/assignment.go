package models

import "time"

// Assignment representa un monto asignado a una categoría en un mes.
// SPEC-093. Única por (budget_month_id, category_id, currency_id).
// source: 'one_time' (puntual) o 'recurring' (derivado de una regla).
type Assignment struct {
	ID              int64     `json:"id"`
	BudgetMonthID   int64     `json:"budget_month_id"`
	CategoryID      int64     `json:"category_id"`
	CurrencyID      int64     `json:"currency_id"`
	CurrencyCode    string    `json:"currency_code,omitempty"`
	Amount          float64   `json:"amount"`
	Source          string    `json:"source"`
	RecurringRuleID int64     `json:"recurring_rule_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}