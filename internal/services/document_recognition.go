package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/recognizers"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// MaxRecognitionDocumentBytes es el tamaño máximo de un comprobante (SPEC-127).
const MaxRecognitionDocumentBytes int64 = 10 * 1024 * 1024

var (
	ErrRecognitionInvalidFile       = errors.New("el archivo del comprobante no es válido")
	ErrRecognitionTooLarge          = errors.New("el comprobante no puede superar los 10 MB")
	ErrRecognitionNotFound          = errors.New("reconocimiento no encontrado")
	ErrRecognitionAlreadyAssociated = errors.New("el reconocimiento ya fue asociado")
	ErrRecognitionInvalidTarget     = errors.New("la cuota destino no es válida")
	ErrRecognitionBillNotPaid       = errors.New("la cuota destino debe estar pagada")
	ErrRecognitionReferenceConflict = errors.New("la cuota ya tiene otra referencia de pago")
	ErrRecognitionReferenceInUse    = errors.New("la referencia ya está asociada a otra cuota")
	ErrRecognitionFileExists        = errors.New("ya existe un comprobante para esa cuota")
	ErrRecognitionUnsupportedFormat = recognizers.ErrUnsupportedFormat
	ErrRecognitionNeedsOCR          = recognizers.ErrNeedsOCR
	ErrRecognitionInvalidPDF        = recognizers.ErrInvalidPDF
)

var reReferenceValue = regexp.MustCompile(`^[A-Za-z0-9_-]{4,64}$`)

// DocumentRecognitionService orquesta el reconocimiento, el matching y la
// asociación confirmada de comprobantes bancarios (SPEC-127).
type DocumentRecognitionService struct {
	recognitions *storage.DocumentRecognitionStorage
	debtBills    *storage.DebtBillStorage
	documentsDir string
	inboxDir     string
}

// NewDocumentRecognitionService crea el servicio. dataDir es el mismo
// directorio donde vive app.db; los PDFs se guardan en dataDir/documents.
func NewDocumentRecognitionService(
	recognitions *storage.DocumentRecognitionStorage,
	debtBills *storage.DebtBillStorage,
	dataDir string,
) *DocumentRecognitionService {
	if strings.TrimSpace(dataDir) == "" {
		dataDir = "."
	}
	documentsDir := filepath.Join(dataDir, "documents")
	return &DocumentRecognitionService{
		recognitions: recognitions,
		debtBills:    debtBills,
		documentsDir: documentsDir,
		inboxDir:     filepath.Join(documentsDir, "_inbox"),
	}
}

