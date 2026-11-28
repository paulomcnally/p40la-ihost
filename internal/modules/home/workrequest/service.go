package workrequest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	homemaintenance "github.com/paulomcnally/p40la-ihost/internal/modules/home/maintenance"
)

const (
	maxTitleLen        = 200
	maxDescriptionLen  = 2000
	maxListItems       = 500
	maxMaintenanceDesc = 500
	dateLayout         = "2006-01-02"
	defaultCategory    = CategoryRepair
	defaultPriority    = PriorityNormal
	defaultKind        = homemaintenance.KindCorrective
	defaultArea        = homemaintenance.AreaOther
)

// MaintenanceCreator es el contrato con home.maintenance (ADR-005). El módulo
// recibe el servicio existente; no modifica su código ni su esquema.
type MaintenanceCreator interface {
	Create(ctx context.Context, homeID int64, input homemaintenance.Input) (*homemaintenance.Maintenance, error)
	Delete(ctx context.Context, homeID, id int64) error
}

// Service contiene la lógica de negocio del módulo (ARCH-LAYERS).
type Service struct {
	storage     *Storage
	maintenance MaintenanceCreator
	dataDir     string
	now         func() time.Time
}

// NewService crea el servicio del módulo.
func NewService(storage *Storage, maintenance MaintenanceCreator, dataDir string) *Service {
	return &Service{storage: storage, maintenance: maintenance, dataDir: dataDir, now: time.Now}
}

// SetClock permite fijar el reloj en tests.
func (s *Service) SetClock(now func() time.Time) { s.now = now }

// List devuelve los reportes de una vivienda.
func (s *Service) List(ctx context.Context, homeID int64, filter Filter) (*List, error) {
	if err := s.requireHome(ctx, homeID); err != nil {
		return nil, err
	}
	filter.Status = strings.TrimSpace(filter.Status)
	if filter.Status != "" && !contains(Statuses, filter.Status) {
		return nil, invalid("invalid_status", "El estado del reporte no es válido")
	}
	filter.Category = strings.TrimSpace(filter.Category)
	if filter.Category != "" && !contains(Categories, filter.Category) {
		return nil, invalid("invalid_category", "La categoría del reporte no es válida")
	}

	filter.Limit = maxListItems + 1
	items, err := s.storage.List(ctx, homeID, filter)
	if err != nil {
		return nil, err
	}
	truncated := false
	if len(items) > maxListItems {
		items = items[:maxListItems]
		truncated = true
	}
	for index := range items {
		items[index].Attachments = nil
	}

	pending, err := s.storage.PendingCount(ctx, homeID)
	if err != nil {
		return nil, err
	}
	return &List{Items: items, PendingCount: pending, Truncated: truncated}, nil
}

// Get devuelve un reporte con su evidencia.
func (s *Service) Get(ctx context.Context, homeID, id int64) (*WorkRequest, error) {
	if err := s.requireHome(ctx, homeID); err != nil {
		return nil, err
	}
	item, err := s.storage.Get(ctx, homeID, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrWorkRequestNotFound
	}
	attachments, err := s.storage.ListAttachments(ctx, id)
	if err != nil {
		return nil, err
	}
	item.Attachments = attachments
	item.AttachmentCount = len(attachments)
	return item, nil
}

// Create registra un reporte en estado reported.
func (s *Service) Create(ctx context.Context, homeID int64, input Input) (*WorkRequest, error) {
	if err := s.requireHome(ctx, homeID); err != nil {
		return nil, err
	}
	if err := s.prepare(&input); err != nil {
		return nil, err
	}
	id, err := s.storage.Create(ctx, homeID, input)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, homeID, id)
}

// Update edita los campos del reporte sin tocar su estado.
func (s *Service) Update(ctx context.Context, homeID, id int64, input Input) (*WorkRequest, error) {
	if err := s.requireWorkRequest(ctx, homeID, id); err != nil {
		return nil, err
	}
	if err := s.prepare(&input); err != nil {
		return nil, err
	}
	exists, err := s.storage.Update(ctx, homeID, id, input)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrWorkRequestNotFound
	}
	return s.Get(ctx, homeID, id)
}

