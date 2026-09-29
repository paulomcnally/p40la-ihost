package services

import (
	"context"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// newTestBudgetBillLink arma el BudgetBillLinkService + helpers sobre una DB en
// memoria con un home, servicio, categoría vinculada y cuenta.
func newTestBudgetBillLink(t *testing.T) (*BudgetBillLinkService, *BudgetTransactionService, *CategoryService, int64, int64, int64) {
	t.Helper()
	ctx := context.Background()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	serviceStorage := storage.NewServiceStorage(database)
	categoryStorage := storage.NewCategoryStorage(database)
	accountStorage := storage.NewAccountStorage(database)
	txStorage := storage.NewTransactionStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	groupStorage := storage.NewCategoryGroupStorage(database)

	currencies, err := currencyStorage.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}
	currencyID := currencies[0].ID

	linkSvc := NewBudgetBillLinkService(serviceStorage, categoryStorage, accountStorage, txStorage)
	txSvc := NewBudgetTransactionService(txStorage, accountStorage, categoryStorage, currencyStorage)
	catSvc := NewCategoryService(categoryStorage, groupStorage)
	catSvc.SetServiceStorage(serviceStorage)

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("insertar dato de prueba (%q): %v", query, err)
		}
	}

	mustExec("INSERT INTO homes (name) VALUES ('Casa')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Internet', 'Claro', ?, 'monthly', 100, 1, 'internet', 'fixed', 1)`, currencyID)
	mustExec("INSERT INTO category_groups (name, icon) VALUES ('Necesidades', 'home')")

	groupID := int64(1)
	serviceID := int64(1)

	account, err := accountStorage.Create(ctx, &models.Account{Name: "Efectivo", Type: "cash", CurrencyID: currencyID})
	if err != nil {
		t.Fatalf("crear cuenta: %v", err)
	}

	category, err := categoryStorage.Create(ctx, &models.Category{
		CategoryGroupID: groupID,
		Name:            "Internet",
		Icon:            "wifi",
		ServiceIDs:      []int64{serviceID},
		AccountID:       &account.ID,
	})
	if err != nil {
		t.Fatalf("crear categoría vinculada: %v", err)
	}
	_ = category

	return linkSvc, txSvc, catSvc, serviceID, account.ID, currencyID
}

func paidBill(id, serviceID int64, amount float64, paidAt time.Time) *models.Bill {
	return &models.Bill{ID: id, ServiceID: serviceID, Amount: amount, PaidAt: &paidAt, InvoiceNumber: "INV-1", Year: 2026, Month: 9}
}

func TestBudgetBillLinkCreatesTransactionOnPaid(t *testing.T) {
	linkSvc, txSvc, _, serviceID, _, currencyID := newTestBudgetBillLink(t)
	ctx := context.Background()

	paidAt := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	bill := paidBill(1, serviceID, 1500, paidAt)
	if err := linkSvc.OnBillPaid(ctx, bill); err != nil {
		t.Fatalf("OnBillPaid: %v", err)
	}

	txs, err := txSvc.ListByMonth(ctx, 2026, 9)
	if err != nil {
		t.Fatalf("listar transacciones: %v", err)
	}
	if len(txs) != 1 {
		t.Fatalf("esperaba 1 transacción, got %d", len(txs))
	}
	tx := txs[0]
	if tx.Outflow != 1500 {
		t.Errorf("outflow esperado 1500, got %v", tx.Outflow)
	}
	if tx.CurrencyID != currencyID {
		t.Errorf("currency_id esperado %d, got %d", currencyID, tx.CurrencyID)
	}
	if tx.Date != "2026-09-25" {
		t.Errorf("date esperado 2026-09-25, got %q", tx.Date)
	}
	if tx.Payee != "Internet" {
		t.Errorf("payee esperado Internet, got %q", tx.Payee)
	}
	if !tx.Cleared {
		t.Error("cleared esperado true")
	}
	if tx.SourceBillID == nil || *tx.SourceBillID != 1 {
		t.Errorf("source_bill_id esperado 1, got %v", tx.SourceBillID)
	}
}

func TestBudgetBillLinkIdempotent(t *testing.T) {
	linkSvc, txSvc, _, serviceID, _, _ := newTestBudgetBillLink(t)
	ctx := context.Background()

	paidAt := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	bill := paidBill(1, serviceID, 1500, paidAt)
	if err := linkSvc.OnBillPaid(ctx, bill); err != nil {
		t.Fatalf("OnBillPaid 1: %v", err)
	}
	// Re-procesar el mismo pago no debe duplicar (idempotencia por source_bill_id).
	if err := linkSvc.OnBillPaid(ctx, bill); err != nil {
		t.Fatalf("OnBillPaid 2: %v", err)
	}

	txs, err := txSvc.ListByMonth(ctx, 2026, 9)
	if err != nil {
		t.Fatalf("listar transacciones: %v", err)
	}
	if len(txs) != 1 {
		t.Errorf("esperaba 1 transacción (idempotencia), got %d", len(txs))
	}
}

