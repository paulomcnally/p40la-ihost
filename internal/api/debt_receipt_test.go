package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

const testReceiptPDF = "%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n"

func setupDebtReceiptHandlers(t *testing.T) (*DebtHandlers, *services.DebtService, context.Context, string) {
	t.Helper()

	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	if _, err := database.ExecContext(ctx, "INSERT INTO institutions (name) VALUES ('Banco Uno')"); err != nil {
		t.Fatalf("insertar institución: %v", err)
	}

	dataDir := t.TempDir()
	svc := services.NewDebtService(
		storage.NewDebtStorage(database),
		storage.NewDebtBillStorage(database),
		storage.NewInstitutionStorage(database),
		storage.NewCurrencyStorage(database),
		dataDir,
	)
	return NewDebtHandlers(svc), svc, ctx, dataDir
}

func createTestDebt(t *testing.T, svc *services.DebtService, ctx context.Context, installments int) int64 {
	t.Helper()
	debt, err := svc.Create(ctx, &models.Debt{
		InstitutionID:     1,
		Identifier:        "D-122",
		Description:       "Deuda de prueba",
		Total:             100,
		Principal:         100,
		CurrencyID:        1,
		InstallmentsTotal: installments,
		InstallmentAmount: 100,
		PaymentDay:        5,
		StartDate:         "2026-09-01",
		Status:            services.DebtStatusActive,
	})
	if err != nil {
		t.Fatalf("crear deuda: %v", err)
	}
	return debt.ID
}

func firstBillID(t *testing.T, svc *services.DebtService, ctx context.Context, debtID int64) int64 {
	t.Helper()
	bills, err := svc.ListBills(ctx, debtID)
	if err != nil || len(bills) == 0 {
		t.Fatalf("ListBills: got %d, err %v", len(bills), err)
	}
	return bills[0].ID
}