// Recognize valida el PDF, lo guarda en _inbox, extrae los campos y devuelve
// una vista previa. No modifica deuda ni cuota (SPEC-127, REQ-006).
func (s *DocumentRecognitionService) Recognize(ctx context.Context, file io.Reader, header *multipart.FileHeader) (*models.DocumentRecognition, error) {
	if header == nil || strings.ToLower(filepath.Ext(header.Filename)) != ".pdf" {
		return nil, ErrRecognitionInvalidFile
	}
	if header.Size > MaxRecognitionDocumentBytes {
		return nil, ErrRecognitionTooLarge
	}
	if file == nil {
		return nil, ErrRecognitionInvalidFile
	}

	if err := os.MkdirAll(s.inboxDir, 0o755); err != nil {
		return nil, fmt.Errorf("crear directorio temporal de comprobantes: %w", err)
	}

	tmp, err := os.CreateTemp(s.inboxDir, ".recognize-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("crear comprobante temporal: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(tmp, hasher), io.LimitReader(file, MaxRecognitionDocumentBytes+1))
	closeErr := tmp.Close()
	if copyErr != nil {
		cleanup()
		return nil, fmt.Errorf("%w: %v", ErrRecognitionInvalidFile, copyErr)
	}
	if closeErr != nil {
		cleanup()
		return nil, fmt.Errorf("%w: %v", ErrRecognitionInvalidFile, closeErr)
	}
	if written == 0 {
		cleanup()
		return nil, ErrRecognitionInvalidFile
	}
	if written > MaxRecognitionDocumentBytes {
		cleanup()
		return nil, ErrRecognitionTooLarge
	}
	if err := validatePDF(tmpPath); err != nil {
		cleanup()
		return nil, fmt.Errorf("%w: %v", ErrRecognitionInvalidFile, err)
	}

	documentHash := hex.EncodeToString(hasher.Sum(nil))

	existing, err := s.recognitions.GetByHash(ctx, documentHash)
	if err != nil {
		cleanup()
		return nil, err
	}
	if existing != nil {
		existing.Duplicate = true
		if existing.Status == models.RecognitionStatusAssociated {
			cleanup()
			existing.Warnings = appendUnique(existing.Warnings, "documento_ya_asociado")
			return existing, nil
		}
		// Si la vista previa perdió su PDF temporal (por una limpieza o un
		// reinicio), se reutiliza el archivo recién subido.
		if !fileExists(existing.InboxPath) {
			inboxPath := filepath.Join(s.inboxDir, fmt.Sprintf("%d.pdf", existing.ID))
			if err := os.Rename(tmpPath, inboxPath); err != nil {
				cleanup()
				return nil, fmt.Errorf("guardar comprobante temporal: %w", err)
			}
			if err := s.recognitions.SetInboxPath(ctx, existing.ID, inboxPath); err != nil {
				return nil, err
			}
			existing.InboxPath = inboxPath
		} else {
			cleanup()
		}
		return s.decoratePreview(ctx, existing)
	}

	pdfFile, err := os.Open(tmpPath)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("abrir comprobante temporal: %w", err)
	}
	recognized, recognizeErr := recognizers.Recognize(recognizers.Source{
		ReaderAt: pdfFile,
		Size:     written,
		MimeType: "application/pdf",
	})
	_ = pdfFile.Close()
	if recognizeErr != nil {
		cleanup()
		return nil, recognizeErr
	}

	candidates, matchWarnings, confidence, reviewRequired, err := s.match(ctx, recognized)
	if err != nil {
		cleanup()
		return nil, err
	}

	recognition := &models.DocumentRecognition{
		DocumentHash:      documentHash,
		RecognizerID:      recognized.RecognizerID,
		DocumentType:      recognized.DocumentType,
		Status:            models.RecognitionStatusRecognized,
		Reference:         recognized.Reference,
		PaidAt:            recognized.PaidAt,
		Amount:            recognized.Amount,
		Currency:          recognized.Currency,
		InstallmentNumber: recognized.InstallmentNumber,
		ConceptText:       recognized.Concept,
		Confidence:        confidence,
		Warnings:          appendUnique(recognized.Warnings, matchWarnings...),
		ReviewRequired:    reviewRequired,
	}

	created, err := s.recognitions.Create(ctx, recognition)
	if err != nil {
		// Otra request pudo insertar el mismo hash entre la consulta y el
		// insert: se devuelve la fila existente en vez de fallar.
		if duplicate, getErr := s.recognitions.GetByHash(ctx, documentHash); getErr == nil && duplicate != nil {
			cleanup()
			duplicate.Duplicate = true
			return s.decoratePreview(ctx, duplicate)
		}
		cleanup()
		return nil, err
	}

	inboxPath := filepath.Join(s.inboxDir, fmt.Sprintf("%d.pdf", created.ID))
	if err := os.Rename(tmpPath, inboxPath); err != nil {
		_ = s.recognitions.Delete(ctx, created.ID)
		cleanup()
		return nil, fmt.Errorf("guardar comprobante temporal: %w", err)
	}
	if err := s.recognitions.SetInboxPath(ctx, created.ID, inboxPath); err != nil {
		_ = os.Remove(inboxPath)
		_ = s.recognitions.Delete(ctx, created.ID)
		return nil, err
	}

	created.InboxPath = inboxPath
	created.Candidates = candidates
	created.ReviewRequired = reviewRequired
	return created, nil
}

