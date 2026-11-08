package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// DocumentRecognitionHandlers agrupa los endpoints de reconocimiento de
// comprobantes bancarios (SPEC-127).
type DocumentRecognitionHandlers struct {
	service *services.DocumentRecognitionService
}

// NewDocumentRecognitionHandlers crea los handlers de reconocimiento.
func NewDocumentRecognitionHandlers(service *services.DocumentRecognitionService) *DocumentRecognitionHandlers {
	return &DocumentRecognitionHandlers{service: service}
}

// Recognize recibe un PDF y devuelve una vista previa sin modificar pagos.
func (h *DocumentRecognitionHandlers) Recognize(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, services.MaxRecognitionDocumentBytes+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			respondError(w, http.StatusRequestEntityTooLarge, "request_too_large", "El comprobante no puede superar los 10 MB")
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_file", "No se pudo leer el comprobante")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_file", "No se encontró el comprobante")
		return
	}
	defer file.Close()

	recognition, err := h.service.Recognize(r.Context(), file, header)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRecognitionTooLarge):
			respondError(w, http.StatusRequestEntityTooLarge, "request_too_large", err.Error())
		case errors.Is(err, services.ErrRecognitionUnsupportedFormat):
			respondError(w, http.StatusBadRequest, "unsupported_format", "El formato del comprobante no está soportado")
		case errors.Is(err, services.ErrRecognitionNeedsOCR):
			respondError(w, http.StatusBadRequest, "needs_ocr", "El PDF no tiene texto extraíble (requiere OCR)")
		case errors.Is(err, services.ErrRecognitionInvalidPDF):
			respondError(w, http.StatusBadRequest, "invalid_file", "El PDF no es válido")
		case errors.Is(err, services.ErrRecognitionInvalidFile):
			respondError(w, http.StatusBadRequest, "invalid_file", err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "internal_error", "No se pudo reconocer el comprobante")
		}
		return
	}

	respondJSON(w, http.StatusOK, recognition)
}

// Confirm asocia un reconocimiento a una cuota pagada tras la revisión del
// usuario. La cuota debe existir y estar pagada.
func (h *DocumentRecognitionHandlers) Confirm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de reconocimiento inválido")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var input models.ConfirmRecognitionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	result, err := h.service.Confirm(r.Context(), id, input)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRecognitionNotFound):
			respondError(w, http.StatusNotFound, "not_found", err.Error())
		case errors.Is(err, services.ErrRecognitionAlreadyAssociated):
			respondError(w, http.StatusConflict, "already_associated", err.Error())
		case errors.Is(err, services.ErrRecognitionInvalidTarget):
			respondError(w, http.StatusBadRequest, "invalid_target", err.Error())
		case errors.Is(err, services.ErrRecognitionBillNotPaid):
			respondError(w, http.StatusBadRequest, "not_paid", err.Error())
		case errors.Is(err, services.ErrRecognitionReferenceConflict):
			respondError(w, http.StatusBadRequest, "reference_conflict", err.Error())
		case errors.Is(err, services.ErrRecognitionReferenceInUse):
			respondError(w, http.StatusConflict, "reference_in_use", err.Error())
		case errors.Is(err, services.ErrRecognitionFileExists):
			respondError(w, http.StatusBadRequest, "file_exists", err.Error())
		case errors.Is(err, services.ErrRecognitionInvalidFile):
			respondError(w, http.StatusBadRequest, "invalid_file", err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "internal_error", "No se pudo asociar el comprobante")
		}
		return
	}

	respondJSON(w, http.StatusOK, result)
}
