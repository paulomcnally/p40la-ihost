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

const testDocumentPDF = "%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n"

type multipartFile struct {
	name    string
	content []byte
}

func setupDebtDocumentHandlers(t *testing.T) (*DebtHandlers, *services.DebtService, context.Context) {
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

	svc := services.NewDebtService(
		storage.NewDebtStorage(database),
		storage.NewDebtBillStorage(database),
		storage.NewInstitutionStorage(database),
		storage.NewCurrencyStorage(database),
		t.TempDir(),
	)
	return NewDebtHandlers(svc), svc, ctx
}

func createDocumentTestDebt(t *testing.T, svc *services.DebtService, ctx context.Context) int64 {
	t.Helper()
	debt, err := svc.Create(ctx, &models.Debt{
		InstitutionID:     1,
		Identifier:        "D-122",
		Description:       "Deuda de prueba",
		Total:             100,
		Principal:         100,
		CurrencyID:        1,
		InstallmentsTotal: 1,
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

func firstDocumentBillID(t *testing.T, svc *services.DebtService, ctx context.Context, debtID int64) int64 {
	t.Helper()
	bills, err := svc.ListBills(ctx, debtID)
	if err != nil || len(bills) == 0 {
		t.Fatalf("ListBills: got %d, err %v", len(bills), err)
	}
	return bills[0].ID
}

func multipartDocumentBody(t *testing.T, fields map[string]string, files map[string]multipartFile) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("campo multipart %s: %v", key, err)
		}
	}
	for field, file := range files {
		if file.name == "" {
			continue
		}
		part, err := writer.CreateFormFile(field, file.name)
		if err != nil {
			t.Fatalf("crear parte %s: %v", field, err)
		}
		if _, err := part.Write(file.content); err != nil {
			t.Fatalf("escribir %s: %v", field, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cerrar multipart: %v", err)
	}
	return &buf, writer.FormDataContentType()
}

func TestPayDebtBillMultipartWithBothDocuments(t *testing.T) {
	handler, svc, ctx := setupDebtDocumentHandlers(t)
	debtID := createDocumentTestDebt(t, svc, ctx)
	billID := firstDocumentBillID(t, svc, ctx, debtID)

	body, contentType := multipartDocumentBody(t, map[string]string{
		"paid_at":           "2026-10-09",
		"payment_reference": "demo-reference-001",
	}, map[string]multipartFile{
		"invoice": {name: "factura.pdf", content: []byte(testDocumentPDF)},
		"receipt": {name: "comprobante.pdf", content: []byte(testDocumentPDF)},
	})

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
	if paid.Status != "paid" || !paid.HasInvoice || !paid.HasReceipt {
		t.Fatalf("cuota pagada inesperada: %+v", paid)
	}

	for _, tc := range []struct {
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"/invoice", handler.DownloadDebtInvoice},
		{"/receipt", handler.DownloadDebtReceipt},
	} {
		downloadReq := httptest.NewRequest(http.MethodGet, "/api/debt-bills/"+strconv.FormatInt(billID, 10)+tc.path, nil)
		downloadReq.SetPathValue("id", strconv.FormatInt(billID, 10))
		downloadRec := httptest.NewRecorder()
		tc.handler(downloadRec, downloadReq)

		if downloadRec.Code != http.StatusOK {
			t.Fatalf("download %s: status=%d body=%s", tc.path, downloadRec.Code, downloadRec.Body.String())
		}
		if got := downloadRec.Header().Get("Content-Type"); got != "application/pdf" {
			t.Errorf("Content-Type %s = %q", tc.path, got)
		}
		if got := downloadRec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("X-Content-Type-Options %s = %q", tc.path, got)
		}
		if !strings.Contains(downloadRec.Body.String(), "%PDF-") {
			t.Errorf("respuesta %s no contiene el PDF: %q", tc.path, downloadRec.Body.String())
		}
	}
}

func TestPayDebtBillJSONCompatibility(t *testing.T) {
	handler, svc, ctx := setupDebtDocumentHandlers(t)
	debtID := createDocumentTestDebt(t, svc, ctx)
	billID := firstDocumentBillID(t, svc, ctx, debtID)

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
	if paid.Status != "paid" || paid.HasInvoice || paid.HasReceipt {
		t.Fatalf("cuota pagada sin documentos inesperada: %+v", paid)
	}
}

