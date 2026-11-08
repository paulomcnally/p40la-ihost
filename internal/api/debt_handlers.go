package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
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

// GetDebtHistory responde con la evolución mensual de deudas (SPEC-105).
func (h *DebtHandlers) GetDebtHistory(w http.ResponseWriter, r *http.Request) {
	months := 12
	if raw := r.URL.Query().Get("months"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 24 {
			respondError(w, http.StatusBadRequest, "invalid_months", "months debe estar entre 1 y 24")
			return
		}
		months = parsed
	}

	history, err := h.service.History(r.Context(), months, time.Now())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, history)
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

// PayDebtBill marca una cuota como pagada y, opcionalmente, adjunta la factura
// y/o el comprobante PDF (SPEC-122). Acepta JSON (compatibilidad) o
// multipart/form-data con los campos invoice y receipt.
func (h *DebtHandlers) PayDebtBill(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}

	var req payDebtBillRequest
	var invoice, receipt *services.PDFUpload

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, 2*services.MaxDebtDocumentBytes+(1<<20))
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				respondError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Los documentos no pueden superar los 10 MB cada uno")
				return
			}
			respondError(w, http.StatusBadRequest, "invalid_request", "No se pudo leer el formulario")
			return
		}
		defer r.MultipartForm.RemoveAll()

		req.PaidAt = r.FormValue("paid_at")
		req.PaymentReference = r.FormValue("payment_reference")

		var invoiceCloser, receiptCloser io.Closer
		invoice, invoiceCloser, err = formFileUpload(r, "invoice")
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_file", "No se pudo leer la factura")
			return
		}
		if invoiceCloser != nil {
			defer invoiceCloser.Close()
		}
		receipt, receiptCloser, err = formFileUpload(r, "receipt")
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_file", "No se pudo leer el comprobante")
			return
		}
		if receiptCloser != nil {
			defer receiptCloser.Close()
		}
	} else if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	paidAt, err := time.Parse("2006-01-02", req.PaidAt)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_paid_at", "La fecha de pago es obligatoria (formato YYYY-MM-DD)")
		return
	}

	bill, err := h.service.PayBillWithDocuments(r.Context(), id, paidAt, req.PaymentReference, invoice, receipt)
	if err != nil {
		switch {
		case err.Error() == "la cuota no existe":
			respondError(w, http.StatusNotFound, "not_found", err.Error())
		case err.Error() == "la cuota ya está pagada":
			respondError(w, http.StatusBadRequest, "already_paid", err.Error())
		case errors.Is(err, services.ErrDocumentAfterPayment):
			respondError(w, http.StatusInternalServerError, "document_save_failed", err.Error())
		default:
			respondError(w, http.StatusBadRequest, "invalid_file", err.Error())
		}
		return
	}
	respondJSON(w, http.StatusOK, bill)
}

func formFileUpload(r *http.Request, field string) (*services.PDFUpload, io.Closer, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return &services.PDFUpload{FileName: header.Filename, Reader: file}, file, nil
}

// DownloadDebtInvoice sirve la factura PDF de una cuota (SPEC-122).
func (h *DebtHandlers) DownloadDebtInvoice(w http.ResponseWriter, r *http.Request) {
	h.downloadDebtDocument(w, r, services.DebtDocumentInvoice)
}

// DownloadDebtReceipt sirve el comprobante PDF de una cuota (SPEC-122).
func (h *DebtHandlers) DownloadDebtReceipt(w http.ResponseWriter, r *http.Request) {
	h.downloadDebtDocument(w, r, services.DebtDocumentReceipt)
}

func (h *DebtHandlers) downloadDebtDocument(w http.ResponseWriter, r *http.Request, kind services.DebtDocumentKind) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}

	filePath, fileName, err := h.service.DocumentPath(r.Context(), id, kind)
	if err != nil {
		if errors.Is(err, services.ErrDocumentNotFound) {
			respondError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	disposition := "inline"
	if r.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, fileName))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, filePath)
}

// UploadDebtInvoice adjunta o reemplaza la factura de una cuota ya pagada.
func (h *DebtHandlers) UploadDebtInvoice(w http.ResponseWriter, r *http.Request) {
	h.uploadDebtDocument(w, r, services.DebtDocumentInvoice)
}

// UploadDebtReceipt adjunta o reemplaza el comprobante de una cuota ya pagada
// (SPEC-122 REQ-013).
func (h *DebtHandlers) UploadDebtReceipt(w http.ResponseWriter, r *http.Request) {
	h.uploadDebtDocument(w, r, services.DebtDocumentReceipt)
}

func (h *DebtHandlers) uploadDebtDocument(w http.ResponseWriter, r *http.Request, kind services.DebtDocumentKind) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		respondError(w, http.StatusBadRequest, "invalid_request", "Se esperaba multipart/form-data")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, services.MaxDebtDocumentBytes+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			respondError(w, http.StatusRequestEntityTooLarge, "request_too_large", "El documento no puede superar los 10 MB")
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_request", "No se pudo leer el formulario")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_file", "No se encontró el archivo")
		return
	}
	defer file.Close()

	bill, err := h.service.SaveDocument(r.Context(), id, kind, services.PDFUpload{
		FileName: header.Filename,
		Reader:   file,
	})
	if err != nil {
		switch {
		case err.Error() == "la cuota no existe":
			respondError(w, http.StatusNotFound, "not_found", err.Error())
		case err.Error() == "la cuota debe estar pagada para adjuntar un documento":
			respondError(w, http.StatusBadRequest, "not_paid", err.Error())
		case err.Error() == "tipo de documento inválido":
			respondError(w, http.StatusBadRequest, "invalid_kind", err.Error())
		default:
			respondError(w, http.StatusBadRequest, "invalid_file", err.Error())
		}
		return
	}
	respondJSON(w, http.StatusOK, bill)
}
