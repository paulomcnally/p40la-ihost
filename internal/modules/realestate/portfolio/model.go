package portfolio

import "errors"

// ModuleID es el identificador canónico del módulo (ARCH-MODULES).
const (
	ModuleID      = "realestate.portfolio"
	ModuleVersion = "1.0.0"
)

// ErrInvalidPeriod indica un mes o año fuera de rango.
var ErrInvalidPeriod = errors.New("período inválido")

// ErrHomeNotFound indica que la vivienda no existe o fue eliminada.
var ErrHomeNotFound = errors.New("vivienda no encontrada")

// Money es un importe en una moneda.
type Money struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

// HomeSummary es el resumen mensual de una vivienda.
type HomeSummary struct {
	HomeID          int64   `json:"home_id"`
	Name            string  `json:"name"`
	Address         string  `json:"address"`
	Usage           string  `json:"usage"`
	State           string  `json:"state"`
	Income          []Money `json:"income"`
	Expenses        []Money `json:"expenses"`
	Net             []Money `json:"net"`
	PendingIncome   []Money `json:"pending_income"`
	PendingExpenses []Money `json:"pending_expenses"`
}

// Totals agrega el resumen de todas las viviendas.
type Totals struct {
	Income          []Money `json:"income"`
	Expenses        []Money `json:"expenses"`
	Net             []Money `json:"net"`
	PendingIncome   []Money `json:"pending_income"`
	PendingExpenses []Money `json:"pending_expenses"`
}

// Summary es la respuesta del resumen mensual.
type Summary struct {
	Year        int            `json:"year"`
	Month       int            `json:"month"`
	Homes       []HomeSummary  `json:"homes"`
	Totals      Totals         `json:"totals"`
	StateCounts map[string]int `json:"state_counts"`
}

// FinanceMonth es un punto de la serie de 12 meses del tab Finanzas.
type FinanceMonth struct {
	Year     int     `json:"year"`
	Month    int     `json:"month"`
	Income   []Money `json:"income"`
	Expenses []Money `json:"expenses"`
	Net      []Money `json:"net"`
}

// FinanceBreakdown es la composición de gastos por categoría.
type FinanceBreakdown struct {
	Kind    string  `json:"kind"`
	Amounts []Money `json:"amounts"`
}

// FinanceItem es un movimiento del mes seleccionado.
type FinanceItem struct {
	Type          string  `json:"type"`
	Date          string  `json:"date"`
	Description   string  `json:"description"`
	Detail        string  `json:"detail"`
	Status        string  `json:"status"`
	Method        string  `json:"method,omitempty"`
	Kind          string  `json:"kind,omitempty"`
	Area          string  `json:"area,omitempty"`
	Amounts       []Money `json:"amounts"`
	LeaseID       *int64  `json:"lease_id,omitempty"`
	ReservationID *int64  `json:"reservation_id,omitempty"`
	BillID        *int64  `json:"bill_id,omitempty"`
	ServiceID     *int64  `json:"service_id,omitempty"`
	MaintenanceID *int64  `json:"maintenance_id,omitempty"`
}

// HomeFinances es la respuesta del tab Finanzas de una vivienda.
type HomeFinances struct {
	HomeID          int64              `json:"home_id"`
	HomeName        string             `json:"home_name"`
	Address         string             `json:"address"`
	Usage           string             `json:"usage"`
	State           string             `json:"state"`
	Year            int                `json:"year"`
	Month           int                `json:"month"`
	Currencies      []string           `json:"currencies"`
	Months          []FinanceMonth     `json:"months"`
	Income          []Money            `json:"income"`
	Expenses        []Money            `json:"expenses"`
	Net             []Money            `json:"net"`
	PendingIncome   []Money            `json:"pending_income"`
	PendingExpenses []Money            `json:"pending_expenses"`
	Breakdown       []FinanceBreakdown `json:"breakdown"`
	Items           []FinanceItem      `json:"items"`
}

// period contiene el rango [Start, End) del mes.
type period struct {
	Year  int
	Month int
	Start string
	End   string
}
