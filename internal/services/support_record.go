package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// ErrMonthClosed se devuelve al intentar modificar un mes cerrado.
var ErrMonthClosed = fmt.Errorf("el mes está cerrado y no puede modificarse")

// SupportRecordService contiene la lógica de negocio de registros de manutención.
type SupportRecordService struct {
	storage         *storage.SupportRecordStorage
	closings        *storage.MonthClosingStorage
	childStorage    *storage.ChildStorage
	categoryStorage *storage.PensionCategoryStorage
	proofDir        string // raíz de comprobantes: {DATA_DIR}/uploads/payment-proofs
}

// NewSupportRecordService crea un nuevo SupportRecordService.
func NewSupportRecordService(st *storage.SupportRecordStorage, closings *storage.MonthClosingStorage, childStorage *storage.ChildStorage, categoryStorage *storage.PensionCategoryStorage, dataDir string) *SupportRecordService {
	return &SupportRecordService{
		storage:         st,
		closings:        closings,
		childStorage:    childStorage,
		categoryStorage: categoryStorage,
		proofDir:        filepath.Join(dataDir, "uploads", "payment-proofs"),
	}
}

// List devuelve los registros de un período.
func (s *SupportRecordService) List(ctx context.Context, year, month int, childID int64) ([]models.SupportRecord, error) {
	records, err := s.storage.ListByFilters(ctx, year, month, childID)
	if err != nil {
		return nil, err
	}
	if records == nil {
		return []models.SupportRecord{}, nil
	}
	return records, nil
}

// GetByID busca un registro por ID.
func (s *SupportRecordService) GetByID(ctx context.Context, id int64) (*models.SupportRecord, error) {
	return s.storage.GetByID(ctx, id)
}

// Create crea un nuevo registro de manutención.
func (s *SupportRecordService) Create(ctx context.Context, childID, categoryID int64, year, month int, amount float64, currency, notes string) (*models.SupportRecord, error) {
	if err := s.validatePeriod(year, month); err != nil {
		return nil, err
	}
	if err := s.checkNotClosed(ctx, year, month); err != nil {
		return nil, err
	}
	if err := s.validateChild(ctx, childID); err != nil {
		return nil, err
	}
	if err := s.validateCategory(ctx, categoryID); err != nil {
		return nil, err
	}
	if amount <= 0 {
		return nil, fmt.Errorf("el monto debe ser mayor a cero")
	}
	currency = normalizeCurrency(currency)

	exists, err := s.storage.Exists(ctx, childID, categoryID, year, month)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("ya existe un registro para este hijo, categoría y mes")
	}

	return s.storage.Create(ctx, &models.SupportRecord{
		ChildID:           childID,
		PensionCategoryID: categoryID,
		Year:              year,
		Month:             month,
		Amount:            amount,
		Currency:          currency,
		Status:            "pending",
		Notes:             nullableString(notes),
	})
}

// Update actualiza monto, categoría y notas de un registro.
func (s *SupportRecordService) Update(ctx context.Context, id int64, amount float64, categoryID int64, notes *string) (*models.SupportRecord, error) {
	year, month, err := s.storage.GetYearMonth(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("registro no encontrado")
	}
	if err := s.checkNotClosed(ctx, year, month); err != nil {
		return nil, err
	}
	if err := s.validateCategory(ctx, categoryID); err != nil {
		return nil, err
	}
	if amount <= 0 {
		return nil, fmt.Errorf("el monto debe ser mayor a cero")
	}
	return s.storage.Update(ctx, id, amount, categoryID, notes)
}

// MarkPaid marca un registro como pagado.
func (s *SupportRecordService) MarkPaid(ctx context.Context, id int64, paidAt *time.Time, paymentMethod, paymentReference, evidenceNotes string, originalAmount *float64, originalCurrency string, exchangeRate *float64) (*models.SupportRecord, error) {
	year, month, err := s.storage.GetYearMonth(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("registro no encontrado")
	}
	if err := s.checkNotClosed(ctx, year, month); err != nil {
		return nil, err
	}

	when := time.Now()
	if paidAt != nil {
		when = *paidAt
	}
	// Si no hay conversión real, no se persisten campos de conversión.
	if originalCurrency == "" || originalAmount == nil {
		originalCurrency = ""
		originalAmount = nil
		exchangeRate = nil
	} else if originalCurrency == "NIO" {
		originalAmount = nil
		originalCurrency = ""
		exchangeRate = nil
	}

	return s.storage.MarkPaid(ctx, id, when, paymentMethod, paymentReference, evidenceNotes, originalAmount, originalCurrency, exchangeRate)
}