func TestBudgetBillLinkNoOpWithoutCategory(t *testing.T) {
	linkSvc, txSvc, _, _, _, _ := newTestBudgetBillLink(t)
	ctx := context.Background()

	// Servicio 2 sin categoría vinculada.
	serviceID := int64(2)

	paidAt := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	bill := paidBill(2, serviceID, 300, paidAt)
	if err := linkSvc.OnBillPaid(ctx, bill); err == nil {
		t.Log("servicio inexistente retorna error (esperado)")
	}

	txs, err := txSvc.ListByMonth(ctx, 2026, 9)
	if err != nil {
		t.Fatalf("listar transacciones: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("no debe crearse transacción sin vínculo, got %d", len(txs))
	}
}

func TestBudgetBillLinkNoOpWhenAccountMissing(t *testing.T) {
	ctx := context.Background()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	serviceStorage := storage.NewServiceStorage(database)
	categoryStorage := storage.NewCategoryStorage(database)
	accountStorage := storage.NewAccountStorage(database)
	txStorage := storage.NewTransactionStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)

	currencies, err := currencyStorage.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}
	currencyID := currencies[0].ID

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("insertar dato de prueba (%q): %v", query, err)
		}
	}
	mustExec("INSERT INTO homes (name) VALUES ('Casa')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Internet', 'Claro', ?, 'monthly', 100, 1, 'internet', 'fixed', 1)`, currencyID)
	mustExec("INSERT INTO category_groups (name, icon) VALUES ('Necesidades', 'home')")

	serviceID := int64(1)
	// Categoría vinculada al servicio SIN account_id (config incompleta).
	mustExec("INSERT INTO categories (category_group_id, name, icon, account_id) VALUES (1, 'Internet', 'wifi', NULL)")
	mustExec("INSERT INTO category_service_links (category_id, service_id) VALUES (1, 1)")

	linkSvc := NewBudgetBillLinkService(serviceStorage, categoryStorage, accountStorage, txStorage)
	txSvc := NewBudgetTransactionService(txStorage, accountStorage, categoryStorage, currencyStorage)

	paidAt := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	bill := paidBill(1, serviceID, 300, paidAt)
	if err := linkSvc.OnBillPaid(ctx, bill); err != nil {
		t.Fatalf("OnBillPaid con cuenta faltante no debe fallar: %v", err)
	}

	txs, err := txSvc.ListByMonth(ctx, 2026, 9)
	if err != nil {
		t.Fatalf("listar transacciones: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("sin cuenta configurada no debe crearse transacción, got %d", len(txs))
	}
}

func TestCategoryServiceServiceLinkValidation(t *testing.T) {
	ctx := context.Background()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	serviceStorage := storage.NewServiceStorage(database)
	categoryStorage := storage.NewCategoryStorage(database)
	accountStorage := storage.NewAccountStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	groupStorage := storage.NewCategoryGroupStorage(database)

	currencies, err := currencyStorage.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}
	currencyID := currencies[0].ID

	catSvc := NewCategoryService(categoryStorage, groupStorage)
	catSvc.SetServiceStorage(serviceStorage)

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("insertar dato de prueba (%q): %v", query, err)
		}
	}
	mustExec("INSERT INTO homes (name) VALUES ('Casa 2')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Luz', 'DISNORTE', ?, 'monthly', 200, 1, 'electricity', 'fixed', 1)`, currencyID)
	mustExec("INSERT INTO category_groups (name, icon) VALUES ('Necesidades', 'home')")

	account, err := accountStorage.Create(ctx, &models.Account{Name: "Banco", Type: "checking", CurrencyID: currencyID})
	if err != nil {
		t.Fatalf("crear cuenta: %v", err)
	}

	serviceID := int64(1)

	// service_ids sin account_id → error.
	_, err = catSvc.Create(ctx, &models.Category{
		CategoryGroupID: 1,
		Name:            "Sin cuenta",
		ServiceIDs:      []int64{serviceID},
		AccountID:       nil,
	})
	if err == nil {
		t.Error("esperaba error: service_ids sin account_id")
	}

	// service_ids con account_id → ok.
	first, err := catSvc.Create(ctx, &models.Category{
		CategoryGroupID: 1,
		Name:            "Luz",
		ServiceIDs:      []int64{serviceID},
		AccountID:       &account.ID,
	})
	if err != nil {
		t.Fatalf("crear categoría con vínculo: %v", err)
	}

	// Mismo servicio en otra categoría activa → error de unicidad.
	_, err = catSvc.Create(ctx, &models.Category{
		CategoryGroupID: 1,
		Name:            "Luz 2",
		ServiceIDs:      []int64{serviceID},
		AccountID:       &account.ID,
	})
	if err == nil {
		t.Error("esperaba error: servicio ya vinculado a otra categoría")
	}

	// Editar la primera para desvincular (service_ids vacío) debe ser válido.
	first.ServiceIDs = []int64{}
	first.AccountID = nil
	if _, err := catSvc.Update(ctx, first); err != nil {
		t.Fatalf("desvincular categoría: %v", err)
	}
}

