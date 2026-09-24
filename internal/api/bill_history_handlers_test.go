package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newBillHistoryTestHandlers(t *testing.T) (*BillHandlers, *services.BillService, *services.ServiceService, *services.HomeService, *services.CurrencyService) {
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

	homeSvc := services.NewHomeService(homeStorage)
	currencySvc := services.NewCurrencyService(currencyStorage)
	serviceSvc := services.NewServiceService(serviceStorage, homeStorage, currencyStorage, billStorage)
	billSvc := services.NewBillService(billStorage, serviceStorage)
	billSvc.SetBillHistoryStorage(historyStorage)

	return NewBillHandlers(billSvc), billSvc, serviceSvc, homeSvc, currencySvc
}

func TestGetBillHistoryHandler(t *testing.T) {
	h, billSvc, serviceSvc, homeSvc, currencySvc := newBillHistoryTestHandlers(t)
	ctx := context.Background()

	home, err := homeSvc.Create(ctx, "Casa Historial", "")
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
		Frequency:       services.FrequencyMonthly,
		SuggestedAmount: 45,
		Active:          true,
		IconKey:         "internet",
	})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}

	bill, err := billSvc.Create(ctx, &models.Bill{
		ServiceID:     svc.ID,
		Year:          2026,
		Month:         4,
		Amount:        100,
		InvoiceNumber: "FAC-H-1",
		Status:        "pending",
	})
	if err != nil {
		t.Fatalf("crear factura: %v", err)
	}

	// Factura inexistente → 404.
	req := httptest.NewRequest(http.MethodGet, "/api/bills/99999/history", nil)
	req.SetPathValue("id", "99999")
	rr := httptest.NewRecorder()
	h.GetBillHistory(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("factura inexistente esperaba 404, got %d", rr.Code)
	}

	// Factura existente → 200 con al menos 1 evento created.
	idStr := strconv.FormatInt(bill.ID, 10)
	req = httptest.NewRequest(http.MethodGet, "/api/bills/"+idStr+"/history", nil)
	req.SetPathValue("id", idStr)
	rr = httptest.NewRecorder()
	h.GetBillHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("historial esperaba 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var events []models.BillHistory
	if err := json.Unmarshal(rr.Body.Bytes(), &events); err != nil {
		t.Fatalf("decodificar historial: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("se esperaba al menos 1 evento de historial")
	}
	if events[0].Action != models.BillActionCreated {
		t.Errorf("evento esperado created, got %s", events[0].Action)
	}
}