func TestUploadDebtDocumentsAfterPayment(t *testing.T) {
	handler, svc, ctx := setupDebtDocumentHandlers(t)
	debtID := createDocumentTestDebt(t, svc, ctx)
	billID := firstDocumentBillID(t, svc, ctx, debtID)

	if _, err := svc.PayBill(ctx, billID, time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), ""); err != nil {
		t.Fatalf("PayBill: %v", err)
	}

	for _, tc := range []struct {
		path        string
		handler     func(http.ResponseWriter, *http.Request)
		wantInvoice bool
		wantReceipt bool
	}{
		{"/invoice", handler.UploadDebtInvoice, true, false},
		{"/receipt", handler.UploadDebtReceipt, true, true},
	} {
		body, contentType := multipartDocumentBody(t, nil, map[string]multipartFile{
			"file": {name: "documento.pdf", content: []byte(testDocumentPDF)},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/debt-bills/"+strconv.FormatInt(billID, 10)+tc.path, body)
		req.Header.Set("Content-Type", contentType)
		req.SetPathValue("id", strconv.FormatInt(billID, 10))
		rec := httptest.NewRecorder()
		tc.handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("upload %s: status=%d body=%s", tc.path, rec.Code, rec.Body.String())
		}
		var updated models.DebtBill
		if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
			t.Fatalf("decodificar respuesta %s: %v", tc.path, err)
		}
		if updated.HasInvoice != tc.wantInvoice || updated.HasReceipt != tc.wantReceipt {
			t.Fatalf("flags tras %s: %+v", tc.path, updated)
		}
	}
}

func TestUploadDebtDocumentRequiresPaidBill(t *testing.T) {
	handler, svc, ctx := setupDebtDocumentHandlers(t)
	debtID := createDocumentTestDebt(t, svc, ctx)
	billID := firstDocumentBillID(t, svc, ctx, debtID)

	body, contentType := multipartDocumentBody(t, nil, map[string]multipartFile{
		"file": {name: "factura.pdf", content: []byte(testDocumentPDF)},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/debt-bills/"+strconv.FormatInt(billID, 10)+"/invoice", body)
	req.Header.Set("Content-Type", contentType)
	req.SetPathValue("id", strconv.FormatInt(billID, 10))
	rec := httptest.NewRecorder()
	handler.UploadDebtInvoice(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("UploadDebtInvoice impaga: status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "not_paid") {
		t.Errorf("se esperaba error not_paid: %s", rec.Body.String())
	}
}

func TestDeleteDebtDocuments(t *testing.T) {
	handler, svc, ctx := setupDebtDocumentHandlers(t)
	debtID := createDocumentTestDebt(t, svc, ctx)
	billID := firstDocumentBillID(t, svc, ctx, debtID)

	if _, err := svc.PayBillWithDocuments(
		ctx,
		billID,
		time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
		"",
		&services.PDFUpload{FileName: "factura.pdf", Reader: strings.NewReader(testDocumentPDF)},
		&services.PDFUpload{FileName: "comprobante.pdf", Reader: strings.NewReader(testDocumentPDF)},
	); err != nil {
		t.Fatalf("PayBillWithDocuments: %v", err)
	}

	for _, tc := range []struct {
		path        string
		handler     func(http.ResponseWriter, *http.Request)
		wantInvoice bool
		wantReceipt bool
	}{
		{"/invoice", handler.DeleteDebtInvoice, false, true},
		{"/receipt", handler.DeleteDebtReceipt, false, false},
	} {
		req := httptest.NewRequest(http.MethodDelete, "/api/debt-bills/"+strconv.FormatInt(billID, 10)+tc.path, nil)
		req.SetPathValue("id", strconv.FormatInt(billID, 10))
		rec := httptest.NewRecorder()
		tc.handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("delete %s: status=%d body=%s", tc.path, rec.Code, rec.Body.String())
		}
		var updated models.DebtBill
		if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
			t.Fatalf("decodificar respuesta %s: %v", tc.path, err)
		}
		if updated.HasInvoice != tc.wantInvoice || updated.HasReceipt != tc.wantReceipt {
			t.Fatalf("flags tras delete %s: %+v", tc.path, updated)
		}
	}

	// Borrar de nuevo un documento inexistente devuelve 404.
	req := httptest.NewRequest(http.MethodDelete, "/api/debt-bills/"+strconv.FormatInt(billID, 10)+"/invoice", nil)
	req.SetPathValue("id", strconv.FormatInt(billID, 10))
	rec := httptest.NewRecorder()
	handler.DeleteDebtInvoice(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("delete inexistente: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestDownloadDebtDocumentsNotFound(t *testing.T) {
	handler, svc, ctx := setupDebtDocumentHandlers(t)
	debtID := createDocumentTestDebt(t, svc, ctx)
	billID := firstDocumentBillID(t, svc, ctx, debtID)

	for _, tc := range []struct {
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"/invoice", handler.DownloadDebtInvoice},
		{"/receipt", handler.DownloadDebtReceipt},
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/debt-bills/"+strconv.FormatInt(billID, 10)+tc.path, nil)
		req.SetPathValue("id", strconv.FormatInt(billID, 10))
		rec := httptest.NewRecorder()
		tc.handler(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("download %s sin archivo: status=%d body=%s", tc.path, rec.Code, rec.Body.String())
		}
	}
}
