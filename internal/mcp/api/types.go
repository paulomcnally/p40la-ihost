package api

import "time"

// Service es el subconjunto del servicio que necesita el MCP.
type Service struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Frequency       string  `json:"frequency"`
	SuggestedAmount float64 `json:"suggested_amount"`
	Active          bool    `json:"active"`
	CurrencyID      int64   `json:"currency_id"`
	HomeID          int64   `json:"home_id"`
	Institution     string  `json:"institution,omitempty"`
	IconKey         string  `json:"icon_key,omitempty"`
	BillingType     string  `json:"billing_type,omitempty"`
	BillingDay      *int    `json:"billing_day,omitempty"`
	AutoGenerate    bool    `json:"auto_generate,omitempty"`
	IsRecurring     bool    `json:"is_recurring,omitempty"`
	StartDate       *string `json:"start_date,omitempty"`
	EndDate         *string `json:"end_date,omitempty"`
}

// Bill es el subconjunto de una factura que necesita el MCP.
type Bill struct {
	ID               int64      `json:"id"`
	ServiceID        int64      `json:"service_id"`
	Year             int        `json:"year"`
	Month            int        `json:"month"`
	Amount           float64    `json:"amount"`
	InvoiceNumber    string     `json:"invoice_number,omitempty"`
	Status           string     `json:"status"`
	IssueDate        *string    `json:"issue_date,omitempty"`
	DueDate          *string    `json:"due_date,omitempty"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
	PaymentReference string     `json:"payment_reference,omitempty"`
}

// GenerateRequest es el contrato de POST /api/services/{id}/bills/generate.
type GenerateRequest struct {
	From          string   `json:"from"`
	To            string   `json:"to,omitempty"`
	Amount        *float64 `json:"amount,omitempty"`
	DryRun        bool     `json:"dry_run"`
	AllowInactive bool     `json:"allow_inactive,omitempty"`
}

// GenerateItem es el resultado por período de una generación.
type GenerateItem struct {
	Year    int     `json:"year"`
	Month   int     `json:"month"`
	Period  string  `json:"period"`
	Amount  float64 `json:"amount"`
	Status  string  `json:"status"`
	BillID  int64   `json:"bill_id,omitempty"`
	Reason  string  `json:"reason,omitempty"`
	Message string  `json:"message,omitempty"`
}

// GenerateResult es el resumen de una generación masiva.
type GenerateResult struct {
	ServiceID    int64          `json:"service_id"`
	ServiceName  string         `json:"service_name,omitempty"`
	From         string         `json:"from"`
	To           string         `json:"to"`
	DryRun       bool           `json:"dry_run"`
	Amount       float64        `json:"amount"`
	CurrencyCode string         `json:"currency_code,omitempty"`
	Created      int            `json:"created"`
	Skipped      int            `json:"skipped"`
	Failed       int            `json:"failed"`
	Items        []GenerateItem `json:"items"`
}

// PayRequest es el contrato de POST /api/services/{id}/bills/pay. Se debe
// indicar exactamente un selector: BillIDs, Month, From+To o AllPendingUpTo.
type PayRequest struct {
	BillIDs          []int64 `json:"bill_ids,omitempty"`
	Month            string  `json:"month,omitempty"`
	From             string  `json:"from,omitempty"`
	To               string  `json:"to,omitempty"`
	AllPendingUpTo   string  `json:"all_pending_up_to,omitempty"`
	PaidAt           string  `json:"paid_at,omitempty"`
	DriveURL         string  `json:"drive_url,omitempty"`
	PaymentReference string  `json:"payment_reference,omitempty"`
	DryRun           bool    `json:"dry_run"`
}

// PayItem es el resultado por factura de un pago.
type PayItem struct {
	BillID  int64   `json:"bill_id"`
	Year    int     `json:"year"`
	Month   int     `json:"month"`
	Period  string  `json:"period"`
	Amount  float64 `json:"amount"`
	Status  string  `json:"status"`
	Reason  string  `json:"reason,omitempty"`
	Message string  `json:"message,omitempty"`
}

// PayResult es el resumen de un pago masivo.
type PayResult struct {
	ServiceID    int64     `json:"service_id"`
	ServiceName  string    `json:"service_name,omitempty"`
	Selector     string    `json:"selector"`
	PaidAt       string    `json:"paid_at"`
	DryRun       bool      `json:"dry_run"`
	Paid         int       `json:"paid"`
	Skipped      int       `json:"skipped"`
	Failed       int       `json:"failed"`
	TotalAmount  float64   `json:"total_amount"`
	CurrencyCode string    `json:"currency_code,omitempty"`
	Items        []PayItem `json:"items"`
}

// APIError representa un error HTTP de la API con código estable.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return e.Code + ": " + e.Message
}