// MarkPending devuelve un registro a estado pendiente.
func (s *SupportRecordService) MarkPending(ctx context.Context, id int64) (*models.SupportRecord, error) {
	year, month, err := s.storage.GetYearMonth(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("registro no encontrado")
	}
	if err := s.checkNotClosed(ctx, year, month); err != nil {
		return nil, err
	}
	return s.storage.MarkPending(ctx, id)
}

// MarkRejected marca un registro como rechazado con el motivo.
func (s *SupportRecordService) MarkRejected(ctx context.Context, id int64, reason string) (*models.SupportRecord, error) {
	year, month, err := s.storage.GetYearMonth(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("registro no encontrado")
	}
	if err := s.checkNotClosed(ctx, year, month); err != nil {
		return nil, err
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("el motivo de rechazo es requerido")
	}
	return s.storage.MarkRejected(ctx, id, strings.TrimSpace(reason))
}

// SaveProof valida y persiste un comprobante en disco, guardando el nombre en DB.
func (s *SupportRecordService) SaveProof(ctx context.Context, id int64, fileName string, data []byte) (*models.SupportRecord, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("el archivo está vacío")
	}
	if len(data) > 10*1024*1024 {
		return nil, fmt.Errorf("el archivo es demasiado grande (máx 10MB)")
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	if !validProofExt(ext) {
		return nil, fmt.Errorf("formato de archivo no soportado. Use PDF, PNG, JPG o WEBP")
	}

	record, err := s.storage.GetByID(ctx, id)
	if err != nil || record == nil {
		return nil, fmt.Errorf("registro no encontrado")
	}
	if err := s.checkNotClosed(ctx, record.Year, record.Month); err != nil {
		return nil, err
	}

	dir := filepath.Join(s.proofDir, fmt.Sprintf("%d", record.Year), fmt.Sprintf("%02d", record.Month))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("crear directorio de comprobantes: %w", err)
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%d%s", record.ID, ext))
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return nil, fmt.Errorf("guardar comprobante: %w", err)
	}

	if err := s.storage.UpdateProofFileName(ctx, id, fileName); err != nil {
		return nil, err
	}
	return s.storage.GetByID(ctx, id)
}

// ProofPath devuelve la ruta y nombre de descarga del comprobante de un registro.
func (s *SupportRecordService) ProofPath(ctx context.Context, id int64) (filePath, fileName string, err error) {
	record, err := s.storage.GetByID(ctx, id)
	if err != nil || record == nil {
		return "", "", fmt.Errorf("registro no encontrado")
	}
	if record.ProofFileName == nil || *record.ProofFileName == "" {
		return "", "", fmt.Errorf("el registro no tiene comprobante")
	}
	ext := strings.ToLower(filepath.Ext(*record.ProofFileName))
	dir := filepath.Join(s.proofDir, fmt.Sprintf("%d", record.Year), fmt.Sprintf("%02d", record.Month))
	filePath = filepath.Join(dir, fmt.Sprintf("%d%s", record.ID, ext))
	if _, err := os.Stat(filePath); err != nil {
		return "", "", fmt.Errorf("comprobante no encontrado en disco")
	}
	return filePath, *record.ProofFileName, nil
}

func (s *SupportRecordService) checkNotClosed(ctx context.Context, year, month int) error {
	closed, err := s.closings.IsClosed(ctx, year, month)
	if err != nil {
		return err
	}
	if closed {
		return ErrMonthClosed
	}
	return nil
}

func (s *SupportRecordService) validateChild(ctx context.Context, childID int64) error {
	child, err := s.childStorage.GetByID(ctx, childID)
	if err != nil {
		return fmt.Errorf("error al validar el hijo")
	}
	if child == nil {
		return fmt.Errorf("el hijo no existe")
	}
	return nil
}

func (s *SupportRecordService) validateCategory(ctx context.Context, categoryID int64) error {
	cat, err := s.categoryStorage.GetByID(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("error al validar la categoría")
	}
	if cat == nil {
		return fmt.Errorf("la categoría no existe")
	}
	return nil
}

func (s *SupportRecordService) validatePeriod(year, month int) error {
	if year < 2000 || year > 2100 {
		return fmt.Errorf("el año no es válido")
	}
	if month < 1 || month > 12 {
		return fmt.Errorf("el mes debe estar entre 1 y 12")
	}
	return nil
}

// normalizeCurrency normaliza un código de moneda a 3 letras mayúsculas (default NIO).
func normalizeCurrency(currency string) string {
	c := strings.ToUpper(strings.TrimSpace(currency))
	if len(c) != 3 {
		return "NIO"
	}
	return c
}

// nullableString convierte una string en *string (nil si está vacía).
func nullableString(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func validProofExt(ext string) bool {
	switch ext {
	case ".pdf", ".png", ".jpg", ".jpeg", ".webp":
		return true
	}
	return false
}
