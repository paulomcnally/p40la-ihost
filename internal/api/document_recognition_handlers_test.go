package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
	"github.com/paulomcnally/p40la-ihost/internal/testpdf"
)

type recognitionAPIEnv struct {
	handler            *DocumentRecognitionHandlers
	recognitionService *services.DocumentRecognitionService
	debtService        *services.DebtService
	database           *sql.DB
	ctx                context.Context
	debtID             int64
	dataDir            string
}

func newRecognitionAPIEnv(t *testing.T, installments int) *recognitionAPIEnv {
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
	debtService := services.NewDebtService(
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
		CurrencyID:        2,
		InstallmentsTotal: installments,
		InstallmentAmount: 100,
		PaymentDay:        5,
		StartDate:         "2026-01-01",
		Status:            services.DebtStatusActive,
	})
	if err != nil {
		t.Fatalf("crear deuda: %v", err)
	}

	recognitionService := services.NewDocumentRecognitionService(
		storage.NewDocumentRecognitionStorage(database),
		debtBillStorage,
		dataDir,
	)

	return &recognitionAPIEnv{
		handler:            NewDocumentRecognitionHandlers(recognitionService),
		recognitionService: recognitionService,
		debtService:        debtService,
		database:           database,
		ctx:                ctx,
		debtID:             debt.ID,
		dataDir:            dataDir,
	}
}

func apiRecognitionPDF(reference, concept, amount string) []byte {
	return testpdf.Build([]string{
		"Comprobante",
		"Nota de debito",
		"Fecha: 01/02/2026",
		"Hora 09:15:22 (GMT-0600)",
		"Referencia " + reference,
		"Detalle " + concept,
		"Moneda Dolares",
		"Monto " + amount,
	})
}

func recognitionMultipartBody(t *testing.T, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("crear parte multipart: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("escribir multipart: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cerrar multipart: %v", err)
	}
	return &buf, writer.FormDataContentType()
}

func doRecognizeRequest(t *testing.T, env *recognitionAPIEnv, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := recognitionMultipartBody(t, filename, content)
	req := httptest.NewRequest(http.MethodPost, "/api/documents/recognize", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	env.handler.Recognize(rec, req)
	return rec
}

func doConfirmRequest(t *testing.T, env *recognitionAPIEnv, recognitionID int64, input models.ConfirmRecognitionInput) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal confirm: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost,
		"/api/documents/recognitions/"+strconv.FormatInt(recognitionID, 10)+"/confirm",
		bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.FormatInt(recognitionID, 10))
	rec := httptest.NewRecorder()
	env.handler.Confirm(rec, req)
	return rec
}

func TestDocumentRecognitionHandlersRecognizeAndConfirm(t *testing.T) {
	env := newRecognitionAPIEnv(t, 1)
	bills, err := env.debtService.ListBills(env.ctx, env.debtID)
	if err != nil || len(bills) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(bills), err)
	}
	if _, err := env.debtService.PayBill(env.ctx, bills[0].ID, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), ""); err != nil {
		t.Fatalf("PayBill: %v", err)
	}

	recognizeRec := doRecognizeRequest(t, env, "comprobante.pdf", apiRecognitionPDF("902000001", "Lote Demo - Cuota 1", "100.00"))
	if recognizeRec.Code != http.StatusOK {
		t.Fatalf("Recognize: status=%d body=%s", recognizeRec.Code, recognizeRec.Body.String())
	}
	var recognition models.DocumentRecognition
	if err := json.Unmarshal(recognizeRec.Body.Bytes(), &recognition); err != nil {
		t.Fatalf("decodificar reconocimiento: %v", err)
	}
	if recognition.ID == 0 || recognition.Reference != "902000001" {
		t.Fatalf("reconocimiento inesperado: %+v", recognition)
	}

	confirmRec := doConfirmRequest(t, env, recognition.ID, models.ConfirmRecognitionInput{
		EntityType:        models.RecognitionEntityDebtBill,
		DebtID:            env.debtID,
		DebtBillID:        bills[0].ID,
		InstallmentNumber: 1,
		Reference:         "902000001",
		PaidAt:            "2026-02-01",
		Amount:            100,
		Currency:          "USD",
	})
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("Confirm: status=%d body=%s", confirmRec.Code, confirmRec.Body.String())
	}
	var result models.AssociationResult
	if err := json.Unmarshal(confirmRec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decodificar asociación: %v", err)
	}
	if result.Status != models.RecognitionStatusAssociated {
		t.Errorf("Status = %q, want associated", result.Status)
	}
	if result.FilePath != "documents/1/receipt_1.pdf" {
		t.Errorf("FilePath = %q", result.FilePath)
	}

	current, err := env.debtService.ListBills(env.ctx, env.debtID)
	if err != nil || len(current) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(current), err)
	}
	if current[0].PaymentReference != "902000001" || !current[0].HasReceipt {
		t.Errorf("cuota tras confirmar: %+v", current[0])
	}

	// Un segundo confirm debe devolver 409 already_associated.
	again := doConfirmRequest(t, env, recognition.ID, models.ConfirmRecognitionInput{
		EntityType: models.RecognitionEntityDebtBill,
		DebtID:     env.debtID,
		DebtBillID: bills[0].ID,
		Reference:  "902000001",
	})
	if again.Code != http.StatusConflict {
		t.Fatalf("segundo confirm: status=%d body=%s", again.Code, again.Body.String())
	}
	var errorBody map[string]string
	_ = json.Unmarshal(again.Body.Bytes(), &errorBody)
	if errorBody["error"] != "already_associated" {
		t.Errorf("error = %q, want already_associated", errorBody["error"])
	}
}

