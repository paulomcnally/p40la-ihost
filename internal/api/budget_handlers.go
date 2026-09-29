package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// BudgetHandlers agrupa los handlers del módulo de presupuesto (SPEC-093).
type BudgetHandlers struct {
	budget         *services.BudgetService
	categoryGroups *services.CategoryGroupService
	categories     *services.CategoryService
	accounts       *services.AccountService
	transactions   *services.BudgetTransactionService
}

// NewBudgetHandlers crea un nuevo BudgetHandlers.
func NewBudgetHandlers(
	budget *services.BudgetService,
	categoryGroups *services.CategoryGroupService,
	categories *services.CategoryService,
	accounts *services.AccountService,
	transactions *services.BudgetTransactionService,
) *BudgetHandlers {
	return &BudgetHandlers{
		budget:         budget,
		categoryGroups: categoryGroups,
		categories:     categories,
		accounts:       accounts,
		transactions:   transactions,
	}
}

// ---- Vista mensual ----

// MonthView responde la vista mensual completa.
func (h *BudgetHandlers) MonthView(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(r.PathValue("year"))
	if err != nil || year < 1900 || year > 3000 {
		respondError(w, http.StatusBadRequest, "invalid_year", "Año inválido")
		return
	}
	month, err := strconv.Atoi(r.PathValue("month"))
	if err != nil || month < 1 || month > 12 {
		respondError(w, http.StatusBadRequest, "invalid_month", "Mes inválido")
		return
	}
	view, err := h.budget.MonthView(r.Context(), year, month)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, view)
}

type assignRequest struct {
	Amount        float64 `json:"amount"`
	CurrencyID    int64   `json:"currency_id"`
	MakeRecurring bool    `json:"make_recurring"`
}

// Assign asigna un monto a una categoría en un mes.
func (h *BudgetHandlers) Assign(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(r.PathValue("year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_year", "Año inválido")
		return
	}
	month, err := strconv.Atoi(r.PathValue("month"))
	if err != nil || month < 1 || month > 12 {
		respondError(w, http.StatusBadRequest, "invalid_month", "Mes inválido")
		return
	}
	categoryID, err := strconv.ParseInt(r.PathValue("category_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de categoría inválido")
		return
	}
	var req assignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	assignment, err := h.budget.Assign(r.Context(), year, month, categoryID, req.CurrencyID, req.Amount, req.MakeRecurring)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, assignment)
}

type toggleRecurringRequest struct {
	Active bool `json:"active"`
}

// ToggleRecurring pausa o reanuda una regla recurrente.
func (h *BudgetHandlers) ToggleRecurring(w http.ResponseWriter, r *http.Request) {
	ruleID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req toggleRecurringRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	rule, err := h.budget.ToggleRecurring(r.Context(), ruleID, req.Active)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, rule)
}

// DeleteRecurringRule elimina una regla recurrente.
func (h *BudgetHandlers) DeleteRecurringRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.budget.DeleteRule(r.Context(), ruleID); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Regla recurrente eliminada"})
}

// ---- Grupos de categorías ----

type categoryGroupRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// ListCategoryGroups responde los grupos con sus categorías.
func (h *BudgetHandlers) ListCategoryGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.categoryGroups.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, groups)
}

// CreateCategoryGroup crea un grupo.
func (h *BudgetHandlers) CreateCategoryGroup(w http.ResponseWriter, r *http.Request) {
	var req categoryGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	group, err := h.categoryGroups.Create(r.Context(), req.Name, req.Icon)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, group)
}

// UpdateCategoryGroup actualiza un grupo.
func (h *BudgetHandlers) UpdateCategoryGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req categoryGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	group, err := h.categoryGroups.Update(r.Context(), id, req.Name, req.Icon)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if group == nil {
		respondError(w, http.StatusNotFound, "not_found", "Grupo no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, group)
}

// DeleteCategoryGroup elimina un grupo vacío.
func (h *BudgetHandlers) DeleteCategoryGroup(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.categoryGroups.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Grupo eliminado"})
}

// ---- Categorías ----

type categoryRequest struct {
	CategoryGroupID int64    `json:"category_group_id"`
	Name            string   `json:"name"`
	Icon            string   `json:"icon"`
	TargetAmount    *float64 `json:"target_amount"`
	ServiceIDs      []int64  `json:"service_ids"`
	AccountID       *int64   `json:"account_id"`
}

// CreateCategory crea una categoría.
func (h *BudgetHandlers) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	cat, err := h.categories.Create(r.Context(), &models.Category{
		CategoryGroupID: req.CategoryGroupID,
		Name:            req.Name,
		Icon:            req.Icon,
		TargetAmount:    req.TargetAmount,
		ServiceIDs:      req.ServiceIDs,
		AccountID:       req.AccountID,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, cat)
}

// UpdateCategory actualiza una categoría.
func (h *BudgetHandlers) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req categoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	cat, err := h.categories.Update(r.Context(), &models.Category{
		ID:              id,
		CategoryGroupID: req.CategoryGroupID,
		Name:            req.Name,
		Icon:            req.Icon,
		TargetAmount:    req.TargetAmount,
		ServiceIDs:      req.ServiceIDs,
		AccountID:       req.AccountID,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if cat == nil {
		respondError(w, http.StatusNotFound, "not_found", "Categoría no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, cat)
}

// DeleteCategory archiva o elimina una categoría.
func (h *BudgetHandlers) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.categories.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Categoría eliminada"})
}

// SuggestedAssignment responde la sugerencia de asignación de una categoría
// (SPEC-098): suma de la factura más reciente por cada servicio vinculado,
// agrupada por moneda.
func (h *BudgetHandlers) SuggestedAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de categoría inválido")
		return
	}
	suggestion, err := h.categories.SuggestedAssignment(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, suggestion)
}

// ---- Cuentas ----

type accountRequest struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	CurrencyID      int64   `json:"currency_id"`
	StartingBalance float64 `json:"starting_balance"`
}

