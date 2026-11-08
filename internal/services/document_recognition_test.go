package services

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
	"github.com/paulomcnally/p40la-ihost/internal/testpdf"
)

type recognitionTestEnv struct {
	database           *sql.DB
	ctx                context.Context
	debtService        *DebtService
	recognitionService *DocumentRecognitionService
	dataDir            string
	debtID             int64
}

func newRecognitionTestEnv(t *testing.T, installments int) *recognitionTestEnv {
	t.Helper()

	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	ctx := context.Background()
	if _, err := database.ExecContext(ctx, "INSERT INTO institutions (name) VALUES ('Banco Demo')"); err != nil {
		t.Fatalf("insertar institución: %v", err)
	}

	dataDir := t.TempDir()
	debtBillStorage := storage.NewDebtBillStorage(database)
	debtService := NewDebtService(
		storage.NewDebtStorage(database),
		debtBillStorage,
		storage.NewInstitutionStorage(database),
		storage.NewCurrencyStorage(database),
		dataDir,
	)

	debt, err := debtService.Create(ctx, &models.Debt{
		InstitutionID:     1,
		Identifier:        "D-127",
		Description:       "Lote Demo",
		Total:             float64(installments) * 100,
		Principal:         float64(installments) * 100,
		CurrencyID:        2, // USD (sembrado por la migración 0002)
		InstallmentsTotal: installments,
		InstallmentAmount: 100,
		PaymentDay:        5,
		StartDate:         "2026-01-01",
		Status:            DebtStatusActive,
	})
	if err != nil {
		t.Fatalf("crear deuda: %v", err)
	}

	return &recognitionTestEnv{
		database:           database,
		ctx:                ctx,
		debtService:        debtService,
		recognitionService: NewDocumentRecognitionService(storage.NewDocumentRecognitionStorage(database), debtBillStorage, dataDir),
		dataDir:            dataDir,
		debtID:             debt.ID,
	}
}

func recognitionPDF(reference, concept, currency, amount string) []byte {
	return testpdf.Build([]string{
		"Comprobante",
		"Nota de debito",
		"Fecha: 01/02/2026",
		"Hora 09:15:22 (GMT-0600)",
		"Referencia " + reference,
		"Detalle " + concept,
		"Moneda " + currency,
		"Monto " + amount,
	})
}

func recognizeTestPDF(t *testing.T, env *recognitionTestEnv, content []byte) *models.DocumentRecognition {
	t.Helper()
	header := &multipart.FileHeader{Filename: "comprobante.pdf", Size: int64(len(content))}
	recognition, err := env.recognitionService.Recognize(env.ctx, bytes.NewReader(content), header)
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}
	return recognition
}

func debtBills(t *testing.T, env *recognitionTestEnv) []models.DebtBill {
	t.Helper()
	bills, err := env.debtService.ListBills(env.ctx, env.debtID)
	if err != nil {
		t.Fatalf("ListBills: %v", err)
	}
	return bills
}

func payTestBill(t *testing.T, env *recognitionTestEnv, billID int64, paidAt time.Time, reference string) {
	t.Helper()
	if _, err := env.debtService.PayBill(env.ctx, billID, paidAt, reference); err != nil {
		t.Fatalf("PayBill %d: %v", billID, err)
	}
}

func countRecognitions(t *testing.T, env *recognitionTestEnv) int {
	t.Helper()
	var count int
	if err := env.database.QueryRow("SELECT COUNT(*) FROM document_recognitions").Scan(&count); err != nil {
		t.Fatalf("contar reconocimientos: %v", err)
	}
	return count
}

