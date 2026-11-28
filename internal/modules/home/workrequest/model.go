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

	// UploadLinkPath es el path público del endpoint multipart.
	UploadLinkPath = "/api/work-request-uploads"
	// DefaultUploadTTL es la vigencia por defecto de un link de carga.
	DefaultUploadTTL = 10 * time.Minute
	// MaxUploadTTL es la vigencia máxima de un link de carga.
	MaxUploadTTL = 30 * time.Minute
	// MinUploadSecretLength es el mínimo de caracteres de UPLOAD_TOKEN_SECRET.
	MinUploadSecretLength = 32
	// MaxUploadFilesPerLink es el cupo máximo de archivos por link.
	MaxUploadFilesPerLink = 10
)

// AcceptedMIMEs es la allowlist de evidencia aceptada por la vía MCP.
var AcceptedMIMEs = []string{
	"image/jpeg", "image/png", "image/webp", "image/heic",
	"video/mp4", "video/quicktime", "video/webm",
}

// Errores de dominio.
var (
	ErrHomeNotFound           = errors.New("vivienda no encontrada")
	ErrWorkRequestNotFound    = errors.New("reporte no encontrado")
	ErrAttachmentNotFound     = errors.New("adjunto no encontrado")
	ErrAlreadyConverted       = errors.New("el reporte ya fue convertido")
	ErrDuplicateClientRequest = errors.New("client_request_id duplicado")
)

// ValidationError describe un error de validación con código estable. Status
// permite fijar un HTTP status específico (por ejemplo 409 para conflictos de
// estado) sin cambiar el código visible.
type ValidationError struct {
	Code    string
	Message string
	Status  int
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(code, message string) error {
	return &ValidationError{Code: code, Message: message}
}

func conflict(code, message string) error {
	return &ValidationError{Code: code, Message: message, Status: 409}
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
	ClientRequestID string       `json:"client_request_id,omitempty"`
	Deduplicated    bool         `json:"deduplicated,omitempty"`
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
	SHA256        string    `json:"sha256"`
	CreatedAt     time.Time `json:"created_at"`
	Deduplicated  bool      `json:"deduplicated,omitempty"`
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

// CreateInput agrega la clave de idempotencia opcional al alta (REQ-015).
type CreateInput struct {
	Input
	ClientRequestID string `json:"client_request_id,omitempty"`
}

// UploadConfig configura el secreto y la vigencia de los links de carga.
type UploadConfig struct {
	Secret string
	TTL    time.Duration
}

// UploadLinkRequest es el cuerpo de POST .../upload-link.
type UploadLinkRequest struct {
	Count      int `json:"count"`
	TTLMinutes int `json:"ttl_minutes"`
}

// UploadLink es la respuesta de POST .../upload-link.
type UploadLink struct {
	Token           string   `json:"token"`
	Path            string   `json:"path"`
	ExpiresAt       string   `json:"expires_at"`
	MaxBytes        int64    `json:"max_bytes"`
	MaxRequestBytes int64    `json:"max_request_bytes"`
	MaxFiles        int      `json:"max_files"`
	AcceptedMIME    []string `json:"accepted_mime"`
	AttachmentCount int      `json:"attachment_count"`
	RemainingSlots  int      `json:"remaining_slots"`
	WorkRequestID   int64    `json:"work_request_id"`
	HomeID          int64    `json:"home_id"`
}

// UploadFileResult es el resultado de un archivo dentro de un lote.
type UploadFileResult struct {
	FileName   string      `json:"file_name"`
	Status     string      `json:"status"`
	Attachment *Attachment `json:"attachment,omitempty"`
	Error      string      `json:"error,omitempty"`
	Message    string      `json:"message,omitempty"`
}

// UploadResult es la respuesta de POST /api/work-request-uploads.
type UploadResult struct {
	HomeID          int64              `json:"home_id"`
	WorkRequestID   int64              `json:"work_request_id"`
	Added           int                `json:"added"`
	Duplicates      int                `json:"duplicates"`
	Rejected        int                `json:"rejected"`
	AttachmentCount int                `json:"attachment_count"`
	RemainingSlots  int                `json:"remaining_slots"`
	Results         []UploadFileResult `json:"results"`
}

// UploadToken es la fila efímera asociada a un jti firmado.
type UploadToken struct {
	JTI           string
	WorkRequestID int64
	MaxFiles      int
	FilesUsed     int
	ExpiresAt     time.Time
}

// UploadClaims es el payload firmado y validado de un token de carga.
type UploadClaims struct {
	HomeID        int64
	WorkRequestID int64
	ExpiresAt     time.Time
	JTI           string
	MaxFiles      int
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