// TestBudgetBillLinkMultipleServices verifica que una categoría vinculada a dos
// servicios genera transacciones al pagar facturas de cualquiera de ellos
// (SPEC-096).
func TestBudgetBillLinkMultipleServices(t *testing.T) {
	ctx := context.Background()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	serviceStorage := storage.NewServiceStorage(database)
	categoryStorage := storage.NewCategoryStorage(database)
	accountStorage := storage.NewAccountStorage(database)
	txStorage := storage.NewTransactionStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	groupStorage := storage.NewCategoryGroupStorage(database)

	currencies, err := currencyStorage.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}
	currencyID := currencies[0].ID

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("insertar dato de prueba (%q): %v", query, err)
		}
	}
	mustExec("INSERT INTO homes (name) VALUES ('Casa')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Claro Internet Móvil', 'Claro', ?, 'monthly', 100, 1, 'internet', 'fixed', 1)`, currencyID)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Tigo Internet Móvil', 'Tigo', ?, 'monthly', 100, 1, 'internet', 'fixed', 1)`, currencyID)
	mustExec("INSERT INTO category_groups (name, icon) VALUES ('Servicios', 'home')")

	catSvc := NewCategoryService(categoryStorage, groupStorage)
	catSvc.SetServiceStorage(serviceStorage)

	account, err := accountStorage.Create(ctx, &models.Account{Name: "Efectivo", Type: "cash", CurrencyID: currencyID})
	if err != nil {
		t.Fatalf("crear cuenta: %v", err)
	}

	// Una categoría "Internet Móvil" vinculada a dos servicios (Claro y Tigo).
	cat, err := catSvc.Create(ctx, &models.Category{
		CategoryGroupID: 1,
		Name:            "Internet Móvil",
		Icon:            "internet",
		ServiceIDs:      []int64{1, 2},
		AccountID:       &account.ID,
	})
	if err != nil {
		t.Fatalf("crear categoría multi-servicio: %v", err)
	}
	if len(cat.ServiceIDs) != 2 {
		t.Errorf("esperaba 2 service_ids, got %d", len(cat.ServiceIDs))
	}

	linkSvc := NewBudgetBillLinkService(serviceStorage, categoryStorage, accountStorage, txStorage)
	txSvc := NewBudgetTransactionService(txStorage, accountStorage, categoryStorage, currencyStorage)

	paidAt := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	// Factura de Claro (servicio 1) y factura de Tigo (servicio 2).
	bill1 := paidBill(1, 1, 1000, paidAt)
	bill2 := paidBill(2, 2, 800, paidAt)
	if err := linkSvc.OnBillPaid(ctx, bill1); err != nil {
		t.Fatalf("OnBillPaid servicio 1: %v", err)
	}
	if err := linkSvc.OnBillPaid(ctx, bill2); err != nil {
		t.Fatalf("OnBillPaid servicio 2: %v", err)
	}

	txs, err := txSvc.ListByCategoryMonth(ctx, cat.ID, 2026, 9)
	if err != nil {
		t.Fatalf("listar transacciones de categoría: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("esperaba 2 transacciones en la categoría, got %d", len(txs))
	}
	got := map[string]bool{}
	for _, tx := range txs {
		got[tx.Payee] = true
	}
	if !got["Claro Internet Móvil"] || !got["Tigo Internet Móvil"] {
		t.Errorf("transacciones esperadas de ambos servicios, got %v", got)
	}

	// Idempotencia: re-procesar la factura de Tigo no duplica.
	if err := linkSvc.OnBillPaid(ctx, bill2); err != nil {
		t.Fatalf("OnBillPaid re-procesado: %v", err)
	}
	txs, err = txSvc.ListByCategoryMonth(ctx, cat.ID, 2026, 9)
	if err != nil {
		t.Fatalf("listar transacciones: %v", err)
	}
	if len(txs) != 2 {
		t.Errorf("esperaba 2 transacciones (idempotencia), got %d", len(txs))
	}
}
