package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// Estados de una deuda.
const (
	DebtStatusActive   = "activa"
	DebtStatusInactive = "inactiva"
	DebtStatusFinished = "finalizada"
)

// MaxDebtDocumentBytes es el tamaño máximo permitido para un PDF de deuda
// (factura o comprobante, SPEC-122).
const MaxDebtDocumentBytes int64 = 10 * 1024 * 1024

var (
	// ErrDocumentAfterPayment indica que la cuota quedó pagada pero un
	// documento no pudo guardarse en disco. El usuario puede reintentar con
	// SaveDocument.
	ErrDocumentAfterPayment = errors.New("la cuota quedó pagada pero no se pudo guardar un documento")
	// ErrDocumentNotFound indica que la cuota no tiene el documento solicitado.
	ErrDocumentNotFound = errors.New("la cuota no tiene el documento solicitado")
)

// DebtDocumentKind identifica el tipo de PDF de un pago (SPEC-122).
type DebtDocumentKind string

const (
	DebtDocumentInvoice DebtDocumentKind = "invoice"
	DebtDocumentReceipt DebtDocumentKind = "receipt"
)

// PDFUpload representa un PDF en tránsito (factura o comprobante).
type PDFUpload struct {
	FileName string
	Reader   io.Reader
}

// DebtService contiene la lógica de negocio para deudas (SPEC-054).
type DebtService struct {
	debts        *storage.DebtStorage
	debtBills    *storage.DebtBillStorage
	institutions *storage.InstitutionStorage
	currencies   *storage.CurrencyStorage
	documentsDir string
}

// NewDebtService crea un nuevo DebtService. dataDir es el mismo directorio
// donde vive app.db; las facturas y comprobantes se guardan en
// dataDir/documents (SPEC-122).
func NewDebtService(
	debts *storage.DebtStorage,
	debtBills *storage.DebtBillStorage,
	institutions *storage.InstitutionStorage,
	currencies *storage.CurrencyStorage,
	dataDir string,
) *DebtService {
	if strings.TrimSpace(dataDir) == "" {
		dataDir = "."
	}
	return &DebtService{
		debts:        debts,
		debtBills:    debtBills,
		institutions: institutions,
		currencies:   currencies,
		documentsDir: filepath.Join(dataDir, "documents"),
	}
}

// List devuelve todas las deudas.
func (s *DebtService) List(ctx context.Context) ([]models.Debt, error) {
	debts, err := s.debts.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range debts {
		debts[i].HasDocuments = s.hasAnyDocument(debts[i].ID)
	}
	return debts, nil
}

// GetByID busca una deuda por ID.
func (s *DebtService) GetByID(ctx context.Context, id int64) (*models.Debt, error) {
	debt, err := s.debts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if debt != nil {
		debt.HasDocuments = s.hasAnyDocument(debt.ID)
	}
	return debt, nil
}

// Create crea una deuda y genera su serie completa de cuotas si está activa.
func (s *DebtService) Create(ctx context.Context, debt *models.Debt) (*models.Debt, error) {
	if err := s.validate(ctx, debt); err != nil {
		return nil, err
	}
	if debt.InstallmentAmount <= 0 {
		debt.InstallmentAmount = computeInstallmentAmount(debt.Total, debt.InstallmentsTotal)
	}

	created, err := s.debts.Create(ctx, debt)
	if err != nil {
		return nil, err
	}

	if err := s.ensureDebtBills(ctx, created); err != nil {
		return nil, err
	}
	return created, nil
}

// Update actualiza una deuda y regenera las cuotas faltantes si está activa.
func (s *DebtService) Update(ctx context.Context, debt *models.Debt) (*models.Debt, error) {
	if debt.ID == 0 {
		return nil, fmt.Errorf("id de deuda requerido")
	}
	existing, err := s.debts.GetByID(ctx, debt.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}
	if err := s.validate(ctx, debt); err != nil {
		return nil, err
	}
	if debt.InstallmentAmount <= 0 {
		debt.InstallmentAmount = computeInstallmentAmount(debt.Total, debt.InstallmentsTotal)
	}

	// SPEC-104 REQ-009a: si se agregan cuotas a una deuda cerrada, vuelve a estar activa.
	if debt.Status == DebtStatusFinished && debt.InstallmentsTotal > existing.InstallmentsTotal {
		debt.Status = DebtStatusActive
	}

	updated, err := s.debts.Update(ctx, debt)
	if err != nil {
		return nil, err
	}

	// SPEC-104 REQ-009b: al reducir el total de cuotas, las sobrantes se eliminan.
	if debt.InstallmentsTotal < existing.InstallmentsTotal {
		if err := s.debtBills.SoftDeleteOverflow(ctx, debt.ID, debt.InstallmentsTotal); err != nil {
			return nil, err
		}
	}

	if err := s.ensureDebtBills(ctx, updated); err != nil {
		return nil, err
	}
	// Releer para devolver el resumen de cuotas actualizado (SPEC-104) y el
	// indicador de comprobantes (SPEC-122).
	return s.GetByID(ctx, debt.ID)
}

