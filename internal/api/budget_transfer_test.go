package api

import (
	"bytes"
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

// newBudgetHandlerTestDB arma un BudgetHandlers con sus servicios sobre una DB
// en memoria con las migraciones aplicadas.
func newBudgetHandlerTestDB(t *testing.T) (*BudgetHandlers, *storage.CurrencyStorage, context.Context) {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	ctx := context.Background()

	currencyStorage := storage.NewCurrencyStorage(database)
	categoryGroupStorage := storage.NewCategoryGroupStorage(database)
	categoryStorage := storage.NewCategoryStorage(database)
	accountStorage := storage.NewAccountStorage(database)
	budgetMonthStorage := storage.NewBudgetMonthStorage(database)
	recurringRuleStorage := storage.NewRecurringRuleStorage(database)
	assignmentStorage := storage.NewAssignmentStorage(database)
	transactionStorage := storage.NewTransactionStorage(database)
	transferStorage := storage.NewTransferStorage(database)

	categoryGroupService := services.NewCategoryGroupService(categoryGroupStorage, categoryStorage)
	categoryService := services.NewCategoryService(categoryStorage, categoryGroupStorage)
	accountService := services.NewAccountService(accountStorage, currencyStorage)
	budgetService := services.NewBudgetService(categoryGroupStorage, categoryStorage, budgetMonthStorage, assignmentStorage, recurringRuleStorage, transactionStorage, currencyStorage)
	budgetTransactionService := services.NewBudgetTransactionService(transactionStorage, accountStorage, categoryStorage, currencyStorage)
	transferService := services.NewTransferService(transferStorage, accountStorage, currencyStorage)

	return NewBudgetHandlers(budgetService, categoryGroupService, categoryService, accountService, budgetTransactionService, transferService), currencyStorage, ctx
}

// TestTransferCRUDAndBalance verifica el flujo completo SPEC-099: crear cuenta A
// y B, registrar una transacción de gasto en A, transferir dinero A→B y
// comprobar que los balances se actualizan y que la transferencia no contamina
// la actividad ni los ingresos del presupuesto.
func TestTransferCRUDAndBalance(t *testing.T) {
	h, currencyStorage, ctx := newBudgetHandlerTestDB(t)
	currencies, err := currencyStorage.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}
	currencyID := currencies[0].ID

	// Cuentas A (corriente) y B (tarjeta de crédito).
	accountA, err := h.accounts.Create(ctx, &models.Account{Name: "Banco X", Type: "checking", CurrencyID: currencyID, StartingBalance: 1000})
	if err != nil {
		t.Fatalf("crear cuenta A: %v", err)
	}
	accountB, err := h.accounts.Create(ctx, &models.Account{Name: "Tarjeta Y", Type: "credit_card", CurrencyID: currencyID})
	if err != nil {
		t.Fatalf("crear cuenta B: %v", err)
	}

	// Gasto de $100 en la tarjeta (cuenta B) categorizado en una categoría.
	group, err := h.categoryGroups.Create(ctx, "Necesidades", "home")
	if err != nil {
		t.Fatalf("crear grupo: %v", err)
	}
	category, err := h.categories.Create(ctx, &models.Category{CategoryGroupID: group.ID, Name: "Comida", Icon: "food"})
	if err != nil {
		t.Fatalf("crear categoría: %v", err)
	}
	if _, err := h.transactions.Create(ctx, &models.Transaction{
		AccountID:  accountB.ID,
		CategoryID: &category.ID,
		CurrencyID: currencyID,
		Date:       "2026-10-01",
		Payee:      "Super",
		Outflow:    100,
		Cleared:    true,
	}); err != nil {
		t.Fatalf("crear transacción: %v", err)
	}

	// Crear transferencia de $100 A→B (pago de tarjeta).
	payload, _ := json.Marshal(map[string]any{
		"from_account_id": accountA.ID,
		"to_account_id":   accountB.ID,
		"currency_id":     currencyID,
		"date":            "2026-10-15",
		"memo":            "Pago tarjeta",
		"amount":          100,
		"cleared":         true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/budget/transfers", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	h.CreateTransfer(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("crear transferencia: status %d, body %s", w.Code, w.Body.String())
	}
	var created models.Transfer
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decodificar transferencia: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("transferencia creada sin id")
	}

	// Balances: A = 1000 - 100 = 900; B = -100 (gasto) + 100 (pago) = 0.
	accounts, err := h.accounts.List(ctx)
	if err != nil {
		t.Fatalf("listar cuentas: %v", err)
	}
	balanceByID := map[int64]float64{}
	for _, a := range accounts {
		balanceByID[a.ID] = a.Balance
	}
	if got := balanceByID[accountA.ID]; got != 900 {
		t.Errorf("balance A = %v, esperaba 900", got)
	}
	if got := balanceByID[accountB.ID]; got != 0 {
		t.Errorf("balance B = %v, esperaba 0", got)
	}

	// La transferencia NO afecta actividad ni ingresos del presupuesto.
	view, err := h.budget.MonthView(ctx, 2026, 10)
	if err != nil {
		t.Fatalf("vista mensual: %v", err)
	}
	var catActivity float64
	for _, g := range view.Groups {
		for _, row := range g.Categories {
			if row.ID == category.ID {
				catActivity = row.Activity[currencyID]
			}
		}
	}
	if catActivity != 100 {
		t.Errorf("actividad de la categoría = %v, esperaba 100 (solo el gasto, no la transferencia)", catActivity)
	}
	var totalIncome float64
	for _, ct := range view.CurrencyTotals {
		if ct.CurrencyID == currencyID {
			totalIncome = ct.TotalIncome
		}
	}
	if totalIncome != 0 {
		t.Errorf("ingresos del mes = %v, esperaba 0 (la transferencia no es ingreso)", totalIncome)
	}

	// Editar la transferencia (cambiar monto a 50).
	payload2, _ := json.Marshal(map[string]any{
		"from_account_id": accountA.ID,
		"to_account_id":   accountB.ID,
		"currency_id":     currencyID,
		"date":            "2026-10-15",
		"memo":            "Pago tarjeta parcial",
		"amount":          50,
		"cleared":         true,
	})
	req2 := httptest.NewRequest(http.MethodPut, "/api/budget/transfers/"+strconv.FormatInt(created.ID, 10), bytes.NewReader(payload2))
	req2.SetPathValue("id", strconv.FormatInt(created.ID, 10))
	w2 := httptest.NewRecorder()
	h.UpdateTransfer(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("editar transferencia: status %d, body %s", w2.Code, w2.Body.String())
	}

	accounts2, err := h.accounts.List(ctx)
	if err != nil {
		t.Fatalf("listar cuentas post-edit: %v", err)
	}
	balanceByID2 := map[int64]float64{}
	for _, a := range accounts2 {
		balanceByID2[a.ID] = a.Balance
	}
	if got := balanceByID2[accountA.ID]; got != 950 {
		t.Errorf("balance A post-edit = %v, esperaba 950", got)
	}
	if got := balanceByID2[accountB.ID]; got != -50 {
		t.Errorf("balance B post-edit = %v, esperaba -50", got)
	}

	// Eliminar la transferencia: vuelve a los balances previos.
	req3 := httptest.NewRequest(http.MethodDelete, "/api/budget/transfers/"+strconv.FormatInt(created.ID, 10), nil)
	req3.SetPathValue("id", strconv.FormatInt(created.ID, 10))
	w3 := httptest.NewRecorder()
	h.DeleteTransfer(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("eliminar transferencia: status %d", w3.Code)
	}

	accounts3, err := h.accounts.List(ctx)
	if err != nil {
		t.Fatalf("listar cuentas post-delete: %v", err)
	}
	balanceByID3 := map[int64]float64{}
	for _, a := range accounts3 {
		balanceByID3[a.ID] = a.Balance
	}
	if got := balanceByID3[accountA.ID]; got != 1000 {
		t.Errorf("balance A post-delete = %v, esperaba 1000", got)
	}
	if got := balanceByID3[accountB.ID]; got != -100 {
		t.Errorf("balance B post-delete = %v, esperaba -100", got)
	}
}

