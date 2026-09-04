package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// DebtHandlers agrupa los handlers de deudas y sus cuotas (SPEC-054).
type DebtHandlers struct {
	service *services.DebtService
}

// NewDebtHandlers crea un nuevo DebtHandlers.
func NewDebtHandlers(service *services.DebtService) *DebtHandlers {
	return &DebtHandlers{service: service}
}

type debtRequest struct {
	InstitutionID     int64   `json:"institution_id"`
	Identifier        string  `json:"identifier"`
	Description       string  `json:"description"`
	Total             float64 `json:"total"`
	Principal         float64 `json:"principal"`
	CurrencyID        int64   `json:"currency_id"`
	InstallmentsTotal int     `json:"installments_total"`
	InstallmentAmount float64 `json:"installment_amount"`
	InterestRate      float64 `json:"interest_rate"`
	PaymentDay        int     `json:"payment_day"`
	StartDate         string  `json:"start_date"`
	Status            string  `json:"status"`
}

func (h *DebtHandlers) toModel(req debtRequest, id int64) *models.Debt {
	return &models.Debt{
		ID:                id,
		InstitutionID:     req.InstitutionID,
		Identifier:        req.Identifier,
		Description:       req.Description,
		Total:             req.Total,
		Principal:         req.Principal,
		CurrencyID:        req.CurrencyID,
		InstallmentsTotal: req.InstallmentsTotal,
		InstallmentAmount: req.InstallmentAmount,
		InterestRate:      req.InterestRate,
		PaymentDay:        req.PaymentDay,
		StartDate:         req.StartDate,
		Status:            req.Status,
	}
}

// ListDebts responde con las deudas, reconciliando cuotas antes de listar.
func (h *DebtHandlers) ListDebts(w http.ResponseWriter, r *http.Request) {
	if err := h.service.ReconcileDebtBills(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	debts, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, debts)
}

// GetDebt responde con una deuda por ID.
func (h *DebtHandlers) GetDebt(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	debt, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if debt == nil {
		respondError(w, http.StatusNotFound, "not_found", "Deuda no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, debt)
}

// CreateDebt crea una nueva deuda y genera sus cuotas.
func (h *DebtHandlers) CreateDebt(w http.ResponseWriter, r *http.Request) {
	var req debtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	debt, err := h.service.Create(r.Context(), h.toModel(req, 0))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, debt)
}

// UpdateDebt actualiza una deuda existente y regenera cuotas faltantes.
func (h *DebtHandlers) UpdateDebt(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req debtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	debt, err := h.service.Update(r.Context(), h.toModel(req, id))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if debt == nil {
		respondError(w, http.StatusNotFound, "not_found", "Deuda no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, debt)
}

// DeleteDebt elimina lógicamente una deuda y sus cuotas.
func (h *DebtHandlers) DeleteDebt(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Deuda eliminada"})
}

// ListDebtBills responde con las cuotas de una deuda.
func (h *DebtHandlers) ListDebtBills(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	bills, err := h.service.ListBills(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, bills)
}

// ListDebtBillsByMonth responde con las cuotas de un mes (vista Calendario).
func (h *DebtHandlers) ListDebtBillsByMonth(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_year", "year inválido")
		return
	}
	month, err := strconv.Atoi(r.URL.Query().Get("month"))
	if err != nil || month < 1 || month > 12 {
		respondError(w, http.StatusBadRequest, "invalid_month", "month inválido")
		return
	}
	bills, err := h.service.ListBillsByMonth(r.Context(), year, month)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, bills)
}

type payDebtBillRequest struct {
	PaidAt           string `json:"paid_at"`
	PaymentReference string `json:"payment_reference"`
}

// PayDebtBill marca una cuota como pagada.
func (h *DebtHandlers) PayDebtBill(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req payDebtBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	paidAt, err := time.Parse("2006-01-02", req.PaidAt)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_paid_at", "La fecha de pago es obligatoria (formato YYYY-MM-DD)")
		return
	}

	bill, err := h.service.PayBill(r.Context(), id, paidAt, req.PaymentReference)
	if err != nil {
		switch {
		case err.Error() == "la cuota no existe":
			respondError(w, http.StatusNotFound, "not_found", err.Error())
		case err.Error() == "la cuota ya está pagada":
			respondError(w, http.StatusBadRequest, "already_paid", err.Error())
		default:
			respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		}
		return
	}
	respondJSON(w, http.StatusOK, bill)
}