// Confirm asocia el comprobante a una cuota pagada. El archivo se escribe de
// forma atómica y la referencia se actualiza en una transacción (SPEC-127,
// REQ-008). Nunca cambia status ni paid_at de la cuota (REQ-009).
func (s *DocumentRecognitionService) Confirm(ctx context.Context, recognitionID int64, input models.ConfirmRecognitionInput) (*models.AssociationResult, error) {
	recognition, err := s.recognitions.GetByID(ctx, recognitionID)
	if err != nil {
		return nil, err
	}
	if recognition == nil {
		return nil, ErrRecognitionNotFound
	}
	if recognition.Status == models.RecognitionStatusAssociated {
		return nil, ErrRecognitionAlreadyAssociated
	}
	if recognition.Status != models.RecognitionStatusRecognized {
		return nil, fmt.Errorf("%w: estado %s", ErrRecognitionInvalidTarget, recognition.Status)
	}

	entityType := strings.TrimSpace(input.EntityType)
	if entityType == "" {
		entityType = models.RecognitionEntityDebtBill
	}
	if entityType != models.RecognitionEntityDebtBill {
		return nil, ErrRecognitionInvalidTarget
	}

	bill, err := s.resolveDebtBill(ctx, input)
	if err != nil {
		return nil, err
	}

	reference := strings.TrimSpace(input.Reference)
	if !reReferenceValue.MatchString(reference) {
		return nil, fmt.Errorf("%w: referencia inválida", ErrRecognitionInvalidTarget)
	}

	existingReference := strings.TrimSpace(bill.PaymentReference)
	if existingReference != "" && existingReference != reference && !input.OverwriteReference {
		return nil, ErrRecognitionReferenceConflict
	}

	other, err := s.debtBills.FindByPaymentReference(ctx, reference)
	if err != nil {
		return nil, err
	}
	if other != nil && other.ID != bill.ID {
		return nil, ErrRecognitionReferenceInUse
	}

	paidAt, err := parseRecognitionPaidAt(input.PaidAt, recognition.PaidAt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRecognitionInvalidTarget, err)
	}
	amount := input.Amount
	if amount <= 0 {
		amount = recognition.Amount
	}
	currency := normalizeRecognitionCurrency(input.Currency)
	if currency == "" {
		currency = recognition.Currency
	}
	concept := sanitizeRecognitionConcept(input.Concept)
	if concept == "" {
		concept = recognition.ConceptText
	}

	inboxPath := recognition.InboxPath
	if inboxPath == "" {
		inboxPath = filepath.Join(s.inboxDir, fmt.Sprintf("%d.pdf", recognition.ID))
	}
	if !fileExists(inboxPath) {
		return nil, fmt.Errorf("%w: no se encontró el PDF temporal", ErrRecognitionInvalidFile)
	}

	finalPath := filepath.Join(
		s.documentsDir,
		strconv.FormatInt(bill.DebtID, 10),
		fmt.Sprintf("receipt_%d.pdf", bill.InstallmentNumber),
	)
	relativeFinalPath := filepath.ToSlash(filepath.Join(
		"documents",
		strconv.FormatInt(bill.DebtID, 10),
		fmt.Sprintf("receipt_%d.pdf", bill.InstallmentNumber),
	))

	staged, err := s.stageFinalFile(inboxPath, finalPath, input.OverwriteFile)
	if err != nil {
		return nil, err
	}

	if err := s.recognitions.Associate(
		ctx,
		recognition.ID,
		bill.ID,
		reference,
		paidAt,
		amount,
		currency,
		bill.InstallmentNumber,
		concept,
		entityType,
		relativeFinalPath,
	); err != nil {
		staged.rollback()
		switch {
		case errors.Is(err, storage.ErrDebtBillNotPayable):
			return nil, ErrRecognitionBillNotPaid
		case errors.Is(err, storage.ErrRecognitionAlreadyAssociated):
			return nil, ErrRecognitionAlreadyAssociated
		default:
			return nil, err
		}
	}

	staged.cleanup()
	_ = os.Remove(inboxPath)

	return &models.AssociationResult{
		RecognitionID:    recognition.ID,
		Status:           models.RecognitionStatusAssociated,
		DebtBillID:       bill.ID,
		PaymentReference: reference,
		FilePath:         relativeFinalPath,
	}, nil
}

