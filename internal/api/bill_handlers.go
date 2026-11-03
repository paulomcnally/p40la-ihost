package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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

type payBillRequest struct {
	PaidAt           string `json:"paid_at"`
	DriveURL         string `json:"drive_url"`
	PaymentReference string `json:"payment_reference"`
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

// GetBillHistory responde con el historial de auditoría de una factura (SPEC-070).
func (h *BillHandlers) GetBillHistory(w http.ResponseWriter, r *http.Request) {
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
	history, err := h.service.History(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, history)
}

// parseHomeIDQuery parsea home_id opcional del querystring.
func parseHomeIDQuery(r *http.Request) (*int64, bool) {
	raw := r.URL.Query().Get("home_id")
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, false
	}
	return &id, true
}

// ListServiceBillsByMonth responde con las facturas de todos los servicios
// cuyo período efectivo cae en el mes (SPEC-120).
func (h *BillHandlers) ListServiceBillsByMonth(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	year, err := strconv.Atoi(r.URL.Query().Get("year"))
	if err != nil {
		year = now.Year()
	}
	month, err := strconv.Atoi(r.URL.Query().Get("month"))
	if err != nil || month < 1 || month > 12 {
		if r.URL.Query().Get("month") != "" {
			respondError(w, http.StatusBadRequest, "invalid_month", "month inválido")
			return
		}
		month = int(now.Month())
	}
	homeID, ok := parseHomeIDQuery(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_home_id", "home_id inválido")
		return
	}

	bills, err := h.service.ListServiceBillsByPeriod(r.Context(), year, month, homeID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, bills)
}

// ServiceBillsSummary responde con el agregado por período y moneda del rango
// solicitado (SPEC-120).
func (h *BillHandlers) ServiceBillsSummary(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		respondError(w, http.StatusBadRequest, "invalid_range", "from y to son obligatorios (YYYY-MM)")
		return
	}
	homeID, ok := parseHomeIDQuery(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_home_id", "home_id inválido")
		return
	}

	summary, err := h.service.ServiceBillsSummary(r.Context(), from, to, homeID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_range", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, summary)
}

// ServiceBillsProjection responde con la proyección de los próximos meses
// separando facturas confirmadas de estimaciones (SPEC-120).
func (h *BillHandlers) ServiceBillsProjection(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	if raw := r.URL.Query().Get("from"); raw != "" {
		parsed, err := time.Parse("2006-01", raw)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_period", "from inválido (YYYY-MM)")
			return
		}
		year = parsed.Year()
		month = int(parsed.Month())
	}
	months := 12
	if raw := r.URL.Query().Get("months"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 24 {
			respondError(w, http.StatusBadRequest, "invalid_months", "months debe estar entre 1 y 24")
			return
		}
		months = parsed
	}
	homeID, ok := parseHomeIDQuery(r)
	if !ok {
		respondError(w, http.StatusBadRequest, "invalid_home_id", "home_id inválido")
		return
	}

	projection, err := h.service.ServiceBillsProjection(r.Context(), year, month, months, homeID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, projection)
}

// PayBill marca una factura como pagada (SPEC-043). Requiere paid_at; drive_url
// (comprobante Google Drive) y payment_reference son opcionales.
func (h *BillHandlers) PayBill(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req payBillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	paidAt, err := time.Parse("2006-01-02", req.PaidAt)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_paid_at", "La fecha de pago es obligatoria (formato YYYY-MM-DD)")
		return
	}

	bill, err := h.service.PayBill(r.Context(), id, paidAt, req.DriveURL, req.PaymentReference)
	if err != nil {
		switch {
		case err.Error() == "la factura no existe":
			respondError(w, http.StatusNotFound, "not_found", err.Error())
		case err.Error() == "la factura ya está pagada":
			respondError(w, http.StatusBadRequest, "already_paid", err.Error())
		default:
			respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		}
		return
	}
	respondJSON(w, http.StatusOK, bill)
}