// Delete elimina lógicamente una deuda y sus cuotas.
func (s *DebtService) Delete(ctx context.Context, id int64) error {
	if err := s.debtBills.SoftDeleteByDebt(ctx, id); err != nil {
		return err
	}
	return s.debts.SoftDelete(ctx, id)
}

// ListBills devuelve las cuotas de una deuda.
func (s *DebtService) ListBills(ctx context.Context, debtID int64) ([]models.DebtBill, error) {
	bills, err := s.debtBills.ListByDebt(ctx, debtID)
	if err != nil {
		return nil, err
	}
	s.attachDocumentFlagsToBills(bills)
	return bills, nil
}

// ListBillsByMonth devuelve las cuotas cuyo vencimiento cae en un mes
// (vista Calendario).
func (s *DebtService) ListBillsByMonth(ctx context.Context, year, month int) ([]models.DebtBill, error) {
	bills, err := s.debtBills.ListByMonth(ctx, year, month)
	if err != nil {
		return nil, err
	}
	s.attachDocumentFlagsToBills(bills)
	return bills, nil
}

// PayBill marca una cuota como pagada. Se mantiene por compatibilidad con las
// llamadas existentes; para adjuntar documentos usar PayBillWithDocuments.
func (s *DebtService) PayBill(ctx context.Context, id int64, paidAt time.Time, paymentReference string) (*models.DebtBill, error) {
	return s.PayBillWithDocuments(ctx, id, paidAt, paymentReference, nil, nil)
}

// PayBillWithDocuments marca una cuota como pagada y, opcionalmente, guarda la
// factura y/o el comprobante PDF en {DATA_DIR}/documents/{debt_id}/
// (SPEC-122).
func (s *DebtService) PayBillWithDocuments(
	ctx context.Context,
	id int64,
	paidAt time.Time,
	paymentReference string,
	invoice *PDFUpload,
	receipt *PDFUpload,
) (*models.DebtBill, error) {
	bill, err := s.debtBills.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if bill == nil {
		return nil, fmt.Errorf("la cuota no existe")
	}
	if bill.Status == "paid" {
		return nil, fmt.Errorf("la cuota ya está pagada")
	}

	uploads := []struct {
		kind   DebtDocumentKind
		upload *PDFUpload
	}{
		{DebtDocumentInvoice, invoice},
		{DebtDocumentReceipt, receipt},
	}

	var staged []*stagedDocument
	for _, item := range uploads {
		if item.upload == nil {
			continue
		}
		doc, stageErr := s.stageDocument(bill, item.kind, *item.upload)
		if stageErr != nil {
			for _, pending := range staged {
				pending.cleanup()
			}
			return nil, stageErr
		}
		staged = append(staged, doc)
	}
	defer func() {
		for _, doc := range staged {
			doc.cleanup()
		}
	}()

	paid, err := s.debtBills.Pay(ctx, id, paidAt, paymentReference)
	if err != nil {
		return nil, err
	}

	for _, doc := range staged {
		if err := doc.commit(); err != nil {
			// La cuota ya quedó pagada; se finaliza la deuda y se devuelve un
			// error específico para que el usuario pueda reintentar el
			// documento con SaveDocument (SPEC-122 REQ-013).
			_, _ = s.debtBills.FinalizeIfCompleted(ctx, bill.DebtID)
			return nil, fmt.Errorf("%w: %v", ErrDocumentAfterPayment, err)
		}
	}

	// SPEC-104 REQ-001: si era la última cuota pendiente, la deuda queda finalizada.
	if _, err := s.debtBills.FinalizeIfCompleted(ctx, bill.DebtID); err != nil {
		return nil, err
	}
	if paid != nil {
		s.attachDocumentFlags(paid)
	}
	return paid, nil
}

