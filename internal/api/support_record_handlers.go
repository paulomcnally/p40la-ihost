package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type createSupportRecordRequest struct {
	ChildID           int64   `json:"child_id"`
	PensionCategoryID int64   `json:"pension_category_id"`
	Year              int     `json:"year"`
	Month             int     `json:"month"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	Notes             string  `json:"notes"`
}

type updateSupportRecordRequest struct {
	Amount            float64 `json:"amount"`
	PensionCategoryID int64   `json:"pension_category_id"`
	Notes             *string `json:"notes"`
}

type markPaidRequest struct {
	PaidAt           string   `json:"paid_at"`
	PaymentMethod    string   `json:"payment_method"`
	PaymentReference string   `json:"payment_reference"`
	EvidenceNotes    string   `json:"evidence_notes"`
	OriginalAmount   *float64 `json:"original_amount"`
	OriginalCurrency string   `json:"original_currency"`
	ExchangeRate     *float64 `json:"exchange_rate"`
}

type markRejectedRequest struct {
	Reason string `json:"reason"`
}

// SupportRecordHandlers agrupa los handlers de registros de manutención.
type SupportRecordHandlers struct {
	service *services.SupportRecordService
}

// NewSupportRecordHandlers crea un nuevo SupportRecordHandlers.
func NewSupportRecordHandlers(service *services.SupportRecordService) *SupportRecordHandlers {
	return &SupportRecordHandlers{service: service}
}

// ListRecords responde con los registros de un período.
func (h *SupportRecordHandlers) ListRecords(w http.ResponseWriter, r *http.Request) {
	year, month := periodFromQuery(r)
	var childID int64
	if v := r.URL.Query().Get("child_id"); v != "" {
		childID, _ = strconv.ParseInt(v, 10, 64)
	}

	records, err := h.service.List(r.Context(), year, month, childID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, records)
}

// GetRecord responde con un registro por ID.
func (h *SupportRecordHandlers) GetRecord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	record, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if record == nil {
		respondError(w, http.StatusNotFound, "not_found", "Registro no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, record)
}

// CreateRecord crea un nuevo registro de manutención.
func (h *SupportRecordHandlers) CreateRecord(w http.ResponseWriter, r *http.Request) {
	var req createSupportRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	record, err := h.service.Create(r.Context(), req.ChildID, req.PensionCategoryID, req.Year, req.Month, req.Amount, req.Currency, req.Notes)
	if err != nil {
		if err == services.ErrMonthClosed {
			respondError(w, http.StatusConflict, "month_closed", err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, record)
}

// UpdateRecord actualiza un registro existente.
func (h *SupportRecordHandlers) UpdateRecord(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req updateSupportRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	record, err := h.service.Update(r.Context(), id, req.Amount, req.PensionCategoryID, req.Notes)
	if err != nil {
		if err == services.ErrMonthClosed {
			respondError(w, http.StatusConflict, "month_closed", err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, record)
}

// MarkPaid marca un registro como pagado.
func (h *SupportRecordHandlers) MarkPaid(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req markPaidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	var paidAt *time.Time
	if req.PaidAt != "" {
		t, err := parseDateTime(req.PaidAt)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_date", "Fecha de pago inválida")
			return
		}
		paidAt = &t
	}

	record, err := h.service.MarkPaid(r.Context(), id, paidAt, req.PaymentMethod, req.PaymentReference, req.EvidenceNotes, req.OriginalAmount, req.OriginalCurrency, req.ExchangeRate)
	if err != nil {
		if err == services.ErrMonthClosed {
			respondError(w, http.StatusConflict, "month_closed", err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, record)
}

// MarkPending devuelve un registro a estado pendiente.
func (h *SupportRecordHandlers) MarkPending(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	record, err := h.service.MarkPending(r.Context(), id)
	if err != nil {
		if err == services.ErrMonthClosed {
			respondError(w, http.StatusConflict, "month_closed", err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, record)
}

// MarkRejected marca un registro como rechazado.
func (h *SupportRecordHandlers) MarkRejected(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req markRejectedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	record, err := h.service.MarkRejected(r.Context(), id, req.Reason)
	if err != nil {
		if err == services.ErrMonthClosed {
			respondError(w, http.StatusConflict, "month_closed", err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, record)
}

// UploadProof sube un comprobante para un registro.
func (h *SupportRecordHandlers) UploadProof(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_file", "No se encontró el archivo")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_file", "Error al leer el archivo")
		return
	}

	record, err := h.service.SaveProof(r.Context(), id, header.Filename, data)
	if err != nil {
		if err == services.ErrMonthClosed {
			respondError(w, http.StatusConflict, "month_closed", err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_file", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"proof_file_name": header.Filename,
		"id":              record.ID,
		"status":          record.Status,
	})
}

// DownloadProof descarga el comprobante de un registro.
func (h *SupportRecordHandlers) DownloadProof(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	filePath, fileName, err := h.service.ProofPath(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+fileName+`"`)
	http.ServeFile(w, r, filePath)
}

// periodFromQuery extrae year/month de la query, default al período actual.
func periodFromQuery(r *http.Request) (year, month int) {
	now := time.Now()
	year = now.Year()
	month = int(now.Month())

	if v := r.URL.Query().Get("year"); v != "" {
		if y, err := strconv.Atoi(v); err == nil && y > 0 {
			year = y
		}
	}
	if v := r.URL.Query().Get("month"); v != "" {
		if m, err := strconv.Atoi(v); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}
	return year, month
}

// parseDateTime parsea fechas en formato RFC3339 o datetime-local sin segundos.
func parseDateTime(s string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, &time.ParseError{}
}
