package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/analyzers"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type DocumentHandlers struct {
	doc *services.DocumentService
}

func NewDocumentHandlers(doc *services.DocumentService) *DocumentHandlers {
	return &DocumentHandlers{doc: doc}
}

func (h *DocumentHandlers) UploadAndAnalyze(w http.ResponseWriter, r *http.Request) {
	serviceID, err := strconv.ParseInt(r.PathValue("service_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de servicio inválido")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_file", "No se encontró el archivo")
		return
	}
	defer file.Close()

	file, err = h.doc.ValidateAndPrepare(file, header)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_file", err.Error())
		return
	}

	result, analyzerID, err := h.doc.UploadAndAnalyze(r.Context(), serviceID, file, header)
	if err != nil {
		respondError(w, http.StatusBadRequest, "analysis_failed", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"extracted":     result,
		"analyzer_used": analyzerID,
	})
}

func (h *DocumentHandlers) CreateBillFromExtracted(w http.ResponseWriter, r *http.Request) {
	serviceID, err := strconv.ParseInt(r.PathValue("service_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de servicio inválido")
		return
	}

	var req struct {
		Amount        float64 `json:"amount"`
		InvoiceNumber string  `json:"invoice_number"`
		Year          int     `json:"year"`
		Month         int     `json:"month"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	extracted := &analyzers.ExtractedBill{
		Amount:        req.Amount,
		InvoiceNumber: req.InvoiceNumber,
		Year:          req.Year,
		Month:         req.Month,
	}

	bill, updated, err := h.doc.CreateBillFromExtracted(r.Context(), serviceID, extracted)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":              bill.ID,
		"service_id":      bill.ServiceID,
		"year":            bill.Year,
		"month":           bill.Month,
		"amount":          bill.Amount,
		"invoice_number":  bill.InvoiceNumber,
		"status":          bill.Status,
		"created_at":      bill.CreatedAt,
		"updated_at":      bill.UpdatedAt,
		"updated":         updated,
	})
}
