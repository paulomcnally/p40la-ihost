package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// bulkSourceHeader identifica el origen de una operación masiva para la
// auditoría de facturas (SPEC-145). Valores permitidos: dashboard y mcp.
const bulkSourceHeader = "X-P40LA-Source"

type generateBillsRequest struct {
	From          string   `json:"from"`
	To            string   `json:"to"`
	Amount        *float64 `json:"amount"`
	DryRun        bool     `json:"dry_run"`
	AllowInactive bool     `json:"allow_inactive"`
}

type payBillsRequest struct {
	BillIDs          []int64 `json:"bill_ids"`
	Month            string  `json:"month"`
	From             string  `json:"from"`
	To               string  `json:"to"`
	AllPendingUpTo   string  `json:"all_pending_up_to"`
	PaidAt           string  `json:"paid_at"`
	DriveURL         string  `json:"drive_url"`
	PaymentReference string  `json:"payment_reference"`
	DryRun           bool    `json:"dry_run"`
}

// GenerateBills genera facturas de un servicio para un rango de períodos.
func (h *BillHandlers) GenerateBills(w http.ResponseWriter, r *http.Request) {
	serviceID, err := strconv.ParseInt(r.PathValue("service_id"), 10, 64)
	if err != nil || serviceID <= 0 {
		respondError(w, http.StatusBadRequest, "invalid_service_id", "service_id inválido")
		return
	}
	var req generateBillsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	result, err := h.service.GenerateForService(r.Context(), serviceID, services.GenerateBillsOptions{
		From:          req.From,
		To:            req.To,
		Amount:        req.Amount,
		DryRun:        req.DryRun,
		AllowInactive: req.AllowInactive,
		Source:        r.Header.Get(bulkSourceHeader),
		Now:           time.Now(),
	})
	if err != nil {
		respondBulkError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// PayBills paga facturas de un servicio según el selector recibido.
func (h *BillHandlers) PayBills(w http.ResponseWriter, r *http.Request) {
	serviceID, err := strconv.ParseInt(r.PathValue("service_id"), 10, 64)
	if err != nil || serviceID <= 0 {
		respondError(w, http.StatusBadRequest, "invalid_service_id", "service_id inválido")
		return
	}
	var req payBillsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	result, err := h.service.PayForService(r.Context(), serviceID, services.PayBillsOptions{
		BillIDs:          req.BillIDs,
		Month:            req.Month,
		From:             req.From,
		To:               req.To,
		AllPendingUpTo:   req.AllPendingUpTo,
		PaidAt:           req.PaidAt,
		DriveURL:         req.DriveURL,
		PaymentReference: req.PaymentReference,
		DryRun:           req.DryRun,
		Source:           r.Header.Get(bulkSourceHeader),
		Now:              time.Now(),
	})
	if err != nil {
		respondBulkError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func respondBulkError(w http.ResponseWriter, err error) {
	var bulkErr *services.BulkError
	if errors.As(err, &bulkErr) {
		status := http.StatusBadRequest
		switch bulkErr.Code {
		case services.BulkErrorServiceNotFound:
			status = http.StatusNotFound
		case services.BulkErrorServiceInactive:
			status = http.StatusConflict
		case services.BulkErrorAmountUnresolved:
			status = http.StatusUnprocessableEntity
		case services.BulkErrorInternal:
			status = http.StatusInternalServerError
			bulkErr.Message = "error interno al procesar la operación"
		}
		respondError(w, status, bulkErr.Code, bulkErr.Message)
		return
	}
	respondError(w, http.StatusInternalServerError, "internal_error", "error interno al procesar la operación")
}
