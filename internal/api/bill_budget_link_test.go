package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// TestPayBillCreatesBudgetTransaction verifica el flujo completo SPEC-094:
// categoría vinculada a un servicio + cuenta → pagar factura del servicio
// (POST /api/bills/{id}/pay) → se crea la transacción de presupuesto.
func TestPayBillCreatesBudgetTransaction(t *testing.T) {
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
	historyStorage := storage.NewBillHistoryStorage(database)
	categoryStorage := storage.NewCategoryStorage(database)
	categoryGroupStorage := storage.NewCategoryGroupStorage(database)
	accountStorage := storage.NewAccountStorage(database)
	txStorage := storage.NewTransactionStorage(database)

	homeSvc := services.NewHomeService(homeStorage)
	currencySvc := services.NewCurrencyService(currencyStorage)
	serviceSvc := services.NewServiceService(serviceStorage, homeStorage, currencyStorage, billStorage)
	billSvc := services.NewBillService(billStorage, serviceStorage)
	billSvc.SetBillHistoryStorage(historyStorage)
	categorySvc := services.NewCategoryService(categoryStorage, categoryGroupStorage)
	categorySvc.SetServiceStorage(serviceStorage)
	groupSvc := services.NewCategoryGroupService(categoryGroupStorage, categoryStorage)

	budgetLink := services.NewBudgetBillLinkService(serviceStorage, categoryStorage, accountStorage, txStorage)
	billSvc.SetBillBudgetLinker(budgetLink)

	handler := NewBillHandlers(billSvc)

	// Fixture: home, servicio, moneda, cuenta y categoría vinculada.
	home, err := homeSvc.Create(ctx, "Casa Presupuesto", "")
	if err != nil {
		t.Fatalf("crear hogar: %v", err)
	}
	currencies, err := currencySvc.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}
	currencyID := currencies[0].ID

	svc, err := serviceSvc.Create(ctx, &models.Service{
		HomeID:          home.ID,
		Name:            "Internet",
		Institution:     "Claro",
		CurrencyID:      currencyID,
		Frequency:       services.FrequencyMonthly,
		SuggestedAmount: 45,
		Active:          true,
		IconKey:         "internet",
	})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}

	group, err := groupSvc.Create(ctx, "Necesidades", "home")
	if err != nil {
		t.Fatalf("crear grupo: %v", err)
	}

	account, err := accountStorage.Create(ctx, &models.Account{Name: "Efectivo", Type: "cash", CurrencyID: currencyID})
	if err != nil {
		t.Fatalf("crear cuenta: %v", err)
	}

	serviceID := svc.ID
	category, err := categorySvc.Create(ctx, &models.Category{
		CategoryGroupID: group.ID,
		Name:            "Internet",
		Icon:            "wifi",
		ServiceID:       &serviceID,
		AccountID:       &account.ID,
	})
	if err != nil {
		t.Fatalf("crear categoría vinculada: %v", err)
	}

	bill, err := billSvc.Create(ctx, &models.Bill{
		ServiceID:     svc.ID,
		Year:          2026,
		Month:         8,
		Amount:        1250,
		InvoiceNumber: "INV-PB-1",
		Status:        "pending",
	})
	if err != nil {
		t.Fatalf("crear factura: %v", err)
	}

	// Pagar la factura vía el handler.
	body := bytes.NewBufferString(`{"paid_at":"2026-08-27","payment_reference":"TRX-PB-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/bills/1/pay", body)
	req.SetPathValue("id", strconv.FormatInt(bill.ID, 10))
	rr := httptest.NewRecorder()
	handler.PayBill(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("pay esperaba 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// La transacción debe existir en la categoría vinculada con source_bill_id.
	txs, err := txStorage.ListByCategoryMonth(ctx, category.ID, 2026, 8)
	if err != nil {
		t.Fatalf("listar transacciones de categoría: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("esperaba 1 transacción, got %d", len(txs))
	}
	tx := txs[0]
	if tx.Outflow != 1250 {
		t.Errorf("outflow esperado 1250, got %v", tx.Outflow)
	}
	if tx.AccountID != account.ID {
		t.Errorf("account_id esperado %d, got %d", account.ID, tx.AccountID)
	}
	if tx.CurrencyID != currencyID {
		t.Errorf("currency_id esperado %d, got %d", currencyID, tx.CurrencyID)
	}
	if tx.Date != "2026-08-27" {
		t.Errorf("date esperado 2026-08-27, got %q", tx.Date)
	}
	if tx.Payee != "Internet" {
		t.Errorf("payee esperado Internet, got %q", tx.Payee)
	}
	if tx.SourceBillID == nil || *tx.SourceBillID != bill.ID {
		t.Errorf("source_bill_id esperado %d, got %v", bill.ID, tx.SourceBillID)
	}

	// Re-pagar no debe duplicar la transacción (idempotencia).
	req2 := httptest.NewRequest(http.MethodPost, "/api/bills/1/pay", bytes.NewBufferString(`{"paid_at":"2026-08-27"}`))
	req2.SetPathValue("id", strconv.FormatInt(bill.ID, 10))
	rr2 := httptest.NewRecorder()
	handler.PayBill(rr2, req2)
	if rr2.Code == http.StatusOK {
		t.Fatal("re-pay de factura ya pagada debería fallar (already_paid)")
	}

	txs, err = txStorage.ListByCategoryMonth(ctx, category.ID, 2026, 8)
	if err != nil {
		t.Fatalf("listar transacciones 2: %v", err)
	}
	if len(txs) != 1 {
		t.Errorf("re-pay no debe duplicar transacción, got %d", len(txs))
	}
}