// Delete elimina un reporte, sus adjuntos y los archivos de evidencia.
func (s *Service) Delete(ctx context.Context, homeID, id int64) error {
	if err := s.requireWorkRequest(ctx, homeID, id); err != nil {
		return err
	}
	exists, err := s.storage.Delete(ctx, homeID, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrWorkRequestNotFound
	}
	_ = os.RemoveAll(s.attachmentDir(id))
	return nil
}

// SetStatus descarta o reactiva un reporte (REQ-012).
func (s *Service) SetStatus(ctx context.Context, homeID, id int64, input StatusInput) (*WorkRequest, error) {
	if err := s.requireHome(ctx, homeID); err != nil {
		return nil, err
	}
	item, err := s.storage.Get(ctx, homeID, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrWorkRequestNotFound
	}
	target := strings.TrimSpace(input.Status)
	if target != StatusReported && target != StatusDismissed {
		return nil, invalid("invalid_status", "El estado destino no es válido")
	}
	if item.Status == StatusConverted {
		return nil, invalid("invalid_status", "Un reporte convertido no cambia de estado")
	}
	if item.Status == target {
		return s.Get(ctx, homeID, id)
	}
	if _, err := s.storage.SetStatus(ctx, homeID, id, target); err != nil {
		return nil, err
	}
	return s.Get(ctx, homeID, id)
}

// Convert crea una reparación vacía vía home.maintenance y enlaza el reporte.
func (s *Service) Convert(ctx context.Context, homeID, id int64, input ConvertInput) (*ConvertResult, error) {
	if err := s.requireHome(ctx, homeID); err != nil {
		return nil, err
	}
	item, err := s.storage.Get(ctx, homeID, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, ErrWorkRequestNotFound
	}
	if item.Status == StatusConverted {
		return nil, ErrAlreadyConverted
	}
	if item.Status != StatusReported {
		return nil, invalid("invalid_status", "Solo un reporte pendiente se puede convertir")
	}
	if s.maintenance == nil {
		return nil, fmt.Errorf("servicio de reparaciones no disponible")
	}

	kind := strings.TrimSpace(input.Kind)
	if kind == "" {
		kind = defaultKind
	}
	if !contains(homemaintenance.Kinds, kind) {
		return nil, invalid("invalid_kind", "El tipo de reparación no es válido")
	}
	area := strings.TrimSpace(input.Area)
	if area == "" {
		area = defaultArea
	}
	if !contains(homemaintenance.Areas, area) {
		return nil, invalid("invalid_area", "El área de la reparación no es válida")
	}

	created, err := s.maintenance.Create(ctx, homeID, homemaintenance.Input{
		PerformedAt: s.now().Format(dateLayout),
		Kind:        kind,
		Area:        area,
		Description: convertedDescription(item),
	})
	if err != nil {
		return nil, err
	}

	convertedAt := s.now().Format(time.RFC3339)
	ok, err := s.storage.SetConverted(ctx, homeID, id, created.ID, convertedAt)
	if err != nil {
		_ = s.maintenance.Delete(ctx, homeID, created.ID)
		return nil, err
	}
	if !ok {
		_ = s.maintenance.Delete(ctx, homeID, created.ID)
		return nil, ErrAlreadyConverted
	}

	updated, err := s.Get(ctx, homeID, id)
	if err != nil {
		return nil, err
	}
	return &ConvertResult{MaintenanceID: created.ID, WorkRequest: updated}, nil
}

