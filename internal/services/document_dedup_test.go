package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/analyzers"
	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newTestDocument(t *testing.T) (*DocumentService, *CurrencyService, *HomeService, *ServiceService) {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	currencyStorage := storage.NewCurrencyStorage(database)
	homeStorage := storage.NewHomeStorage(database)
	serviceStorage := storage.NewServiceStorage(database)
	billStorage := storage.NewBillStorage(database)
	institutionStorage := storage.NewInstitutionStorage(database)

	return NewDocumentService(serviceStorage, billStorage, institutionStorage),
		NewCurrencyService(currencyStorage),
		NewHomeService(homeStorage),
		NewServiceService(serviceStorage, homeStorage, currencyStorage, billStorage)
}

func TestCreateBillFromExtractedDedup(t *testing.T) {
	docSvc, currencySvc, homeSvc, serviceSvc := newTestDocument(t)
	ctx := context.Background()

	home, err := homeSvc.Create(ctx, "Casa Test", "")
	if err != nil {
		t.Fatalf("crear hogar: %v", err)
	}
	currencies, err := currencySvc.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}
	svc, err := serviceSvc.Create(ctx, &models.Service{
		HomeID:          home.ID,
		Name:            "Internet",
		Institution:     "Claro",
		CurrencyID:      currencies[0].ID,
		Frequency:       FrequencyMonthly,
		SuggestedAmount: 45,
		Active:          true,
		IconKey:         "internet",
	})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}

	hash := "abc123hash"
	extracted := &analyzers.ExtractedBill{Amount: 100, InvoiceNumber: "A123", Year: 2024, Month: 5}

	// Primera importación → create con file_hash persistido.
	bill, updated, duplicate, err := docSvc.CreateBillFromExtracted(ctx, svc.ID, extracted, hash)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if updated || duplicate {
		t.Errorf("primera importación debería ser create (updated=%v duplicate=%v)", updated, duplicate)
	}
	if bill.FileHash != hash {
		t.Errorf("file_hash esperado %s, got %s", hash, bill.FileHash)
	}

	// Mismo archivo re-subido (mismo hash, distinto periodo) → duplicate.
	_, _, dup, err := docSvc.CreateBillFromExtracted(ctx, svc.ID,
		&analyzers.ExtractedBill{Amount: 100, InvoiceNumber: "A123", Year: 2024, Month: 6}, hash)
	if err != nil {
		t.Fatalf("duplicate: %v", err)
	}
	if !dup {
		t.Error("re-subir el mismo hash debería marcar duplicate")
	}

	// Mismo periodo con archivo distinto (nuevo hash) → update, sin duplicar.
	_, upd, dup2, err := docSvc.CreateBillFromExtracted(ctx, svc.ID,
		&analyzers.ExtractedBill{Amount: 120, InvoiceNumber: "A456", Year: 2024, Month: 5}, "otherhash")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !upd || dup2 {
		t.Errorf("mismo periodo con hash nuevo debería actualizar (updated=%v duplicate=%v)", upd, dup2)
	}
}

func TestCreateBillFromExtractedNoHash(t *testing.T) {
	docSvc, currencySvc, homeSvc, serviceSvc := newTestDocument(t)
	ctx := context.Background()

	home, err := homeSvc.Create(ctx, "Casa Test", "")
	if err != nil {
		t.Fatalf("crear hogar: %v", err)
	}
	currencies, err := currencySvc.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}
	svc, err := serviceSvc.Create(ctx, &models.Service{
		HomeID:          home.ID,
		Name:            "Internet",
		CurrencyID:      currencies[0].ID,
		Frequency:       FrequencyMonthly,
		SuggestedAmount: 45,
		Active:          true,
		IconKey:         "internet",
	})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}

	// Sin hash (factura manual / sin PDF) → no aplica dedup, se crea normalmente.
	_, _, duplicate, err := docSvc.CreateBillFromExtracted(ctx, svc.ID,
		&analyzers.ExtractedBill{Amount: 100, InvoiceNumber: "X1", Year: 2023, Month: 3}, "")
	if err != nil {
		t.Fatalf("create sin hash: %v", err)
	}
	if duplicate {
		t.Error("sin hash no debería marcar duplicate")
	}
}
