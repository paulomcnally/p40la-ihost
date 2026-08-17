package models

import "time"

// Bill representa una factura de pago asociada a un servicio.
type Bill struct {
	ID            int64      `json:"id"`
	ServiceID     int64      `json:"service_id"`
	Year          int        `json:"year"`
	Month         int        `json:"month"`
	Amount        float64    `json:"amount"`
	InvoiceNumber string     `json:"invoice_number,omitempty"`
	Status        string     `json:"status"`
	DriveURL      string     `json:"drive_url,omitempty"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// PendingBillDetail detalla una factura pendiente con contexto (casa,
// institución, servicio, moneda) para el resumen diario (SPEC-031).
type PendingBillDetail struct {
	BillID         int64     `json:"bill_id"`
	ServiceID      int64     `json:"service_id"`
	HomeID         int64     `json:"home_id"`
	HomeName       string    `json:"home_name"`
	Institution    string    `json:"institution"`
	ServiceName    string    `json:"service_name"`
	CurrencySymbol string    `json:"currency_symbol"`
	Year           int       `json:"year"`
	Month          int       `json:"month"`
	Amount         float64   `json:"amount"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}
