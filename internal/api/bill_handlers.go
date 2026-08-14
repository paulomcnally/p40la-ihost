package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// BillHandlers agrupa los handlers de facturas.
type BillHandlers struct {
	service *services.BillService
}

// NewBillHandlers crea un nuevo BillHandlers.
func NewBillHandlers(service *services.BillService) *BillHandlers {
	return &BillHandlers{service: service}
}

type billRequest struct {
	ServiceID     int64   `json:"service_id"`
	Year          int     `json:"year"`
	Month         int     `json:"month"`
	Amount        float64 `json:"amount"`
	InvoiceNumber string  `json:"invoice_number"`
	Status        string  `json:"status"`
	DriveURL      string  `json:"drive_url"`
}

func (h *BillHandlers) toModel(req billRequest, id int64) *models.Bill {
	return &models.Bill{
		ID:            id,
		ServiceID:     req.ServiceID,
		Year:          req.Year,
		Month:         req.Month,
		Amount:        req.Amount,
		InvoiceNumber: req.InvoiceNumber,
		Status:        req.Status,
		DriveURL:      req.DriveURL,
	}
}

// ListBills responde con las facturas de un servicio.
func (h *BillHandlers) ListBills(w http.ResponseWriter, r *http.Request) {
	serviceID, err := strconv.ParseInt(r.PathValue("service_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_service_id", "service_id inválido")
		return
	}
	bills, err := h.service.ListByService(r.Context(), serviceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, bills)
}

// GetBill responde con una factura por ID.
func (h *BillHandlers) GetBill(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	bill, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if bill == nil {
		respondError(w, http.StatusNotFound, "not_found", "Factura no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, bill)
}

// CreateBill crea una nueva factura.
func (h *BillHandlers) CreateBill(w http.ResponseWriter, r *http.Request) {
	var req billRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	bill, err := h.service.Create(r.Context(), h.toModel(req, 0))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, bill)
}

// UpdateBill actualiza una factura existente.
func (h *BillHandlers) UpdateBill(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req billRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	bill, err := h.service.Update(r.Context(), h.toModel(req, id))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if bill == nil {
		respondError(w, http.StatusNotFound, "not_found", "Factura no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, bill)
}

// DeleteBill elimina lógicamente una factura.
func (h *BillHandlers) DeleteBill(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Factura eliminada"})
}