// SaveDocument adjunta o reemplaza una factura o comprobante de una cuota ya
// pagada (SPEC-122 REQ-013).
func (s *DebtService) SaveDocument(ctx context.Context, id int64, kind DebtDocumentKind, upload PDFUpload) (*models.DebtBill, error) {
	if !validDocumentKind(kind) {
		return nil, fmt.Errorf("tipo de documento inválido")
	}
	bill, err := s.debtBills.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if bill == nil {
		return nil, fmt.Errorf("la cuota no existe")
	}
	if bill.Status != "paid" {
		return nil, fmt.Errorf("la cuota debe estar pagada para adjuntar un documento")
	}

	staged, err := s.stageDocument(bill, kind, upload)
	if err != nil {
		return nil, err
	}
	defer staged.cleanup()
	if err := staged.commit(); err != nil {
		return nil, err
	}

	updated, err := s.debtBills.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if updated != nil {
		s.attachDocumentFlags(updated)
	}
	return updated, nil
}

// DeleteDocument remueve la factura o el comprobante de una cuota pagada
// (SPEC-122).
func (s *DebtService) DeleteDocument(ctx context.Context, id int64, kind DebtDocumentKind) (*models.DebtBill, error) {
	if !validDocumentKind(kind) {
		return nil, fmt.Errorf("tipo de documento inválido")
	}
	bill, err := s.debtBills.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if bill == nil {
		return nil, fmt.Errorf("la cuota no existe")
	}
	if bill.Status != "paid" {
		return nil, fmt.Errorf("la cuota debe estar pagada para remover un documento")
	}

	path := s.documentPath(bill.DebtID, bill.InstallmentNumber, kind)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDocumentNotFound
		}
		return nil, fmt.Errorf("remover documento: %w", err)
	}

	updated, err := s.debtBills.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if updated != nil {
		s.attachDocumentFlags(updated)
	}
	return updated, nil
}

// DocumentPath devuelve la ruta física y el nombre de descarga de una factura
// o comprobante (SPEC-122).
func (s *DebtService) DocumentPath(ctx context.Context, id int64, kind DebtDocumentKind) (filePath, fileName string, err error) {
	if !validDocumentKind(kind) {
		return "", "", ErrDocumentNotFound
	}
	bill, err := s.debtBills.GetByID(ctx, id)
	if err != nil {
		return "", "", err
	}
	if bill == nil || bill.Status != "paid" {
		return "", "", ErrDocumentNotFound
	}

	path := s.documentPath(bill.DebtID, bill.InstallmentNumber, kind)
	info, statErr := os.Stat(path)
	if statErr != nil || info.IsDir() {
		return "", "", ErrDocumentNotFound
	}
	return path, filepath.Base(path), nil
}

// documentDir devuelve el directorio de documentos de una deuda.
func (s *DebtService) documentDir(debtID int64) string {
	return filepath.Join(s.documentsDir, strconv.FormatInt(debtID, 10))
}

// documentPath construye la ruta determinística de una factura o comprobante.
// Solo usa enteros de la base de datos, nunca nombres provistos por el usuario.
func (s *DebtService) documentPath(debtID int64, installment int, kind DebtDocumentKind) string {
	return filepath.Join(s.documentDir(debtID), fmt.Sprintf("%s_%d.pdf", kind, installment))
}

// attachDocumentFlags completa HasInvoice/HasReceipt de una cuota.
func (s *DebtService) attachDocumentFlags(bill *models.DebtBill) {
	if bill == nil {
		return
	}
	bill.HasInvoice = s.hasDocument(bill, DebtDocumentInvoice)
	bill.HasReceipt = s.hasDocument(bill, DebtDocumentReceipt)
}

// attachDocumentFlagsToBills completa los flags de una lista de cuotas.
func (s *DebtService) attachDocumentFlagsToBills(bills []models.DebtBill) {
	for i := range bills {
		s.attachDocumentFlags(&bills[i])
	}
}

// hasDocument verifica si existe el PDF de una cuota pagada.
func (s *DebtService) hasDocument(bill *models.DebtBill, kind DebtDocumentKind) bool {
	if bill == nil || bill.Status != "paid" {
		return false
	}
	info, err := os.Stat(s.documentPath(bill.DebtID, bill.InstallmentNumber, kind))
	return err == nil && !info.IsDir()
}

