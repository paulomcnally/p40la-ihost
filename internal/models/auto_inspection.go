package models

import "time"

// AutoInspection representa una inspección mecánica realizada a un auto
// (SPEC-124). inspection_year es el año del ciclo que cumple la inspección.
type AutoInspection struct {
	ID             int64     `json:"id"`
	AutoID         int64     `json:"auto_id"`
	PerformedAt    string    `json:"performed_at"`
	InspectionYear int64     `json:"inspection_year"`
	InstitutionID  int64     `json:"institution_id"`
	Result         string    `json:"result"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AutoInspectionDetail agrega el nombre de la institución y los adjuntos.
type AutoInspectionDetail struct {
	AutoInspection
	InstitutionName string           `json:"institution_name"`
	Attachments     []AutoAttachment `json:"attachments"`
}

// AutoInspectionInput es el payload de creación/edición.
type AutoInspectionInput struct {
	PerformedAt    string `json:"performed_at"`
	InspectionYear int64  `json:"inspection_year"`
	InstitutionID  int64  `json:"institution_id"`
	Result         string `json:"result"`
	Notes          string `json:"notes"`
}

// InspectionSchedule es el cronograma calculado a partir del año del auto y de
// las inspecciones aprobadas (SPEC-124). No se persiste.
type InspectionSchedule struct {
	FirstDueYear  int64   `json:"first_due_year"`
	NextDueYear   int64   `json:"next_due_year"`
	NextDueDate   string  `json:"next_due_date"`
	Status        string  `json:"status"`
	DaysUntilDue  int64   `json:"days_until_due"`
	ApprovedYears []int64 `json:"approved_years"`
	RejectedYears []int64 `json:"rejected_years"`
}

// AutoInspectionList es la respuesta de GET /api/autos/{id}/inspections.
type AutoInspectionList struct {
	Items    []AutoInspectionDetail `json:"items"`
	Schedule InspectionSchedule     `json:"schedule"`
}

// AutoInspectionAlert es el estado compacto que consume el listado de autos.
type AutoInspectionAlert struct {
	AutoID       int64  `json:"auto_id"`
	NextDueYear  int64  `json:"next_due_year"`
	NextDueDate  string `json:"next_due_date"`
	Status       string `json:"status"`
	DaysUntilDue int64  `json:"days_until_due"`
}

const (
	InspectionResultApproved = "approved"
	InspectionResultRejected = "rejected"

	InspectionStatusOverdue = "overdue"
	InspectionStatusDueSoon = "due_soon"
	InspectionStatusOK      = "ok"
)

// InspectionDueSoonDays es el umbral para marcar una inspección como próxima.
const InspectionDueSoonDays = 60
