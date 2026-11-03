package models

import "time"

// ServiceBillDetail es una factura de servicio con el contexto necesario para
// el home global de Servicios (SPEC-120): servicio, casa, moneda, período
// efectivo y fecha efectiva de vencimiento.
type ServiceBillDetail struct {
	ID               int64      `json:"id"`
	ServiceID        int64      `json:"service_id"`
	ServiceName      string     `json:"service_name"`
	Institution      string     `json:"institution"`
	HomeID           int64      `json:"home_id"`
	HomeName         string     `json:"home_name"`
	IconKey          string     `json:"icon_key"`
	CurrencyID       int64      `json:"currency_id"`
	CurrencyCode     string     `json:"currency_code"`
	CurrencySymbol   string     `json:"currency_symbol"`
	Frequency        string     `json:"frequency"`
	BillingDay       *int       `json:"billing_day"`
	Active           bool       `json:"active"`
	Year             int        `json:"year"`
	Month            int        `json:"month"`
	PeriodYear       int        `json:"period_year"`
	PeriodMonth      int        `json:"period_month"`
	PeriodDate       string     `json:"period_date"`
	DateEstimated    bool       `json:"date_estimated"`
	Amount           float64    `json:"amount"`
	InvoiceNumber    string     `json:"invoice_number,omitempty"`
	Status           string     `json:"status"`
	DriveURL         string     `json:"drive_url,omitempty"`
	IssueDate        *string    `json:"issue_date,omitempty"`
	DueDate          *string    `json:"due_date,omitempty"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
	PaymentReference string     `json:"payment_reference,omitempty"`
}

// ServiceBillSummaryRow es una fila agregada por período y moneda para el
// histórico de la tab Gráfica (SPEC-120).
type ServiceBillSummaryRow struct {
	Period         string  `json:"period"`
	Year           int     `json:"year"`
	Month          int     `json:"month"`
	CurrencyCode   string  `json:"currency_code"`
	CurrencySymbol string  `json:"currency_symbol"`
	Total          float64 `json:"total"`
	Paid           float64 `json:"paid"`
	Pending        float64 `json:"pending"`
	Count          int     `json:"count"`
}

// ServiceBillSummaryResponse es la respuesta de /api/service-bills/summary.
type ServiceBillSummaryResponse struct {
	From    string                  `json:"from"`
	To      string                  `json:"to"`
	Periods []ServiceBillSummaryRow `json:"periods"`
}

// ProjectionBillRow es una factura existente usada para calcular la
// proyección (SPEC-120): confirmado + estimado.
type ProjectionBillRow struct {
	BillID         int64
	ServiceID      int64
	ServiceName    string
	Year           int
	Month          int
	DueDate        *string
	Amount         float64
	Status         string
	CurrencyCode   string
	CurrencySymbol string
}

// ProjectionServiceRow es un servicio recurrente activo usado para estimar
// facturas futuras sin factura existente.
type ProjectionServiceRow struct {
	ID              int64
	Name            string
	HomeID          int64
	CurrencyID      int64
	CurrencyCode    string
	CurrencySymbol  string
	Frequency       string
	SuggestedAmount float64
	StartDate       *string
	EndDate         *string
}

// ServiceProjectionItem detalla el origen de un monto proyectado.
type ServiceProjectionItem struct {
	ServiceID    int64   `json:"service_id"`
	Name         string  `json:"name"`
	Amount       float64 `json:"amount"`
	Source       string  `json:"source"` // bill | estimate
	CurrencyCode string  `json:"currency_code"`
}

// ServiceProjectionPeriod es un período de la proyección para una moneda.
type ServiceProjectionPeriod struct {
	Period         string                  `json:"period"`
	Year           int                     `json:"year"`
	Month          int                     `json:"month"`
	CurrencyCode   string                  `json:"currency_code"`
	CurrencySymbol string                  `json:"currency_symbol"`
	Confirmed      float64                 `json:"confirmed"`
	Estimated      float64                 `json:"estimated"`
	Total          float64                 `json:"total"`
	Services       []ServiceProjectionItem `json:"services"`
}

// ServiceProjectionResponse es la respuesta de /api/service-bills/projection.
type ServiceProjectionResponse struct {
	From         string                    `json:"from"`
	Months       int                       `json:"months"`
	Periods      []ServiceProjectionPeriod `json:"periods"`
	EstimateNote bool                      `json:"estimate_note"`
}