func TestRecognitionConfirmAssociatesReferenceAndReceipt(t *testing.T) {
	env := newRecognitionTestEnv(t, 1)
	bill := debtBills(t, env)[0]
	originalPaidAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	payTestBill(t, env, bill.ID, originalPaidAt, "")

	pdf := recognitionPDF("902000001", "Lote Demo - Cuota 1", "Dolares", "100.00")
	recognition := recognizeTestPDF(t, env, pdf)
	if recognition.Reference != "902000001" {
		t.Fatalf("Reference = %q, want 902000001", recognition.Reference)
	}
	if recognition.Candidates == nil || len(recognition.Candidates) == 0 {
		t.Fatalf("se esperaban candidatos: %+v", recognition)
	}

	result, err := env.recognitionService.Confirm(env.ctx, recognition.ID, models.ConfirmRecognitionInput{
		EntityType:        models.RecognitionEntityDebtBill,
		DebtID:            env.debtID,
		DebtBillID:        bill.ID,
		InstallmentNumber: 1,
		Reference:         "902000001",
		PaidAt:            "2026-02-01",
		Amount:            100,
		Currency:          "USD",
		Concept:           "Lote Demo - Cuota 1",
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if result.PaymentReference != "902000001" {
		t.Errorf("PaymentReference = %q", result.PaymentReference)
	}
	if result.FilePath != "documents/1/receipt_1.pdf" {
		t.Errorf("FilePath = %q, want documents/1/receipt_1.pdf", result.FilePath)
	}

	updated := debtBills(t, env)[0]
	if updated.PaymentReference != "902000001" {
		t.Errorf("payment_reference = %q, want 902000001", updated.PaymentReference)
	}
	if updated.Status != "paid" {
		t.Errorf("status = %q, want paid", updated.Status)
	}
	if updated.PaidAt == nil || !updated.PaidAt.Equal(originalPaidAt) {
		t.Errorf("paid_at cambió: got %v, want %v", updated.PaidAt, originalPaidAt)
	}
	if !updated.HasReceipt {
		t.Error("HasReceipt debería ser true")
	}

	receiptPath := filepath.Join(env.dataDir, "documents", strconv.FormatInt(env.debtID, 10), "receipt_1.pdf")
	written, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("leer comprobante: %v", err)
	}
	if !bytes.Equal(written, pdf) {
		t.Error("el comprobante guardado no coincide con el PDF reconocido")
	}

	var status, finalPath string
	if err := env.database.QueryRow(
		"SELECT status, final_path FROM document_recognitions WHERE id = ?", recognition.ID,
	).Scan(&status, &finalPath); err != nil {
		t.Fatalf("consultar reconocimiento: %v", err)
	}
	if status != models.RecognitionStatusAssociated {
		t.Errorf("status reconocimiento = %q, want associated", status)
	}
	if finalPath != "documents/1/receipt_1.pdf" {
		t.Errorf("final_path = %q", finalPath)
	}

	if _, err := os.Stat(filepath.Join(env.dataDir, "documents", "_inbox", fmt.Sprintf("%d.pdf", recognition.ID))); !os.IsNotExist(err) {
		t.Errorf("el PDF temporal debería haberse eliminado: %v", err)
	}
}

func TestRecognitionDuplicateHashDoesNotCreateAnotherRow(t *testing.T) {
	env := newRecognitionTestEnv(t, 1)
	bill := debtBills(t, env)[0]
	payTestBill(t, env, bill.ID, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "")

	pdf := recognitionPDF("902000001", "Lote Demo - Cuota 1", "Dolares", "100.00")
	first := recognizeTestPDF(t, env, pdf)
	second := recognizeTestPDF(t, env, pdf)

	if first.ID != second.ID {
		t.Errorf("IDs distintos: %d y %d", first.ID, second.ID)
	}
	if !second.Duplicate {
		t.Error("el segundo reconocimiento debería marcarse como duplicado")
	}
	if got := countRecognitions(t, env); got != 1 {
		t.Errorf("reconocimientos = %d, want 1", got)
	}
}

func TestRecognitionConfirmReferenceConflict(t *testing.T) {
	env := newRecognitionTestEnv(t, 1)
	bill := debtBills(t, env)[0]
	payTestBill(t, env, bill.ID, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "900000001")

	recognition := recognizeTestPDF(t, env, recognitionPDF("902000001", "Lote Demo - Cuota 1", "Dolares", "100.00"))

	input := models.ConfirmRecognitionInput{
		EntityType: models.RecognitionEntityDebtBill,
		DebtID:     env.debtID,
		DebtBillID: bill.ID,
		Reference:  "902000001",
	}
	if _, err := env.recognitionService.Confirm(env.ctx, recognition.ID, input); !errors.Is(err, ErrRecognitionReferenceConflict) {
		t.Fatalf("error = %v, want ErrRecognitionReferenceConflict", err)
	}

	input.OverwriteReference = true
	if _, err := env.recognitionService.Confirm(env.ctx, recognition.ID, input); err != nil {
		t.Fatalf("Confirm con overwrite: %v", err)
	}
	if got := debtBills(t, env)[0].PaymentReference; got != "902000001" {
		t.Errorf("payment_reference = %q, want 902000001", got)
	}
}

func TestRecognitionConfirmFileExists(t *testing.T) {
	env := newRecognitionTestEnv(t, 1)
	bill := debtBills(t, env)[0]
	payTestBill(t, env, bill.ID, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "")

	dir := filepath.Join(env.dataDir, "documents", strconv.FormatInt(env.debtID, 10))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("crear directorio: %v", err)
	}
	oldContent := []byte("%PDF-1.4\nold\n%%EOF\n")
	if err := os.WriteFile(filepath.Join(dir, "receipt_1.pdf"), oldContent, 0o644); err != nil {
		t.Fatalf("escribir comprobante previo: %v", err)
	}

	pdf := recognitionPDF("902000001", "Lote Demo - Cuota 1", "Dolares", "100.00")
	recognition := recognizeTestPDF(t, env, pdf)
	input := models.ConfirmRecognitionInput{
		EntityType: models.RecognitionEntityDebtBill,
		DebtID:     env.debtID,
		DebtBillID: bill.ID,
		Reference:  "902000001",
	}
	if _, err := env.recognitionService.Confirm(env.ctx, recognition.ID, input); !errors.Is(err, ErrRecognitionFileExists) {
		t.Fatalf("error = %v, want ErrRecognitionFileExists", err)
	}

	input.OverwriteFile = true
	if _, err := env.recognitionService.Confirm(env.ctx, recognition.ID, input); err != nil {
		t.Fatalf("Confirm con overwrite_file: %v", err)
	}
	written, err := os.ReadFile(filepath.Join(dir, "receipt_1.pdf"))
	if err != nil {
		t.Fatalf("leer comprobante: %v", err)
	}
	if !bytes.Equal(written, pdf) {
		t.Error("el comprobante existente no fue reemplazado")
	}
}