func multipartReceiptBody(t *testing.T, fields map[string]string, fileName string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("campo multipart %s: %v", key, err)
		}
	}
	if fileName != "" {
		part, err := writer.CreateFormFile("file", fileName)
		if err != nil {
			t.Fatalf("crear parte de archivo: %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("escribir archivo: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cerrar multipart: %v", err)
	}
	return &buf, writer.FormDataContentType()
}

func TestPayDebtBillMultipartWithReceipt(t *testing.T) {
	handler, svc, ctx, _ := setupDebtReceiptHandlers(t)
	debtID := createTestDebt(t, svc, ctx, 1)
	billID := firstBillID(t, svc, ctx, debtID)

	body, contentType := multipartReceiptBody(t, map[string]string{
		"paid_at":           "2026-10-09",
		"payment_reference": "demo-reference-001",
	}, "comprobante.pdf", []byte(testReceiptPDF))

	req := httptest.NewRequest(http.MethodPut, "/api/debt-bills/"+strconv.FormatInt(billID, 10)+"/pay", body)
	req.Header.Set("Content-Type", contentType)
	req.SetPathValue("id", strconv.FormatInt(billID, 10))
	rec := httptest.NewRecorder()
	handler.PayDebtBill(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PayDebtBill: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var paid models.DebtBill
	if err := json.Unmarshal(rec.Body.Bytes(), &paid); err != nil {
		t.Fatalf("decodificar respuesta: %v", err)
	}
	if paid.Status != "paid" || !paid.HasReceipt {
		t.Fatalf("cuota pagada inesperada: %+v", paid)
	}

	downloadReq := httptest.NewRequest(http.MethodGet, "/api/debt-bills/"+strconv.FormatInt(billID, 10)+"/receipt", nil)
	downloadReq.SetPathValue("id", strconv.FormatInt(billID, 10))
	downloadRec := httptest.NewRecorder()
	handler.DownloadDebtReceipt(downloadRec, downloadReq)

	if downloadRec.Code != http.StatusOK {
		t.Fatalf("DownloadDebtReceipt: status=%d body=%s", downloadRec.Code, downloadRec.Body.String())
	}
	if got := downloadRec.Header().Get("Content-Type"); got != "application/pdf" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := downloadRec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if !strings.Contains(downloadRec.Body.String(), "%PDF-") {
		t.Errorf("respuesta no contiene el PDF: %q", downloadRec.Body.String())
	}
}

func TestPayDebtBillJSONCompatibility(t *testing.T) {
	handler, svc, ctx, _ := setupDebtReceiptHandlers(t)
	debtID := createTestDebt(t, svc, ctx, 1)
	billID := firstBillID(t, svc, ctx, debtID)

	req := httptest.NewRequest(http.MethodPut, "/api/debt-bills/"+strconv.FormatInt(billID, 10)+"/pay",
		strings.NewReader(`{"paid_at":"2026-10-09","payment_reference":"demo-reference-002"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", strconv.FormatInt(billID, 10))
	rec := httptest.NewRecorder()
	handler.PayDebtBill(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PayDebtBill JSON: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var paid models.DebtBill
	if err := json.Unmarshal(rec.Body.Bytes(), &paid); err != nil {
		t.Fatalf("decodificar respuesta: %v", err)
	}
	if paid.Status != "paid" || paid.HasReceipt {
		t.Fatalf("cuota pagada sin comprobante inesperada: %+v", paid)
	}
}

func TestUploadDebtReceiptAfterPayment(t *testing.T) {
	handler, svc, ctx, _ := setupDebtReceiptHandlers(t)
	debtID := createTestDebt(t, svc, ctx, 1)
	billID := firstBillID(t, svc, ctx, debtID)

	if _, err := svc.PayBill(ctx, billID, time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), ""); err != nil {
		t.Fatalf("PayBill: %v", err)
	}

	body, contentType := multipartReceiptBody(t, nil, "comprobante.pdf", []byte(testReceiptPDF))
	req := httptest.NewRequest(http.MethodPost, "/api/debt-bills/"+strconv.FormatInt(billID, 10)+"/receipt", body)
	req.Header.Set("Content-Type", contentType)
	req.SetPathValue("id", strconv.FormatInt(billID, 10))
	rec := httptest.NewRecorder()
	handler.UploadDebtReceipt(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("UploadDebtReceipt: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated models.DebtBill
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decodificar respuesta: %v", err)
	}
	if !updated.HasReceipt {
		t.Fatalf("UploadDebtReceipt no devolvió has_receipt: %+v", updated)
	}
}

func TestUploadDebtReceiptRequiresPaidBill(t *testing.T) {
	handler, svc, ctx, _ := setupDebtReceiptHandlers(t)
	debtID := createTestDebt(t, svc, ctx, 1)
	billID := firstBillID(t, svc, ctx, debtID)

	body, contentType := multipartReceiptBody(t, nil, "comprobante.pdf", []byte(testReceiptPDF))
	req := httptest.NewRequest(http.MethodPost, "/api/debt-bills/"+strconv.FormatInt(billID, 10)+"/receipt", body)
	req.Header.Set("Content-Type", contentType)
	req.SetPathValue("id", strconv.FormatInt(billID, 10))
	rec := httptest.NewRecorder()
	handler.UploadDebtReceipt(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("UploadDebtReceipt impaga: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "not_paid") {
		t.Errorf("se esperaba error not_paid: %s", rec.Body.String())
	}
}

func TestDownloadDebtReceiptNotFound(t *testing.T) {
	handler, svc, ctx, _ := setupDebtReceiptHandlers(t)
	debtID := createTestDebt(t, svc, ctx, 1)
	billID := firstBillID(t, svc, ctx, debtID)

	req := httptest.NewRequest(http.MethodGet, "/api/debt-bills/"+strconv.FormatInt(billID, 10)+"/receipt", nil)
	req.SetPathValue("id", strconv.FormatInt(billID, 10))
	rec := httptest.NewRecorder()
	handler.DownloadDebtReceipt(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("DownloadDebtReceipt sin archivo: status=%d body=%s", rec.Code, rec.Body.String())
	}
}