func TestDocumentRecognitionHandlersErrorCodes(t *testing.T) {
	env := newRecognitionAPIEnv(t, 1)

	cases := []struct {
		name     string
		filename string
		content  []byte
		want     int
		code     string
	}{
		{
			name:     "formato no soportado",
			filename: "factura.pdf",
			content:  testpdf.Build([]string{"Factura de otra empresa"}),
			want:     http.StatusBadRequest,
			code:     "unsupported_format",
		},
		{
			name:     "pdf sin texto",
			filename: "escaneado.pdf",
			content:  testpdf.Build(nil),
			want:     http.StatusBadRequest,
			code:     "needs_ocr",
		},
		{
			name:     "archivo no pdf",
			filename: "comprobante.txt",
			content:  []byte("no es un pdf"),
			want:     http.StatusBadRequest,
			code:     "invalid_file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRecognizeRequest(t, env, tc.filename, tc.content)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tc.want, rec.Body.String())
			}
			var body map[string]string
			_ = json.Unmarshal(rec.Body.Bytes(), &body)
			if body["error"] != tc.code {
				t.Errorf("error = %q, want %q", body["error"], tc.code)
			}
		})
	}
}

func TestDocumentRecognitionHandlersReferenceConflict(t *testing.T) {
	env := newRecognitionAPIEnv(t, 1)
	bills, err := env.debtService.ListBills(env.ctx, env.debtID)
	if err != nil || len(bills) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(bills), err)
	}
	if _, err := env.debtService.PayBill(env.ctx, bills[0].ID, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "900000001"); err != nil {
		t.Fatalf("PayBill: %v", err)
	}

	recognizeRec := doRecognizeRequest(t, env, "comprobante.pdf", apiRecognitionPDF("902000001", "Lote Demo - Cuota 1", "100.00"))
	if recognizeRec.Code != http.StatusOK {
		t.Fatalf("Recognize: status=%d body=%s", recognizeRec.Code, recognizeRec.Body.String())
	}
	var recognition models.DocumentRecognition
	if err := json.Unmarshal(recognizeRec.Body.Bytes(), &recognition); err != nil {
		t.Fatalf("decodificar reconocimiento: %v", err)
	}

	// Sin overwrite_reference debe fallar con 400 reference_conflict.
	conflict := doConfirmRequest(t, env, recognition.ID, models.ConfirmRecognitionInput{
		EntityType: models.RecognitionEntityDebtBill,
		DebtID:     env.debtID,
		DebtBillID: bills[0].ID,
		Reference:  "902000001",
	})
	if conflict.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%s)", conflict.Code, conflict.Body.String())
	}
	var body map[string]string
	_ = json.Unmarshal(conflict.Body.Bytes(), &body)
	if body["error"] != "reference_conflict" {
		t.Errorf("error = %q, want reference_conflict", body["error"])
	}

	// Con overwrite_reference debe asociar.
	ok := doConfirmRequest(t, env, recognition.ID, models.ConfirmRecognitionInput{
		EntityType:         models.RecognitionEntityDebtBill,
		DebtID:             env.debtID,
		DebtBillID:         bills[0].ID,
		Reference:          "902000001",
		OverwriteReference: true,
	})
	if ok.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", ok.Code, ok.Body.String())
	}
}

func TestDocumentRecognitionHandlersFileExists(t *testing.T) {
	env := newRecognitionAPIEnv(t, 1)
	bills, err := env.debtService.ListBills(env.ctx, env.debtID)
	if err != nil || len(bills) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(bills), err)
	}
	if _, err := env.debtService.PayBill(env.ctx, bills[0].ID, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), ""); err != nil {
		t.Fatalf("PayBill: %v", err)
	}
	if _, err := env.debtService.SaveDocument(env.ctx, bills[0].ID, services.DebtDocumentReceipt, services.PDFUpload{
		FileName: "comprobante.pdf",
		Reader:   bytes.NewReader([]byte("%PDF-1.4\nold\n%%EOF\n")),
	}); err != nil {
		t.Fatalf("SaveDocument: %v", err)
	}

	recognizeRec := doRecognizeRequest(t, env, "comprobante.pdf", apiRecognitionPDF("902000001", "Lote Demo - Cuota 1", "100.00"))
	if recognizeRec.Code != http.StatusOK {
		t.Fatalf("Recognize: status=%d body=%s", recognizeRec.Code, recognizeRec.Body.String())
	}
	var recognition models.DocumentRecognition
	if err := json.Unmarshal(recognizeRec.Body.Bytes(), &recognition); err != nil {
		t.Fatalf("decodificar reconocimiento: %v", err)
	}

	conflict := doConfirmRequest(t, env, recognition.ID, models.ConfirmRecognitionInput{
		EntityType: models.RecognitionEntityDebtBill,
		DebtID:     env.debtID,
		DebtBillID: bills[0].ID,
		Reference:  "902000001",
	})
	if conflict.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body=%s)", conflict.Code, conflict.Body.String())
	}
	var body map[string]string
	_ = json.Unmarshal(conflict.Body.Bytes(), &body)
	if body["error"] != "file_exists" {
		t.Errorf("error = %q, want file_exists", body["error"])
	}

	ok := doConfirmRequest(t, env, recognition.ID, models.ConfirmRecognitionInput{
		EntityType:    models.RecognitionEntityDebtBill,
		DebtID:        env.debtID,
		DebtBillID:    bills[0].ID,
		Reference:     "902000001",
		OverwriteFile: true,
	})
	if ok.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", ok.Code, ok.Body.String())
	}
}
