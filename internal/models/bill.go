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
