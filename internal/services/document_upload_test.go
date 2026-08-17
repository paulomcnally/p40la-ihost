package services

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/analyzers"
	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type fakeAnalyzer struct{}

func (a *fakeAnalyzer) Info() analyzers.AnalyzerInfo {
	return analyzers.AnalyzerInfo{ID: "fake-analyzer", Name: "Fake Analyzer"}
}

func (a *fakeAnalyzer) Analyze(reader io.Reader, mimeType string) (*analyzers.ExtractedBill, error) {
	if _, err := io.ReadAll(reader); err != nil {
		return nil, err
	}
	return &analyzers.ExtractedBill{Amount: 99, InvoiceNumber: "FAKE1", Year: 2024, Month: 7}, nil
}

func TestUploadAndAnalyzeComputesFileHash(t *testing.T) {
	analyzers.Register(&fakeAnalyzer{})

	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	currencyStorage := storage.NewCurrencyStorage(database)
	homeStorage := storage.NewHomeStorage(database)
	serviceStorage := storage.NewServiceStorage(database)
	billStorage := storage.NewBillStorage(database)
	institutionStorage := storage.NewInstitutionStorage(database)

	docSvc := NewDocumentService(serviceStorage, billStorage, institutionStorage)
	currencySvc := NewCurrencyService(currencyStorage)
	homeSvc := NewHomeService(homeStorage)
	serviceSvc := NewServiceService(serviceStorage, homeStorage, currencyStorage, billStorage)
	institutionSvc := NewInstitutionService(institutionStorage)

	home, err := homeSvc.Create(ctx, "Casa Test", "")
	if err != nil {
		t.Fatalf("crear hogar: %v", err)
	}
	currencies, err := currencySvc.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}
	inst, err := institutionSvc.Create(ctx, &models.Institution{Name: "Fake Inst"})
	if err != nil {
		t.Fatalf("crear institución: %v", err)
	}
	if err := institutionStorage.SetAnalyzers(ctx, inst.ID, []string{"fake-analyzer"}); err != nil {
		t.Fatalf("set analyzers: %v", err)
	}
	instAnalyzers, err := institutionStorage.GetAnalyzers(ctx, inst.ID)
	if err != nil || len(instAnalyzers) == 0 {
		t.Fatalf("obtener analyzers de institución: %v", err)
	}

	svc, err := serviceSvc.Create(ctx, &models.Service{
		HomeID:                home.ID,
		Name:                  "Internet",
		CurrencyID:            currencies[0].ID,
		Frequency:             FrequencyMonthly,
		SuggestedAmount:       45,
		Active:                true,
		IconKey:               "internet",
		InstitutionID:         &inst.ID,
		InstitutionAnalyzerID: &instAnalyzers[0].ID,
	})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}

	payload := []byte("contenido-del-pdf-de-prueba-para-hash")
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "factura.pdf")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(payload); err != nil {
		t.Fatalf("escribir payload: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest("POST", "/", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	file, header, err := req.FormFile("file")
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	defer file.Close()

	result, analyzerID, fileHash, err := docSvc.UploadAndAnalyze(ctx, svc.ID, file, header)
	if err != nil {
		t.Fatalf("upload and analyze: %v", err)
	}
	if result == nil || result.Amount != 99 {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if analyzerID != "fake-analyzer" {
		t.Errorf("analyzer esperado fake-analyzer, got %s", analyzerID)
	}

	sum := md5.Sum(payload)
	expected := hex.EncodeToString(sum[:])
	if fileHash != expected {
		t.Errorf("file_hash esperado %s, got %s", expected, fileHash)
	}
}