// AddAttachment valida y guarda una foto o video de evidencia.
func (s *Service) AddAttachment(ctx context.Context, homeID, workRequestID int64, mediaType, fileName string, reader io.Reader) (*Attachment, error) {
	if err := s.requireWorkRequest(ctx, homeID, workRequestID); err != nil {
		return nil, err
	}
	mediaType = strings.TrimSpace(mediaType)
	if mediaType == "" {
		mediaType = MediaImage
	}
	if !contains(MediaTypes, mediaType) {
		return nil, invalid("invalid_file", "El tipo de evidencia no es válido")
	}

	count, err := s.storage.CountAttachments(ctx, workRequestID)
	if err != nil {
		return nil, err
	}
	if count >= MaxAttachmentsPerRequest {
		return nil, invalid("attachment_limit", "El reporte alcanzó el máximo de 10 adjuntos")
	}

	data, err := io.ReadAll(io.LimitReader(reader, MaxAttachmentBytes+1))
	if err != nil {
		return nil, invalid("invalid_file", "No se pudo leer el archivo")
	}
	if len(data) == 0 {
		return nil, invalid("invalid_file", "El archivo está vacío")
	}
	if int64(len(data)) > MaxAttachmentBytes {
		return nil, invalid("request_too_large", "El archivo supera el máximo de 25 MB")
	}

	detected := detectFileKind(data)
	if detected.mimeType == "" || detected.mediaType != mediaType {
		return nil, invalid("invalid_file", "Tipo de archivo no permitido")
	}

	name := sanitizeFileName(fileName)
	if name == "" || name == "." {
		name = "evidencia" + detected.ext
	} else if filepath.Ext(strings.ToLower(name)) == "" {
		name += detected.ext
	}

	attachmentID, err := s.storage.CreateAttachment(ctx, workRequestID, detected.mediaType, name, detected.mimeType, int64(len(data)))
	if err != nil {
		return nil, err
	}
	path := s.attachmentFilePath(workRequestID, attachmentID, name)
	if err := writeAtomic(path, data); err != nil {
		_, _ = s.storage.DeleteAttachment(ctx, workRequestID, attachmentID)
		return nil, err
	}
	return s.storage.GetAttachment(ctx, workRequestID, attachmentID)
}

// AttachmentPath valida la pertenencia vivienda → reporte → adjunto y devuelve
// la ruta física del archivo.
func (s *Service) AttachmentPath(ctx context.Context, homeID, workRequestID, attachmentID int64) (*Attachment, string, error) {
	if err := s.requireWorkRequest(ctx, homeID, workRequestID); err != nil {
		return nil, "", err
	}
	attachment, err := s.storage.GetAttachment(ctx, workRequestID, attachmentID)
	if err != nil {
		return nil, "", err
	}
	if attachment == nil {
		return nil, "", ErrAttachmentNotFound
	}
	return attachment, s.attachmentFilePath(workRequestID, attachment.ID, attachment.FileName), nil
}

// DeleteAttachment elimina la evidencia y su archivo.
func (s *Service) DeleteAttachment(ctx context.Context, homeID, workRequestID, attachmentID int64) error {
	if err := s.requireWorkRequest(ctx, homeID, workRequestID); err != nil {
		return err
	}
	attachment, err := s.storage.GetAttachment(ctx, workRequestID, attachmentID)
	if err != nil {
		return err
	}
	if attachment == nil {
		return ErrAttachmentNotFound
	}
	exists, err := s.storage.DeleteAttachment(ctx, workRequestID, attachmentID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAttachmentNotFound
	}
	_ = os.Remove(s.attachmentFilePath(workRequestID, attachment.ID, attachment.FileName))
	return nil
}

func (s *Service) prepare(input *Input) error {
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return invalid("title_required", "El título del reporte es requerido")
	}
	if utf8.RuneCountInString(input.Title) > maxTitleLen {
		return invalid("title_too_long", "El título del reporte supera el máximo permitido")
	}

	input.Description = strings.TrimSpace(input.Description)
	if utf8.RuneCountInString(input.Description) > maxDescriptionLen {
		return invalid("description_too_long", "La descripción del reporte supera el máximo permitido")
	}

	input.Category = strings.TrimSpace(input.Category)
	if input.Category == "" {
		input.Category = defaultCategory
	}
	if !contains(Categories, input.Category) {
		return invalid("invalid_category", "La categoría del reporte no es válida")
	}

	input.Priority = strings.TrimSpace(input.Priority)
	if input.Priority == "" {
		input.Priority = defaultPriority
	}
	if !contains(Priorities, input.Priority) {
		return invalid("invalid_priority", "La prioridad del reporte no es válida")
	}

	reportedAt, err := time.Parse(dateLayout, strings.TrimSpace(input.ReportedAt))
	if err != nil {
		return invalid("invalid_date", "La fecha del reporte no es válida")
	}
	today := s.now()
	if reportedAt.After(time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())) {
		return invalid("future_date", "La fecha del reporte no puede ser futura")
	}
	input.ReportedAt = reportedAt.Format(dateLayout)
	return nil
}

