package models

import "time"

// ServiceCycle representa un período de vigencia renovable de un servicio
// (SPEC-091). Cada renovación crea un ciclo nuevo con sequence creciente; las
// facturas se asocian a su ciclo vía bills.cycle_id.
type ServiceCycle struct {
	ID        int64     `json:"id"`
	ServiceID int64     `json:"service_id"`
	Sequence  int       `json:"sequence"`
	StartDate *string   `json:"start_date,omitempty"`
	EndDate   *string   `json:"end_date,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