func (s *DocumentRecognitionService) resolveDebtBill(ctx context.Context, input models.ConfirmRecognitionInput) (*models.DebtBill, error) {
	var bill *models.DebtBill
	var err error

	switch {
	case input.DebtBillID > 0:
		bill, err = s.debtBills.GetByID(ctx, input.DebtBillID)
	case input.DebtID > 0 && input.InstallmentNumber > 0:
		bill, err = s.debtBills.FindByDebtInstallment(ctx, input.DebtID, input.InstallmentNumber)
	default:
		return nil, ErrRecognitionInvalidTarget
	}
	if err != nil {
		return nil, err
	}
	if bill == nil || bill.DeletedAt != nil {
		return nil, ErrRecognitionInvalidTarget
	}
	if bill.Status != "paid" {
		return nil, ErrRecognitionBillNotPaid
	}
	if input.DebtID > 0 && bill.DebtID != input.DebtID {
		return nil, ErrRecognitionInvalidTarget
	}
	if input.InstallmentNumber > 0 && bill.InstallmentNumber != input.InstallmentNumber {
		return nil, ErrRecognitionInvalidTarget
	}
	return bill, nil
}

// match busca cuotas pagadas candidatas y calcula confianza y advertencias.
func (s *DocumentRecognitionService) match(
	ctx context.Context,
	doc *recognizers.RecognizedDocument,
) ([]models.RecognitionCandidate, []string, float64, bool, error) {
	bills, err := s.debtBills.ListPaidWithDetails(ctx)
	if err != nil {
		return nil, nil, 0, false, err
	}

	candidates := make([]models.RecognitionCandidate, 0, len(bills))
	for _, bill := range bills {
		candidate, ok := scoreCandidate(doc, bill)
		if ok {
			candidates = append(candidates, candidate)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].DueDate < candidates[j].DueDate
		}
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > 5 {
		candidates = candidates[:5]
	}

	warnings := make([]string, 0, 4)
	reviewRequired := len(candidates) == 0
	confidence := doc.Confidence

	if len(candidates) == 0 {
		warnings = append(warnings, "sin_candidatos")
		// La confianza global incluye la asociación: sin una cuota candidata
		// no debería mostrarse como una coincidencia alta.
		confidence = clampScore(doc.Confidence * 0.5)
	} else {
		best := candidates[0]
		// La confianza global combina el reconocimiento del documento con la
		// calidad de la asociación a una cuota concreta.
		confidence = 0.35*doc.Confidence + 0.65*best.Score
		if best.Score < 0.65 || hasConflictWarning(best.Warnings) || hasCloseCandidates(candidates) {
			reviewRequired = true
		}
		if best.Score < 0.65 {
			warnings = appendUnique(warnings, "coincidencia_parcial")
		}
		if doc.Reference == "" {
			reviewRequired = true
		}
		if doc.InstallmentNumber > 0 &&
			best.InstallmentNumber != doc.InstallmentNumber &&
			containsString(best.Reasons, "referencia") {
			warnings = append(warnings, "detalle_repetido")
			reviewRequired = true
		}
		if hasCloseCandidates(candidates) {
			warnings = append(warnings, "multiples_candidatos")
		}
		for _, warning := range best.Warnings {
			warnings = appendUnique(warnings, warning)
		}
	}

	if doc.Reference != "" {
		other, err := s.debtBills.FindByPaymentReference(ctx, doc.Reference)
		if err != nil {
			return nil, nil, 0, false, err
		}
		if other != nil {
			// Si la referencia ya está en una cuota y el mejor candidato no es
			// esa cuota, hay un conflicto que el usuario debe revisar.
			if len(candidates) == 0 || candidates[0].DebtBillID != other.ID {
				warnings = appendUnique(warnings, "referencia_duplicada")
				reviewRequired = true
			}
		}
	}

	return candidates, warnings, clampScore(confidence), reviewRequired, nil
}

func (s *DocumentRecognitionService) decoratePreview(ctx context.Context, recognition *models.DocumentRecognition) (*models.DocumentRecognition, error) {
	doc := &recognizers.RecognizedDocument{
		DocumentType:      recognition.DocumentType,
		Reference:         recognition.Reference,
		PaidAt:            recognition.PaidAt,
		Amount:            recognition.Amount,
		Currency:          recognition.Currency,
		InstallmentNumber: recognition.InstallmentNumber,
		Concept:           recognition.ConceptText,
		Confidence:        recognition.Confidence,
	}
	candidates, warnings, _, reviewRequired, err := s.match(ctx, doc)
	if err != nil {
		return nil, err
	}
	recognition.Candidates = candidates
	recognition.Warnings = appendUnique(recognition.Warnings, warnings...)
	recognition.ReviewRequired = recognition.ReviewRequired || reviewRequired
	return recognition, nil
}

