package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

const (
	// MaxAutoAttachmentBytes es el tamaño máximo por PDF (SPEC-124).
	MaxAutoAttachmentBytes = 10 * 1024 * 1024
	// MaxAutoAttachmentsPerRecord es la cantidad máxima por mantenimiento o inspección.
	MaxAutoAttachmentsPerRecord = 20
	maxAutoAttachmentNameLen    = 200
)

// AutoAttachmentUpload representa un archivo recibido por multipart.
type AutoAttachmentUpload struct {
	FileName string
	Reader   io.Reader
}

// AutoAttachmentService guarda y sirve los PDFs de mantenimientos e
// inspecciones (SPEC-124).
type AutoAttachmentService struct {
	storage *storage.AutoAttachmentStorage
	autos   *storage.AutoStorage
	dataDir string
}

// NewAutoAttachmentService crea un nuevo AutoAttachmentService.
func NewAutoAttachmentService(st *storage.AutoAttachmentStorage, autos *storage.AutoStorage, dataDir string) *AutoAttachmentService {
	return &AutoAttachmentService{storage: st, autos: autos, dataDir: dataDir}
}

// ListByAuto devuelve todos los adjuntos de un auto.
func (s *AutoAttachmentService) ListByAuto(ctx context.Context, autoID int64) ([]models.AutoAttachment, error) {
	return s.storage.ListByAuto(ctx, autoID)
}

// ListByOwner devuelve los adjuntos de un mantenimiento o inspección.
func (s *AutoAttachmentService) ListByOwner(ctx context.Context, ownerType string, ownerID int64) ([]models.AutoAttachment, error) {
	return s.storage.ListByOwner(ctx, ownerType, ownerID)
}

// Save valida y guarda un PDF para un mantenimiento o inspección.
func (s *AutoAttachmentService) Save(ctx context.Context, autoID int64, ownerType string, ownerID int64, upload AutoAttachmentUpload) (*models.AutoAttachment, error) {
	if !validAutoAttachmentOwner(ownerType) {
		return nil, ErrAttachmentInvalid
	}

	exists, err := s.storage.OwnerBelongsToAuto(ctx, ownerType, ownerID, autoID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrAttachmentOwnerAbsent
	}

	count, err := s.storage.CountByOwner(ctx, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	if count >= MaxAutoAttachmentsPerRecord {
		return nil, ErrAttachmentLimit
	}

	fileName := sanitizeAutoAttachmentName(upload.FileName)
	if !strings.HasSuffix(strings.ToLower(fileName), ".pdf") {
		return nil, fmt.Errorf("%w: solo se aceptan archivos PDF", ErrAttachmentInvalid)
	}

	dir := s.ownerDir(autoID, ownerType, ownerID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("crear directorio de adjuntos: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".upload-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("crear archivo temporal: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	size, err := io.Copy(tmp, io.LimitReader(upload.Reader, MaxAutoAttachmentBytes+1))
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("guardar adjunto: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return nil, fmt.Errorf("cerrar adjunto: %w", err)
	}
	if size == 0 {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("%w: el archivo está vacío", ErrAttachmentInvalid)
	}
	if size > MaxAutoAttachmentBytes {
		_ = os.Remove(tmpPath)
		return nil, ErrAttachmentTooLarge
	}
	if err := validateAutoPDF(tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("ajustar permisos del adjunto: %w", err)
	}

	att, err := s.storage.Create(ctx, autoID, ownerType, ownerID, fileName, size)
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}

	finalPath := s.attachmentPath(att)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = s.storage.Delete(ctx, att.ID)
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("mover adjunto a su ubicación final: %w", err)
	}
	return att, nil
}

// Get devuelve un adjunto verificando auto, propietario y existencia en disco.
func (s *AutoAttachmentService) Get(ctx context.Context, autoID int64, ownerType string, ownerID, attachmentID int64) (*models.AutoAttachment, string, error) {
	att, err := s.storage.GetByID(ctx, attachmentID)
	if err != nil {
		return nil, "", err
	}
	if att == nil || att.AutoID != autoID || att.OwnerType != ownerType || att.OwnerID != ownerID {
		return nil, "", ErrAttachmentNotFound
	}

	path := s.attachmentPath(att)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, "", ErrAttachmentNotFound
		}
		return nil, "", fmt.Errorf("verificar adjunto: %w", err)
	}
	return att, path, nil
}

