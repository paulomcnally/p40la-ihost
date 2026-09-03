package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type markSalaryReceivedRequest struct {
	ReceivedAt     string   `json:"received_at"`
	ReceivedAmount *float64 `json:"received_amount"`
	Notes          *string  `json:"notes"`
}

// SalaryPaymentHandlers agrupa los handlers de pagos de salario.
type SalaryPaymentHandlers struct {
	service *services.SalaryPaymentService
}

// NewSalaryPaymentHandlers crea un nuevo SalaryPaymentHandlers.
func NewSalaryPaymentHandlers(service *services.SalaryPaymentService) *SalaryPaymentHandlers {
	return &SalaryPaymentHandlers{service: service}
}

// ListSalaryPayments responde con los pagos de salario de un período.
func (h *SalaryPaymentHandlers) ListSalaryPayments(w http.ResponseWriter, r *http.Request) {
	year, month := periodFromQuery(r)
	var salaryID int64
	if v := r.URL.Query().Get("salary_id"); v != "" {
		salaryID, _ = strconv.ParseInt(v, 10, 64)
	}

	payments, err := h.service.List(r.Context(), year, month, salaryID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, payments)
}

// GetSalaryPayment responde con un pago por ID.
func (h *SalaryPaymentHandlers) GetSalaryPayment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	payment, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if payment == nil {
		respondError(w, http.StatusNotFound, "not_found", "Pago no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, payment)
}

// MarkSalaryReceived marca un pago de salario como recibido.
func (h *SalaryPaymentHandlers) MarkSalaryReceived(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req markSalaryReceivedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	receivedAt, err := parseDateTime(req.ReceivedAt)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_date", "La fecha de recepción es requerida")
		return
	}

	payment, err := h.service.MarkReceived(r.Context(), id, receivedAt, req.ReceivedAmount, req.Notes)
	if err != nil {
		if err == services.ErrMonthClosed {
			respondError(w, http.StatusConflict, "month_closed", err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, payment)
}

// MarkSalaryPending devuelve un pago de salario a estado pendiente.
func (h *SalaryPaymentHandlers) MarkSalaryPending(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	payment, err := h.service.MarkPending(r.Context(), id)
	if err != nil {
		if err == services.ErrMonthClosed {
			respondError(w, http.StatusConflict, "month_closed", err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, payment)
}