// ListAccounts responde todas las cuentas.
func (h *BudgetHandlers) ListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.accounts.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, accounts)
}

// CreateAccount crea una cuenta.
func (h *BudgetHandlers) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req accountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	account, err := h.accounts.Create(r.Context(), &models.Account{
		Name:            req.Name,
		Type:            req.Type,
		CurrencyID:      req.CurrencyID,
		StartingBalance: req.StartingBalance,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, account)
}

// UpdateAccount actualiza una cuenta.
func (h *BudgetHandlers) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req accountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	account, err := h.accounts.Update(r.Context(), &models.Account{
		ID:              id,
		Name:            req.Name,
		Type:            req.Type,
		CurrencyID:      req.CurrencyID,
		StartingBalance: req.StartingBalance,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if account == nil {
		respondError(w, http.StatusNotFound, "not_found", "Cuenta no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, account)
}

// DeleteAccount elimina una cuenta.
func (h *BudgetHandlers) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.accounts.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Cuenta eliminada"})
}

// ---- Transacciones ----

type transactionRequest struct {
	AccountID  int64   `json:"account_id"`
	CategoryID *int64  `json:"category_id"`
	CurrencyID int64   `json:"currency_id"`
	Date       string  `json:"date"`
	Payee      string  `json:"payee"`
	Memo       string  `json:"memo"`
	Outflow    float64 `json:"outflow"`
	Inflow     float64 `json:"inflow"`
	Cleared    bool    `json:"cleared"`
}

// ListTransactions responde las transacciones de un mes.
func (h *BudgetHandlers) ListTransactions(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil || year < 1900 || year > 3000 {
		respondError(w, http.StatusBadRequest, "invalid_year", "Año inválido")
		return
	}
	month, err := strconv.Atoi(r.URL.Query().Get("month"))
	if err != nil || month < 1 || month > 12 {
		respondError(w, http.StatusBadRequest, "invalid_month", "Mes inválido")
		return
	}
	transactions, err := h.transactions.ListByMonth(r.Context(), year, month)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, transactions)
}

// ListCategoryTransactions responde el historial de una categoría en un mes.
func (h *BudgetHandlers) ListCategoryTransactions(w http.ResponseWriter, r *http.Request) {
	categoryID, err := strconv.ParseInt(r.PathValue("category_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de categoría inválido")
		return
	}
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_year", "Año inválido")
		return
	}
	month, err := strconv.Atoi(r.URL.Query().Get("month"))
	if err != nil || month < 1 || month > 12 {
		respondError(w, http.StatusBadRequest, "invalid_month", "Mes inválido")
		return
	}
	transactions, err := h.transactions.ListByCategoryMonth(r.Context(), categoryID, year, month)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, transactions)
}

// CreateTransaction crea una transacción.
func (h *BudgetHandlers) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req transactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	tx, err := h.transactions.Create(r.Context(), &models.Transaction{
		AccountID:  req.AccountID,
		CategoryID: req.CategoryID,
		CurrencyID: req.CurrencyID,
		Date:       req.Date,
		Payee:      req.Payee,
		Memo:       req.Memo,
		Outflow:    req.Outflow,
		Inflow:     req.Inflow,
		Cleared:    req.Cleared,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, tx)
}

// UpdateTransaction actualiza una transacción.
func (h *BudgetHandlers) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req transactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	tx, err := h.transactions.Update(r.Context(), &models.Transaction{
		ID:         id,
		AccountID:  req.AccountID,
		CategoryID: req.CategoryID,
		CurrencyID: req.CurrencyID,
		Date:       req.Date,
		Payee:      req.Payee,
		Memo:       req.Memo,
		Outflow:    req.Outflow,
		Inflow:     req.Inflow,
		Cleared:    req.Cleared,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if tx == nil {
		respondError(w, http.StatusNotFound, "not_found", "Transacción no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, tx)
}

// DeleteTransaction elimina una transacción.
func (h *BudgetHandlers) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.transactions.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Transacción eliminada"})
}
