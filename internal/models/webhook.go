package models

// WebhookBillPayload es el schema de entrada del webhook de facturas por
// servicio (SPEC-069). Los clientes externos deben adaptar sus envíos a este
// contrato. `year`, `month` y `amount` son obligatorios; el resto opcional.
type WebhookBillPayload struct {
	Year             int     `json:"year"`
	Month            int     `json:"month"`
	Amount           float64 `json:"amount"`
	InvoiceNumber    string  `json:"invoice_number,omitempty"`
	Status           string  `json:"status,omitempty"`
	PaidAt           string  `json:"paid_at,omitempty"`
	PaymentReference string  `json:"payment_reference,omitempty"`
	DriveURL         string  `json:"drive_url,omitempty"`
}

// WebhookResult describe el resultado de un upsert vía webhook (SPEC-069).
type WebhookResult struct {
	Bill    *Bill `json:"bill"`
	Created bool  `json:"created"`
}