// Delete elimina un adjunto individual y su archivo.
func (s *AutoAttachmentService) Delete(ctx context.Context, autoID int64, ownerType string, ownerID, attachmentID int64) error {
	att, err := s.storage.GetByID(ctx, attachmentID)
	if err != nil {
		return err
	}
	if att == nil || att.AutoID != autoID || att.OwnerType != ownerType || att.OwnerID != ownerID {
		return ErrAttachmentNotFound
	}

	path := s.attachmentPath(att)
	if err := s.storage.Delete(ctx, att.ID); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		slog.Warn("no se pudo eliminar el archivo del adjunto", "attachment_id", att.ID, "error", err)
	}
	_ = os.Remove(filepath.Dir(path))
	return nil
}

// DeleteByOwner elimina todos los adjuntos de un mantenimiento o inspección.
func (s *AutoAttachmentService) DeleteByOwner(ctx context.Context, autoID int64, ownerType string, ownerID int64) error {
	items, err := s.storage.ListByOwner(ctx, ownerType, ownerID)
	if err != nil {
		return err
	}
	for _, att := range items {
		if att.AutoID != autoID {
			continue
		}
		path := s.attachmentPath(&att)
		if err := s.storage.Delete(ctx, att.ID); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			slog.Warn("no se pudo eliminar el archivo del adjunto", "attachment_id", att.ID, "error", err)
		}
	}
	_ = os.Remove(s.ownerDir(autoID, ownerType, ownerID))
	return nil
}

// DeleteAutoFiles elimina el directorio completo de adjuntos de un auto.
func (s *AutoAttachmentService) DeleteAutoFiles(autoID int64) error {
	dir := filepath.Join(s.dataDir, "autos", strconv.FormatInt(autoID, 10))
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("eliminar adjuntos del auto: %w", err)
	}
	return nil
}

func (s *AutoAttachmentService) ownerDir(autoID int64, ownerType string, ownerID int64) string {
	folder := "maintenances"
	if ownerType == models.AutoAttachmentOwnerInspection {
		folder = "inspections"
	}
	return filepath.Join(
		s.dataDir,
		"autos",
		strconv.FormatInt(autoID, 10),
		folder,
		strconv.FormatInt(ownerID, 10),
	)
}

func (s *AutoAttachmentService) attachmentPath(att *models.AutoAttachment) string {
	return filepath.Join(s.ownerDir(att.AutoID, att.OwnerType, att.OwnerID), fmt.Sprintf("%d.pdf", att.ID))
}

func validAutoAttachmentOwner(ownerType string) bool {
	return ownerType == models.AutoAttachmentOwnerMaintenance || ownerType == models.AutoAttachmentOwnerInspection
}

func sanitizeAutoAttachmentName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '"' || r < 32 {
			return -1
		}
		return r
	}, name)
	if utf8.RuneCountInString(name) > maxAutoAttachmentNameLen {
		runes := []rune(name)
		name = string(runes[:maxAutoAttachmentNameLen])
	}
	if name == "" || name == "." {
		name = "documento.pdf"
	}
	return name
}

func validateAutoPDF(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("abrir PDF: %w", err)
	}
	defer file.Close()

	header := make([]byte, 5)
	if _, err := io.ReadFull(file, header); err != nil {
		return fmt.Errorf("%w: no se pudo leer el PDF", ErrAttachmentInvalid)
	}
	if string(header) != "%PDF-" {
		return fmt.Errorf("%w: el archivo no es un PDF válido", ErrAttachmentInvalid)
	}
	return nil
}
