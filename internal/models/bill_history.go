package models

import "time"

// Fuentes de modificación de facturas (SPEC-070).
const (
	BillSourceDashboard = "dashboard"
	BillSourceWebhook   = "webhook"
)

// Acciones de auditoría de facturas (SPEC-070).
const (
	BillActionCreated = "created"
	BillActionUpdated = "updated"
	BillActionPaid    = "paid"
)

// FieldChange describe un campo modificado con su valor anterior y nuevo.
type FieldChange struct {
	Field string `json:"field"`
	Old   any    `json:"old,omitempty"`
	New   any    `json:"new,omitempty"`
}

// BillHistory es un evento de auditoría de una factura (SPEC-070): registra
// la acción (creada/editada/pagada), la fuente (dashboard/webhook) y el diff
// de campos modificados.
type BillHistory struct {
	ID        int64         `json:"id"`
	BillID    int64         `json:"bill_id"`
	Action    string        `json:"action"`
	Source    string        `json:"source"`
	Changes   []FieldChange `json:"changes,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}