func (s *Service) requireHome(ctx context.Context, homeID int64) error {
	if homeID <= 0 {
		return ErrHomeNotFound
	}
	ok, err := s.storage.HomeExists(ctx, homeID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrHomeNotFound
	}
	return nil
}

func (s *Service) requireWorkRequest(ctx context.Context, homeID, id int64) error {
	if err := s.requireHome(ctx, homeID); err != nil {
		return err
	}
	if id <= 0 {
		return ErrWorkRequestNotFound
	}
	item, err := s.storage.Get(ctx, homeID, id)
	if err != nil {
		return err
	}
	if item == nil {
		return ErrWorkRequestNotFound
	}
	return nil
}

func (s *Service) attachmentDir(workRequestID int64) string {
	return filepath.Join(s.dataDir, "workrequests", strconv.FormatInt(workRequestID, 10))
}

func (s *Service) attachmentFilePath(workRequestID, attachmentID int64, fileName string) string {
	return filepath.Join(s.attachmentDir(workRequestID), fmt.Sprintf("%d-%s", attachmentID, fileName))
}

func convertedDescription(item *WorkRequest) string {
	description := item.Title
	if item.Description != "" {
		description += "\n\n" + item.Description
	}
	if utf8.RuneCountInString(description) > maxMaintenanceDesc {
		description = string([]rune(description)[:maxMaintenanceDesc])
	}
	return description
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

type fileKind struct {
	mediaType string
	mimeType  string
	ext       string
}

// detectFileKind valida la firma real del archivo contra la allowlist de
// imágenes y videos (ADR-003). No confía en la extensión ni en el Content-Type.
func detectFileKind(data []byte) fileKind {
	switch {
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return fileKind{MediaImage, "image/jpeg", ".jpg"}
	case len(data) >= 8 && bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")):
		return fileKind{MediaImage, "image/png", ".png"}
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return fileKind{MediaImage, "image/webp", ".webp"}
	case len(data) >= 4 && bytes.Equal(data[:4], []byte{0x1A, 0x45, 0xDF, 0xA3}):
		head := data
		if len(head) > 128 {
			head = head[:128]
		}
		if bytes.Contains(head, []byte("webm")) {
			return fileKind{MediaVideo, "video/webm", ".webm"}
		}
		return fileKind{}
	case len(data) >= 12 && string(data[4:8]) == "ftyp":
		brand := string(data[8:12])
		switch brand {
		case "heic", "heix", "hevc", "hevx", "heim", "heis", "hevm", "hevs", "mif1", "msf1":
			return fileKind{MediaImage, "image/heic", ".heic"}
		case "qt  ":
			return fileKind{MediaVideo, "video/quicktime", ".mov"}
		case "isom", "iso2", "iso4", "iso5", "iso6", "mp41", "mp42", "avc1", "dash", "M4V ", "mmp4":
			return fileKind{MediaVideo, "video/mp4", ".mp4"}
		}
		return fileKind{}
	default:
		return fileKind{}
	}
}

func sanitizeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r == '/' || r == '\\' || r == '"' {
			return -1
		}
		return r
	}, name)
	if name == "." || name == ".." {
		return ""
	}
	if utf8.RuneCountInString(name) > 180 {
		name = string([]rune(name)[:180])
	}
	return name
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("crear directorio de evidencia: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("escribir evidencia: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("mover evidencia: %w", err)
	}
	return nil
}