func scoreCandidate(doc *recognizers.RecognizedDocument, bill models.DebtBill) (models.RecognitionCandidate, bool) {
	candidate := models.RecognitionCandidate{
		EntityType:        models.RecognitionEntityDebtBill,
		DebtID:            bill.DebtID,
		DebtBillID:        bill.ID,
		InstallmentNumber: bill.InstallmentNumber,
		DebtDescription:   bill.DebtDescription,
		InstitutionName:   bill.InstitutionName,
		DueDate:           bill.DueDate,
		Amount:            bill.Amount,
		CurrencyCode:      bill.CurrencyCode,
		Status:            bill.Status,
		ExistingReference: bill.PaymentReference,
	}

	score := 0.0
	reasons := make([]string, 0, 5)
	warnings := make([]string, 0, 3)

	if doc.Reference != "" {
		switch bill.PaymentReference {
		case doc.Reference:
			score += 0.60
			reasons = append(reasons, "referencia")
		case "":
			// Sin referencia previa: no suma ni resta.
		default:
			score -= 0.15
			warnings = append(warnings, "referencia_distinta")
		}
	}

	if doc.Amount > 0 {
		difference := math.Abs(doc.Amount - bill.Amount)
		switch {
		case difference < 0.01:
			score += 0.20
			reasons = append(reasons, "monto")
		case bill.Amount > 0 && difference/bill.Amount <= 0.01:
			score += 0.10
			reasons = append(reasons, "monto_aproximado")
			warnings = append(warnings, "monto_distinto")
		}
	}

	if doc.PaidAt != nil {
		billTime := bill.PaidAt
		if billTime == nil {
			if parsed, parseErr := time.Parse("2006-01-02", bill.DueDate); parseErr == nil {
				billTime = &parsed
			}
		}
		if billTime != nil {
			days := math.Abs(doc.PaidAt.Sub(*billTime).Hours() / 24)
			switch {
			case days <= 1:
				score += 0.15
				reasons = append(reasons, "fecha")
			case days <= 5:
				score += 0.08
				reasons = append(reasons, "fecha_cercana")
			}
		}
	}

	if doc.InstallmentNumber > 0 {
		if doc.InstallmentNumber == bill.InstallmentNumber {
			score += 0.15
			reasons = append(reasons, "cuota")
		} else {
			score -= 0.10
			warnings = append(warnings, "cuota_distinta")
		}
	}

	if doc.Currency != "" && bill.CurrencyCode != "" {
		if strings.EqualFold(doc.Currency, bill.CurrencyCode) {
			score += 0.05
			reasons = append(reasons, "moneda")
		} else {
			warnings = append(warnings, "moneda_distinta")
		}
	}

	if score <= 0 {
		return candidate, false
	}
	candidate.Score = clampScore(score)
	candidate.Reasons = reasons
	candidate.Warnings = warnings
	return candidate, true
}

func hasConflictWarning(warnings []string) bool {
	for _, warning := range warnings {
		switch warning {
		case "cuota_distinta", "referencia_distinta", "monto_distinto":
			return true
		}
	}
	return false
}

func hasCloseCandidates(candidates []models.RecognitionCandidate) bool {
	return len(candidates) > 1 && candidates[0].Score-candidates[1].Score < 0.10
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func appendUnique(values []string, additions ...string) []string {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		seen[value] = true
	}
	for _, value := range additions {
		if value == "" || seen[value] {
			continue
		}
		values = append(values, value)
		seen[value] = true
	}
	return values
}

func clampScore(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return math.Round(value*100) / 100
}

func parseRecognitionPaidAt(value string, fallback *time.Time) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02",
		"02/01/2006",
		"02/01/2006 15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.ParseInLocation(layout, value, time.UTC)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" && fallback != nil {
			combined := time.Date(parsed.Year(), parsed.Month(), parsed.Day(),
				fallback.Hour(), fallback.Minute(), fallback.Second(), 0, fallback.Location())
			return &combined, nil
		}
		return &parsed, nil
	}
	return nil, fmt.Errorf("fecha inválida")
}