// TestTransferValidation verifica las validaciones de negocio de transferencias.
func TestTransferValidation(t *testing.T) {
	h, currencyStorage, ctx := newBudgetHandlerTestDB(t)
	currencies, err := currencyStorage.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}
	currencyID := currencies[0].ID

	accountA, err := h.accounts.Create(ctx, &models.Account{Name: "Banco X", Type: "checking", CurrencyID: currencyID})
	if err != nil {
		t.Fatalf("crear cuenta A: %v", err)
	}

	cases := []struct {
		name   string
		body   map[string]any
		status int
	}{
		{"misma cuenta", map[string]any{"from_account_id": accountA.ID, "to_account_id": accountA.ID, "currency_id": currencyID, "date": "2026-10-15", "amount": 100}, http.StatusBadRequest},
		{"monto cero", map[string]any{"from_account_id": accountA.ID, "to_account_id": accountA.ID + 1, "currency_id": currencyID, "date": "2026-10-15", "amount": 0}, http.StatusBadRequest},
		{"cuenta inexistente", map[string]any{"from_account_id": 9999, "to_account_id": accountA.ID + 1, "currency_id": currencyID, "date": "2026-10-15", "amount": 100}, http.StatusBadRequest},
		{"sin fecha", map[string]any{"from_account_id": accountA.ID, "to_account_id": accountA.ID + 1, "currency_id": currencyID, "amount": 100}, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/budget/transfers", bytes.NewReader(payload))
			w := httptest.NewRecorder()
			h.CreateTransfer(w, req)
			if w.Code != tc.status {
				t.Errorf("status = %d, esperaba %d (body %s)", w.Code, tc.status, w.Body.String())
			}
		})
	}
}