func TestRecognitionReferenceAlreadyInUse(t *testing.T) {
	env := newRecognitionTestEnv(t, 2)
	bills := debtBills(t, env)
	payTestBill(t, env, bills[0].ID, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "902000001")
	payTestBill(t, env, bills[1].ID, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), "")

	recognition := recognizeTestPDF(t, env, recognitionPDF("902000001", "Lote Demo - Cuota 2", "Dolares", "100.00"))
	_, err := env.recognitionService.Confirm(env.ctx, recognition.ID, models.ConfirmRecognitionInput{
		EntityType:         models.RecognitionEntityDebtBill,
		DebtID:             env.debtID,
		DebtBillID:         bills[1].ID,
		Reference:          "902000001",
		OverwriteReference: true,
	})
	if !errors.Is(err, ErrRecognitionReferenceInUse) {
		t.Fatalf("error = %v, want ErrRecognitionReferenceInUse", err)
	}
	if got := debtBills(t, env)[1].PaymentReference; got != "" {
		t.Errorf("la cuota no debería haberse modificado: %q", got)
	}
}

func TestRecognitionDetalleRepetidoRequiresReview(t *testing.T) {
	env := newRecognitionTestEnv(t, 2)
	bills := debtBills(t, env)
	payTestBill(t, env, bills[0].ID, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "900000001")
	payTestBill(t, env, bills[1].ID, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), "900000002")

	// El banco repite "Cuota 1" pero la referencia 900000002 corresponde a la
	// cuota 2. El matching debe priorizar la referencia y marcar revisión.
	recognition := recognizeTestPDF(t, env, recognitionPDF("900000002", "Lote Demo - Cuota 1", "Dolares", "100.00"))
	if len(recognition.Candidates) == 0 {
		t.Fatal("se esperaban candidatos")
	}
	if recognition.Candidates[0].DebtBillID != bills[1].ID {
		t.Errorf("mejor candidato = %d, want %d (cuota 2)", recognition.Candidates[0].DebtBillID, bills[1].ID)
	}
	if !recognition.ReviewRequired {
		t.Error("ReviewRequired debería ser true")
	}
	if !containsString(recognition.Warnings, "detalle_repetido") {
		t.Errorf("Warnings = %v, want detalle_repetido", recognition.Warnings)
	}

	// El reconocimiento no debe asociar ni sobrescribir nada por sí solo.
	current := debtBills(t, env)
	if current[0].PaymentReference != "900000001" || current[1].PaymentReference != "900000002" {
		t.Errorf("las referencias no deberían cambiar: %+v", current)
	}
	if current[0].HasReceipt || current[1].HasReceipt {
		t.Error("no debería existir comprobante antes de confirmar")
	}
}

