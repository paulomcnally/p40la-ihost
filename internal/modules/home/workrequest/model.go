package workrequest

import (
	"errors"
	"time"
)

// ModuleID es el identificador canónico del módulo (ARCH-MODULES).
const (
	ModuleID      = "home.workrequest"
	ModuleVersion = "1.0.0"
)

// Estados del reporte.
const (
	StatusReported  = "reported"
	StatusConverted = "converted"
	StatusDismissed = "dismissed"
)

// Statuses contiene los estados válidos.
var Statuses = []string{StatusReported, StatusConverted, StatusDismissed}

// Categorías del reporte (ADR-002).
const (
	CategoryRepair      = "repair"
	CategoryImprovement = "improvement"
	CategoryMaintenance = "maintenance"
	CategoryOther       = "other"
)

// Categories contiene las categorías válidas.
var Categories = []string{CategoryRepair, CategoryImprovement, CategoryMaintenance, CategoryOther}

// Prioridades del reporte.
const (
	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
)

// Priorities contiene las prioridades válidas.
var Priorities = []string{PriorityLow, PriorityNormal, PriorityHigh}

// Tipos de evidencia multimedia.
const (
	MediaImage = "image"
	MediaVideo = "video"
)

// MediaTypes contiene los tipos de evidencia válidos.
var MediaTypes = []string{MediaImage, MediaVideo}

// Límites de evidencia.
const (
	MaxAttachmentsPerRequest = 10
	MaxAttachmentBytes       = 25 << 20 // 25 MB
)

// Errores de dominio.
var (
	ErrHomeNotFound        = errors.New("vivienda no encontrada")
	ErrWorkRequestNotFound = errors.New("reporte no encontrado")
	ErrAttachmentNotFound  = errors.New("adjunto no encontrado")
	ErrAlreadyConverted    = errors.New("el reporte ya fue convertido")
)

// ValidationError describe un error de validación con código estable.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(code, message string) error {
	return &ValidationError{Code: code, Message: message}
}

// WorkRequest es un reporte de problema o trabajo deseado en una vivienda.
type WorkRequest struct {
	ID              int64        `json:"id"`
	HomeID          int64        `json:"home_id"`
	Title           string       `json:"title"`
	Description     string       `json:"description"`
	Category        string       `json:"category"`
	Priority        string       `json:"priority"`
	Status          string       `json:"status"`
	ReportedAt      string       `json:"reported_at"`
	MaintenanceID   *int64       `json:"maintenance_id"`
	ConvertedAt     *string      `json:"converted_at"`
	AttachmentCount int          `json:"attachment_count"`
	Attachments     []Attachment `json:"attachments,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// Attachment es la evidencia (foto o video) de un reporte.
type Attachment struct {
	ID            int64     `json:"id"`
	WorkRequestID int64     `json:"work_request_id"`
	MediaType     string    `json:"media_type"`
	FileName      string    `json:"file_name"`
	MimeType      string    `json:"mime_type"`
	SizeBytes     int64     `json:"size_bytes"`
	CreatedAt     time.Time `json:"created_at"`
}

// Input es el cuerpo para crear o actualizar un reporte. El estado no se
// edita directamente: cambia por conversión o descarte.
type Input struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Priority    string `json:"priority"`
	ReportedAt  string `json:"reported_at"`
}

// ConvertInput es el cuerpo de la conversión a reparación.
type ConvertInput struct {
	Kind string `json:"kind"`
	Area string `json:"area"`
}

// StatusInput es el cuerpo para descartar o reactivar un reporte.
type StatusInput struct {
	Status string `json:"status"`
}

// Filter restringe el listado por estado y categoría. Limit es interno.
type Filter struct {
	Status   string
	Category string
	Limit    int
}

// List es la respuesta del listado por vivienda.
type List struct {
	Items        []WorkRequest `json:"items"`
	PendingCount int           `json:"pending_count"`
	Truncated    bool          `json:"truncated"`
}

// ConvertResult es la respuesta de la conversión.
type ConvertResult struct {
	MaintenanceID int64        `json:"maintenance_id"`
	WorkRequest   *WorkRequest `json:"work_request"`
}