// hasAnyDocument verifica si el directorio de una deuda contiene al menos una
// factura o comprobante (SPEC-122 REQ-014).
func (s *DebtService) hasAnyDocument(debtID int64) bool {
	entries, err := os.ReadDir(s.documentDir(debtID))
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if !strings.HasSuffix(name, ".pdf") {
			continue
		}
		if strings.HasPrefix(name, string(DebtDocumentInvoice)+"_") ||
			strings.HasPrefix(name, string(DebtDocumentReceipt)+"_") {
			return true
		}
	}
	return false
}

// validDocumentKind valida el tipo de documento.
func validDocumentKind(kind DebtDocumentKind) bool {
	return kind == DebtDocumentInvoice || kind == DebtDocumentReceipt
}

// stagedDocument representa un PDF escrito en un temporal, listo para
// renombrarse de forma atómica al path final.
type stagedDocument struct {
	tmpPath   string
	finalPath string
}

// stageDocument valida el PDF y lo copia en streaming a un temporal dentro del
// directorio destino, sin cargarlo completo en memoria.
func (s *DebtService) stageDocument(bill *models.DebtBill, kind DebtDocumentKind, upload PDFUpload) (*stagedDocument, error) {
	if upload.Reader == nil {
		return nil, fmt.Errorf("el archivo está vacío")
	}
	if strings.ToLower(filepath.Ext(upload.FileName)) != ".pdf" {
		return nil, fmt.Errorf("el documento debe ser un archivo PDF")
	}

	dir := s.documentDir(bill.DebtID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("crear directorio de documentos: %w", err)
	}

	tmp, err := os.CreateTemp(dir, fmt.Sprintf(".%s-*.tmp", kind))
	if err != nil {
		return nil, fmt.Errorf("crear documento temporal: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	written, err := io.Copy(tmp, io.LimitReader(upload.Reader, MaxDebtDocumentBytes+1))
	if err != nil {
		_ = tmp.Close()
		cleanup()
		return nil, fmt.Errorf("guardar documento: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return nil, fmt.Errorf("cerrar documento: %w", err)
	}
	if written == 0 {
		cleanup()
		return nil, fmt.Errorf("el archivo está vacío")
	}
	if written > MaxDebtDocumentBytes {
		cleanup()
		return nil, fmt.Errorf("el documento no puede superar los 10 MB")
	}
	if err := validatePDF(tmpPath); err != nil {
		cleanup()
		return nil, err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		cleanup()
		return nil, fmt.Errorf("aplicar permisos al documento: %w", err)
	}

	return &stagedDocument{
		tmpPath:   tmpPath,
		finalPath: s.documentPath(bill.DebtID, bill.InstallmentNumber, kind),
	}, nil
}

// commit mueve el temporal al nombre final de forma atómica.
func (r *stagedDocument) commit() error {
	if err := os.Rename(r.tmpPath, r.finalPath); err != nil {
		return fmt.Errorf("guardar documento: %w", err)
	}
	r.tmpPath = ""
	return nil
}

// cleanup borra el temporal si todavía existe.
func (r *stagedDocument) cleanup() {
	if r.tmpPath != "" {
		_ = os.Remove(r.tmpPath)
	}
}

// validatePDF verifica la firma de un archivo PDF.
func validatePDF(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("validar documento: %w", err)
	}
	defer file.Close()

	header := make([]byte, 5)
	if _, err := io.ReadFull(file, header); err != nil {
		return fmt.Errorf("el documento no es un PDF válido")
	}
	if !bytes.Equal(header, []byte("%PDF-")) {
		return fmt.Errorf("el documento no es un PDF válido")
	}
	return nil
}

// ReconcileDebtBills genera las cuotas faltantes de todas las deudas activas
// (patrón ReconcileBills de servicios) y finaliza las que ya están completas.
func (s *DebtService) ReconcileDebtBills(ctx context.Context) error {
	debts, err := s.debts.List(ctx)
	if err != nil {
		return err
	}
	for i := range debts {
		if debts[i].Status == DebtStatusActive && debts[i].PaidCount+debts[i].PendingCount < debts[i].InstallmentsTotal {
			if err := s.ensureDebtBills(ctx, &debts[i]); err != nil {
				return err
			}
		}
	}
	// SPEC-104 REQ-002: corrige deudas completas que quedaron en 'activa'.
	if _, err := s.debtBills.FinalizeCompleted(ctx); err != nil {
		return err
	}
	return nil
}

// ensureDebtBills genera la serie completa de cuotas (1..N) que falten para
// una deuda activa. Dedup por (debt_id, installment_number).
func (s *DebtService) ensureDebtBills(ctx context.Context, debt *models.Debt) error {
	if debt.Status != DebtStatusActive {
		return nil
	}
	amount := debt.InstallmentAmount
	if amount <= 0 {
		amount = computeInstallmentAmount(debt.Total, debt.InstallmentsTotal)
	}

	start, err := time.Parse("2006-01-02", debt.StartDate)
	if err != nil {
		return fmt.Errorf("fecha de inicio inválida: %w", err)
	}

	for k := 1; k <= debt.InstallmentsTotal; k++ {
		existing, err := s.debtBills.FindByDebtInstallment(ctx, debt.ID, k)
		if err != nil {
			return err
		}
		dueDate := dueDateForInstallment(start, k, debt.PaymentDay)
		if existing != nil {
			// SPEC-104: si la cuota había sido eliminada por reducir el total,
			// se reutiliza en lugar de chocar con la restricción única.
			if existing.DeletedAt != nil {
				if err := s.debtBills.Restore(ctx, existing.ID, dueDate, amount); err != nil {
					return err
				}
			}
			continue
		}

		bill := &models.DebtBill{
			DebtID:            debt.ID,
			InstallmentNumber: k,
			DueDate:           dueDate,
			Amount:            amount,
			Status:            "pending",
		}
		if _, err := s.debtBills.Create(ctx, bill); err != nil {
			return err
		}
	}
	return nil
}

// computeInstallmentAmount calcula el monto por cuota cuando no se especifica.
func computeInstallmentAmount(total float64, installments int) float64 {
	if installments <= 0 {
		return 0
	}
	return math.Round(total/float64(installments)*100) / 100
}

// dueDateForInstallment calcula el vencimiento de la cuota k (1-indexed):
// día de pago del mes (start + k meses), clampeado al último día del mes.
func dueDateForInstallment(start time.Time, k, paymentDay int) string {
	months := start.Month() + time.Month(k)
	year := start.Year() + (int(months)-1)/12
	month := time.Month((int(months)-1)%12 + 1)

	last := daysInMonth(year, month)
	day := paymentDay
	if day > last {
		day = last
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

func (s *DebtService) validate(ctx context.Context, debt *models.Debt) error {
	debt.Identifier = strings.TrimSpace(debt.Identifier)
	debt.Description = strings.TrimSpace(debt.Description)
	debt.StartDate = strings.TrimSpace(debt.StartDate)
	debt.Status = strings.TrimSpace(debt.Status)
	if debt.Status == "" {
		debt.Status = DebtStatusActive
	}
	if debt.Status != DebtStatusActive && debt.Status != DebtStatusInactive && debt.Status != DebtStatusFinished {
		return fmt.Errorf("el estado debe ser 'activa', 'inactiva' o 'finalizada'")
	}
	if debt.InstitutionID == 0 {
		return fmt.Errorf("debe seleccionar un acreedor")
	}
	if debt.CurrencyID == 0 {
		return fmt.Errorf("debe seleccionar una moneda")
	}
	if debt.Description == "" {
		return fmt.Errorf("la descripción es requerida")
	}
	if debt.Total < 0 || debt.Principal < 0 || debt.InstallmentAmount < 0 || debt.InterestRate < 0 {
		return fmt.Errorf("los montos no pueden ser negativos")
	}
	if debt.InstallmentsTotal < 1 {
		return fmt.Errorf("el total de cuotas debe ser al menos 1")
	}
	if debt.PaymentDay < 1 || debt.PaymentDay > 31 {
		return fmt.Errorf("el día de pago debe estar entre 1 y 31")
	}
	if _, err := time.Parse("2006-01-02", debt.StartDate); err != nil {
		return fmt.Errorf("la fecha de inicio es requerida (formato YYYY-MM-DD)")
	}

	institution, err := s.institutions.GetByID(ctx, debt.InstitutionID)
	if err != nil {
		return fmt.Errorf("validar acreedor: %w", err)
	}
	if institution == nil {
		return fmt.Errorf("el acreedor seleccionado no existe")
	}

	currency, err := s.currencies.GetByID(ctx, debt.CurrencyID)
	if err != nil {
		return fmt.Errorf("validar moneda: %w", err)
	}
	if currency == nil {
		return fmt.Errorf("la moneda seleccionada no existe")
	}

	return nil
}