func normalizeRecognitionCurrency(value string) string {
	trimmed := strings.TrimSpace(value)
	switch strings.ToUpper(trimmed) {
	case "USD", "US$", "$":
		return "USD"
	case "NIO", "C$":
		return "NIO"
	}
	switch strings.ToLower(trimmed) {
	case "dólares", "dolares":
		return "USD"
	case "córdobas", "cordobas":
		return "NIO"
	}
	if len(trimmed) == 3 {
		return strings.ToUpper(trimmed)
	}
	return ""
}

func sanitizeRecognitionConcept(value string) string {
	return recognizers.SanitizeConcept(value)
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

type recognitionFileCommit struct {
	finalPath  string
	backupPath string
}

func (c *recognitionFileCommit) restoreBackup() {
	if c.backupPath == "" {
		return
	}
	_ = os.Rename(c.backupPath, c.finalPath)
	c.backupPath = ""
}

func (c *recognitionFileCommit) rollback() {
	_ = os.Remove(c.finalPath)
	c.restoreBackup()
}

func (c *recognitionFileCommit) cleanup() {
	if c.backupPath != "" {
		_ = os.Remove(c.backupPath)
		c.backupPath = ""
	}
}

// stageFinalFile copia el PDF temporal al destino final usando un archivo
// temporal en el mismo directorio y un rename atómico. Si ya existe un
// comprobante y overwrite es false, devuelve ErrRecognitionFileExists.
func (s *DocumentRecognitionService) stageFinalFile(inboxPath, finalPath string, overwrite bool) (*recognitionFileCommit, error) {
	if err := os.MkdirAll(filepath.Dir(finalPath), 0o755); err != nil {
		return nil, fmt.Errorf("crear directorio de comprobantes: %w", err)
	}

	commit := &recognitionFileCommit{finalPath: finalPath}
	if info, err := os.Stat(finalPath); err == nil {
		if info.IsDir() {
			return nil, ErrRecognitionFileExists
		}
		if !overwrite {
			return nil, ErrRecognitionFileExists
		}
		backupPath := finalPath + ".bak-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		if err := os.Rename(finalPath, backupPath); err != nil {
			return nil, fmt.Errorf("respaldar comprobante existente: %w", err)
		}
		commit.backupPath = backupPath
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("verificar comprobante existente: %w", err)
	}

	source, err := os.Open(inboxPath)
	if err != nil {
		commit.restoreBackup()
		return nil, fmt.Errorf("abrir comprobante temporal: %w", err)
	}
	defer source.Close()

	tmp, err := os.CreateTemp(filepath.Dir(finalPath), ".receipt-*.tmp")
	if err != nil {
		commit.restoreBackup()
		return nil, fmt.Errorf("crear comprobante temporal: %w", err)
	}
	tmpPath := tmp.Name()
	cleanupTmp := func() { _ = os.Remove(tmpPath) }

	written, copyErr := io.Copy(tmp, io.LimitReader(source, MaxRecognitionDocumentBytes+1))
	closeErr := tmp.Close()
	if copyErr != nil {
		cleanupTmp()
		commit.restoreBackup()
		return nil, fmt.Errorf("copiar comprobante: %w", copyErr)
	}
	if closeErr != nil {
		cleanupTmp()
		commit.restoreBackup()
		return nil, fmt.Errorf("cerrar comprobante: %w", closeErr)
	}
	if written == 0 {
		cleanupTmp()
		commit.restoreBackup()
		return nil, ErrRecognitionInvalidFile
	}
	if written > MaxRecognitionDocumentBytes {
		cleanupTmp()
		commit.restoreBackup()
		return nil, ErrRecognitionTooLarge
	}
	if err := validatePDF(tmpPath); err != nil {
		cleanupTmp()
		commit.restoreBackup()
		return nil, fmt.Errorf("%w: %v", ErrRecognitionInvalidFile, err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		cleanupTmp()
		commit.restoreBackup()
		return nil, fmt.Errorf("aplicar permisos al comprobante: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		cleanupTmp()
		commit.restoreBackup()
		return nil, fmt.Errorf("guardar comprobante: %w", err)
	}

	return commit, nil
}
