package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newTestBillHistory(t *testing.T) (*BillService, *ServiceService, *storage.BillHistoryStorage) {
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
	historyStorage := storage.NewBillHistoryStorage(database)

	billSvc := NewBillService(billStorage, serviceStorage)
	billSvc.SetBillHistoryStorage(historyStorage)

	serviceSvc := NewServiceService(serviceStorage, homeStorage, currencyStorage, billStorage)
	return billSvc, serviceSvc, historyStorage
}

func createTestServiceForHistory(t *testing.T, serviceSvc *ServiceService) *models.Service {
	t.Helper()
	ctx := context.Background()

	home, err := serviceSvc.homes.Create(ctx, "Casa Historial", "")
	if err != nil {
		t.Fatalf("crear hogar: %v", err)
	}
	currencies, err := serviceSvc.currencies.List(ctx)
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
	return svc
}

func TestBillHistoryCreatedAndUpdated(t *testing.T) {
	billSvc, serviceSvc, history := newTestBillHistory(t)
	ctx := context.Background()
	svc := createTestServiceForHistory(t, serviceSvc)

	bill, err := billSvc.Create(ctx, &models.Bill{
		ServiceID:     svc.ID,
		Year:          2026,
		Month:         4,
		Amount:        350.5,
		InvoiceNumber: "FAC-001",
		Status:        "pending",
	})
	if err != nil {
		t.Fatalf("crear factura: %v", err)
	}

	events, err := history.ListByBill(ctx, bill.ID)
	if err != nil {
		t.Fatalf("listar historial: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("se esperaba 1 evento creado, got %d", len(events))
	}
	if events[0].Action != models.BillActionCreated || events[0].Source != models.BillSourceDashboard {
		t.Errorf("evento esperado created/dashboard, got %s/%s", events[0].Action, events[0].Source)
	}

	bill.Amount = 400
	updated, err := billSvc.Update(ctx, bill)
	if err != nil {
		t.Fatalf("actualizar factura: %v", err)
	}

	events, err = history.ListByBill(ctx, updated.ID)
	if err != nil {
		t.Fatalf("listar historial: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("se esperaban 2 eventos, got %d", len(events))
	}
	ev := events[0]
	if ev.Action != models.BillActionUpdated || ev.Source != models.BillSourceDashboard {
		t.Errorf("evento esperado updated/dashboard, got %s/%s", ev.Action, ev.Source)
	}
	if len(ev.Changes) != 1 || ev.Changes[0].Field != "amount" {
		t.Errorf("se esperaba 1 cambio en amount, got %+v", ev.Changes)
	}
	if ev.Changes[0].Old.(float64) != 350.5 || ev.Changes[0].New.(float64) != 400 {
		t.Errorf("diff amount incorrecto: %v -> %v", ev.Changes[0].Old, ev.Changes[0].New)
	}
}

func TestBillHistoryNoNoopUpdate(t *testing.T) {
	billSvc, serviceSvc, history := newTestBillHistory(t)
	ctx := context.Background()
	svc := createTestServiceForHistory(t, serviceSvc)

	bill, err := billSvc.Create(ctx, &models.Bill{
		ServiceID:     svc.ID,
		Year:          2026,
		Month:         5,
		Amount:        100,
		InvoiceNumber: "FAC-002",
		Status:        "pending",
	})
	if err != nil {
		t.Fatalf("crear factura: %v", err)
	}

	// Update sin cambios reales no debe registrar evento.
	if _, err := billSvc.Update(ctx, bill); err != nil {
		t.Fatalf("actualizar factura: %v", err)
	}

	events, err := history.ListByBill(ctx, bill.ID)
	if err != nil {
		t.Fatalf("listar historial: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("noop no debe crear evento, got %d", len(events))
	}
}
