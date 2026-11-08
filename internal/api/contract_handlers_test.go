package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func contractPDF(body string) []byte {
	return []byte("%PDF-1.4\n" + body + "\n%%EOF\n")
}

func multipartContractRequest(t *testing.T, method, target, id, fileName string, data []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if fileName != "" {
		part, err := writer.CreateFormFile("file", fileName)
		if err != nil {
			t.Fatalf("crear form file: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("escribir archivo: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cerrar multipart: %v", err)
	}
	req := httptest.NewRequest(method, target, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("id", id)
	return req
}

func contractRequest(method, target, id string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	req.SetPathValue("id", id)
	return req
}

func TestServiceContractHandlers(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	homeStorage := storage.NewHomeStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	serviceStorage := storage.NewServiceStorage(database)
	homeSvc := services.NewHomeService(homeStorage)
	currencySvc := services.NewCurrencyService(currencyStorage)
	serviceSvc := services.NewServiceService(serviceStorage, homeStorage, currencyStorage, storage.NewBillStorage(database))
	store := services.NewContractStore(t.TempDir())
	serviceSvc.SetContractStore(store)
	handlers := NewServiceHandlers(serviceSvc, homeSvc, storage.NewInstitutionStorage(database), nil)

	home, err := homeSvc.Create(ctx, "Casa Demo", "")
	if err != nil {
		t.Fatalf("crear casa: %v", err)
	}
	currencies, err := currencySvc.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("monedas: %v", err)
	}
	svc, err := serviceSvc.Create(ctx, &models.Service{
		HomeID:          home.ID,
		Name:            "Servicio Demo",
		CurrencyID:      currencies[0].ID,
		Frequency:       "monthly",
		SuggestedAmount: 100,
		Active:          true,
		IconKey:         "internet",
		BillingType:     "fixed",
	})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	id := strconv.FormatInt(svc.ID, 10)
	pdf := contractPDF("servicio")

	// Upload válido.
	req := multipartContractRequest(t, http.MethodPost, "/api/services/"+id+"/contract", id, "contrato.pdf", pdf)
	rec := httptest.NewRecorder()
	handlers.UploadContract(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload: status %d, body %s", rec.Code, rec.Body.String())
	}
	var payload map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("parsear upload: %v", err)
	}
	if !payload["has_contract"] {
		t.Error("upload debería devolver has_contract=true")
	}
	if !store.Exists(services.ContractOwnerService, svc.ID) {
		t.Fatal("el contrato no se guardó en el store")
	}

	// Download inline.
	req = contractRequest(http.MethodGet, "/api/services/"+id+"/contract", id)
	rec = httptest.NewRecorder()
	handlers.DownloadContract(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("download: status %d, body %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/pdf" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("nosniff = %q", got)
	}
	if !bytes.Equal(rec.Body.Bytes(), pdf) {
		t.Error("el PDF descargado no coincide")
	}

	// Download como adjunto.
	req = contractRequest(http.MethodGet, "/api/services/"+id+"/contract?download=1", id)
	rec = httptest.NewRecorder()
	handlers.DownloadContract(rec, req)
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="contract.pdf"` {
		t.Errorf("Content-Disposition = %q", got)
	}

	// Delete.
	req = contractRequest(http.MethodDelete, "/api/services/"+id+"/contract", id)
	rec = httptest.NewRecorder()
	handlers.DeleteContract(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: status %d, body %s", rec.Code, rec.Body.String())
	}
	req = contractRequest(http.MethodGet, "/api/services/"+id+"/contract", id)
	rec = httptest.NewRecorder()
	handlers.DownloadContract(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("download sin contrato: status %d, want 404", rec.Code)
	}

	// Archivo inválido y entidad inexistente.
	req = multipartContractRequest(t, http.MethodPost, "/api/services/"+id+"/contract", id, "contrato.txt", pdf)
	rec = httptest.NewRecorder()
	handlers.UploadContract(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("upload no PDF: status %d, want 400", rec.Code)
	}

	req = multipartContractRequest(t, http.MethodPost, "/api/services/99999/contract", "99999", "contrato.pdf", pdf)
	rec = httptest.NewRecorder()
	handlers.UploadContract(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("upload entidad inexistente: status %d, want 404", rec.Code)
	}

	req = multipartContractRequest(t, http.MethodPost, "/api/services/"+id+"/contract", id, "", nil)
	rec = httptest.NewRecorder()
	handlers.UploadContract(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("upload sin archivo: status %d, want 400", rec.Code)
	}

	big := append([]byte("%PDF-"), bytes.Repeat([]byte("A"), services.MaxContractSize)...)
	req = multipartContractRequest(t, http.MethodPost, "/api/services/"+id+"/contract", id, "contrato.pdf", big)
	rec = httptest.NewRecorder()
	handlers.UploadContract(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("upload demasiado grande: status %d, want 413", rec.Code)
	}
}

func TestDebtContractHandlers(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	if _, err := database.ExecContext(ctx, "INSERT INTO institutions (name) VALUES ('Proveedor Demo')"); err != nil {
		t.Fatalf("insertar institución: %v", err)
	}
	dataDir := t.TempDir()
	debtSvc := services.NewDebtService(
		storage.NewDebtStorage(database),
		storage.NewDebtBillStorage(database),
		storage.NewInstitutionStorage(database),
		storage.NewCurrencyStorage(database),
		dataDir,
	)
	handlers := NewDebtHandlers(debtSvc)

	debt, err := debtSvc.Create(ctx, &models.Debt{
		InstitutionID:     1,
		Identifier:        "demo-account-001",
		Description:       "Deuda Demo",
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
	id := strconv.FormatInt(debt.ID, 10)
	pdf := contractPDF("deuda")

	req := multipartContractRequest(t, http.MethodPost, "/api/debts/"+id+"/contract", id, "contrato.pdf", pdf)
	rec := httptest.NewRecorder()
	handlers.UploadContract(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload: status %d, body %s", rec.Code, rec.Body.String())
	}

	req = contractRequest(http.MethodGet, "/api/debts/"+id+"/contract", id)
	rec = httptest.NewRecorder()
	handlers.DownloadContract(rec, req)
	if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), pdf) {
		t.Fatalf("download: status %d, body coincide=%v", rec.Code, bytes.Equal(rec.Body.Bytes(), pdf))
	}

	req = contractRequest(http.MethodDelete, "/api/debts/"+id+"/contract", id)
	rec = httptest.NewRecorder()
	handlers.DeleteContract(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: status %d, body %s", rec.Code, rec.Body.String())
	}

	req = multipartContractRequest(t, http.MethodPost, "/api/debts/99999/contract", "99999", "contrato.pdf", pdf)
	rec = httptest.NewRecorder()
	handlers.UploadContract(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("upload deuda inexistente: status %d, want 404", rec.Code)
	}
}
