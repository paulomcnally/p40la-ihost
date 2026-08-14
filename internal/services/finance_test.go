package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newTestFinance(t *testing.T) (*CurrencyService, *HomeService, *ServiceService, *BillService) {
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

	return NewCurrencyService(currencyStorage),
		NewHomeService(homeStorage),
		NewServiceService(serviceStorage, homeStorage, currencyStorage, billStorage),
		NewBillService(billStorage, serviceStorage)
}

func TestCurrencyCreateAndList(t *testing.T) {
	currencySvc, _, _, _ := newTestFinance(t)
	ctx := context.Background()

	currency, err := currencySvc.Create(ctx, "eur", "Euro", "€")
	if err != nil {
		t.Fatalf("crear moneda: %v", err)
	}
	if currency.Code != "EUR" {
		t.Errorf("código esperado EUR, got %s", currency.Code)
	}

	list, err := currencySvc.List(ctx)
	if err != nil {
		t.Fatalf("listar monedas: %v", err)
	}
	if len(list) < 3 {
		t.Errorf("esperaba al menos 3 monedas (seed + EUR), got %d", len(list))
	}
}

func TestHomeCreateAndCount(t *testing.T) {
	_, homeSvc, _, _ := newTestFinance(t)
	ctx := context.Background()

	home, err := homeSvc.Create(ctx, "Casa Test", "Dirección")
	if err != nil {
		t.Fatalf("crear hogar: %v", err)
	}
	if home.Name != "Casa Test" {
		t.Errorf("nombre esperado Casa Test, got %s", home.Name)
	}

	count, err := homeSvc.Count(ctx)
	if err != nil {
		t.Fatalf("contar hogares: %v", err)
	}
	if count != 1 {
		t.Errorf("count esperado 1, got %d", count)
	}
}

func TestServiceRequiresHome(t *testing.T) {
	_, _, serviceSvc, _ := newTestFinance(t)
	ctx := context.Background()

	_, err := serviceSvc.Create(ctx, &models.Service{
		Name:            "Internet",
		CurrencyID:      1,
		Frequency:       FrequencyMonthly,
		SuggestedAmount: 50,
		Active:          true,
		IconKey:         "internet",
	})
	if err == nil {
		t.Error("debería rechazar servicio sin hogar")
	}
}

func TestServiceCreatesBillAutomatically(t *testing.T) {
	currencySvc, homeSvc, serviceSvc, billSvc := newTestFinance(t)
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

	bills, err := billSvc.ListByService(ctx, svc.ID)
	if err != nil {
		t.Fatalf("listar facturas: %v", err)
	}
	if len(bills) != 1 {
		t.Fatalf("esperaba 1 factura, got %d", len(bills))
	}
	if bills[0].Status != "pending" {
		t.Errorf("estado esperado pending, got %s", bills[0].Status)
	}
	if bills[0].Amount != 45 {
		t.Errorf("monto esperado 45, got %f", bills[0].Amount)
	}
}

func TestBillPaidRequiresDriveURL(t *testing.T) {
	currencySvc, homeSvc, serviceSvc, billSvc := newTestFinance(t)
	ctx := context.Background()

	home, _ := homeSvc.Create(ctx, "Casa Test", "")
	currencies, _ := currencySvc.List(ctx)
	svc, _ := serviceSvc.Create(ctx, &models.Service{
		HomeID:          home.ID,
		Name:            "Internet",
		CurrencyID:      currencies[0].ID,
		Frequency:       FrequencyMonthly,
		SuggestedAmount: 45,
		Active:          true,
		IconKey:         "internet",
	})

	bills, _ := billSvc.ListByService(ctx, svc.ID)
	bill := bills[0]

	// Sin Drive URL debe fallar.
	bill.Status = "paid"
	bill.DriveURL = ""
	if _, err := billSvc.Update(ctx, &bill); err == nil {
		t.Error("debería requerir URL de Drive para pagar")
	}

	// URL inválida debe fallar.
	bill.DriveURL = "https://example.com/file"
	if _, err := billSvc.Update(ctx, &bill); err == nil {
		t.Error("debería rechazar URL de Drive inválida")
	}

	// URL válida debe funcionar.
	bill.DriveURL = "https://drive.google.com/file/d/abc123/view"
	updated, err := billSvc.Update(ctx, &bill)
	if err != nil {
		t.Fatalf("actualizar factura pagada: %v", err)
	}
	if updated.Status != "paid" {
		t.Errorf("estado esperado paid, got %s", updated.Status)
	}
}

func TestServiceFilterByHome(t *testing.T) {
	currencySvc, homeSvc, serviceSvc, _ := newTestFinance(t)
	ctx := context.Background()

	home1, _ := homeSvc.Create(ctx, "Casa 1", "")
	home2, _ := homeSvc.Create(ctx, "Casa 2", "")
	currencies, _ := currencySvc.List(ctx)

	_, _ = serviceSvc.Create(ctx, &models.Service{
		HomeID:          home1.ID,
		Name:            "Servicio 1",
		CurrencyID:      currencies[0].ID,
		Frequency:       FrequencyMonthly,
		SuggestedAmount: 10,
		Active:          true,
		IconKey:         "other",
	})
	_, _ = serviceSvc.Create(ctx, &models.Service{
		HomeID:          home2.ID,
		Name:            "Servicio 2",
		CurrencyID:      currencies[0].ID,
		Frequency:       FrequencyMonthly,
		SuggestedAmount: 20,
		Active:          true,
		IconKey:         "other",
	})

	list, err := serviceSvc.List(ctx, &home1.ID)
	if err != nil {
		t.Fatalf("listar servicios filtrados: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("esperaba 1 servicio, got %d", len(list))
	}
}