func TestRecognitionNoCandidates(t *testing.T) {
	env := newRecognitionTestEnv(t, 1)
	recognition := recognizeTestPDF(t, env, recognitionPDF("902000001", "Lote Demo - Cuota 1", "Dolares", "100.00"))
	if len(recognition.Candidates) != 0 {
		t.Errorf("Candidates = %v, want none", recognition.Candidates)
	}
	if !recognition.ReviewRequired {
		t.Error("ReviewRequired debería ser true sin candidatos")
	}
	if !containsString(recognition.Warnings, "sin_candidatos") {
		t.Errorf("Warnings = %v, want sin_candidatos", recognition.Warnings)
	}
	if recognition.Confidence >= 0.8 {
		t.Errorf("Confidence = %v, debería bajar al no haber candidatos", recognition.Confidence)
	}
}

func TestRecognitionDuplicateRecreatesMissingInbox(t *testing.T) {
	env := newRecognitionTestEnv(t, 1)
	pdf := recognitionPDF("902000001", "Lote Demo - Cuota 1", "Dolares", "100.00")

	first := recognizeTestPDF(t, env, pdf)
	inboxPath := filepath.Join(env.dataDir, "documents", "_inbox", fmt.Sprintf("%d.pdf", first.ID))
	if err := os.Remove(inboxPath); err != nil {
		t.Fatalf("eliminar inbox: %v", err)
	}

	second := recognizeTestPDF(t, env, pdf)
	if !second.Duplicate {
		t.Error("el segundo reconocimiento debería ser duplicado")
	}
	if _, err := os.Stat(inboxPath); err != nil {
		t.Errorf("el PDF temporal debería haberse recreado: %v", err)
	}
}

func TestRecognitionInvalidPDF(t *testing.T) {
	env := newRecognitionTestEnv(t, 1)
	content := []byte("%PDF-1.4\nesto no tiene estructura")
	header := &multipart.FileHeader{Filename: "roto.pdf", Size: int64(len(content))}
	_, err := env.recognitionService.Recognize(env.ctx, bytes.NewReader(content), header)
	if !errors.Is(err, ErrRecognitionInvalidPDF) {
		t.Fatalf("error = %v, want ErrRecognitionInvalidPDF", err)
	}
}

func TestRecognitionConfirmTargetMustBePaid(t *testing.T) {
	env := newRecognitionTestEnv(t, 2)
	bills := debtBills(t, env)
	payTestBill(t, env, bills[0].ID, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "")

	recognition := recognizeTestPDF(t, env, recognitionPDF("902000001", "Lote Demo - Cuota 2", "Dolares", "100.00"))
	_, err := env.recognitionService.Confirm(env.ctx, recognition.ID, models.ConfirmRecognitionInput{
		EntityType: models.RecognitionEntityDebtBill,
		DebtID:     env.debtID,
		DebtBillID: bills[1].ID,
		Reference:  "902000001",
	})
	if !errors.Is(err, ErrRecognitionBillNotPaid) {
		t.Fatalf("error = %v, want ErrRecognitionBillNotPaid", err)
	}
}

func TestRecognitionUnsupportedAndNeedsOCR(t *testing.T) {
	env := newRecognitionTestEnv(t, 1)

	unsupported := testpdf.Build([]string{"Factura de otra empresa"})
	header := &multipart.FileHeader{Filename: "factura.pdf", Size: int64(len(unsupported))}
	if _, err := env.recognitionService.Recognize(env.ctx, bytes.NewReader(unsupported), header); !errors.Is(err, ErrRecognitionUnsupportedFormat) {
		t.Errorf("error = %v, want ErrRecognitionUnsupportedFormat", err)
	}

	emptyPDF := testpdf.Build(nil)
	header = &multipart.FileHeader{Filename: "escaneado.pdf", Size: int64(len(emptyPDF))}
	if _, err := env.recognitionService.Recognize(env.ctx, bytes.NewReader(emptyPDF), header); !errors.Is(err, ErrRecognitionNeedsOCR) {
		t.Errorf("error = %v, want ErrRecognitionNeedsOCR", err)
	}

	if got := countRecognitions(t, env); got != 0 {
		t.Errorf("no deberían quedar reconocimientos tras errores: %d", got)
	}
}

func TestRecognitionRejectsInvalidFile(t *testing.T) {
	env := newRecognitionTestEnv(t, 1)
	header := &multipart.FileHeader{Filename: "comprobante.txt", Size: 4}
	if _, err := env.recognitionService.Recognize(env.ctx, bytes.NewReader([]byte("test")), header); !errors.Is(err, ErrRecognitionInvalidFile) {
		t.Fatalf("error = %v, want ErrRecognitionInvalidFile", err)
	}
}
