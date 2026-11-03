package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

const maxImportSize = 2 << 20 // 2 MB

type previewTextRequest struct {
	Text    string `json:"text"`
	Year    int    `json:"year"`
	Month   int    `json:"month"`
	ChildID int64  `json:"child_id"`
}

type createKeywordRequest struct {
	PensionCategoryID int64  `json:"pension_category_id"`
	Keyword           string `json:"keyword"`
}

// PensionImportHandlers agrupa los handlers del importador (SPEC-119).
type PensionImportHandlers struct {
	service *services.PensionImportService
}

// NewPensionImportHandlers crea un nuevo PensionImportHandlers.
func NewPensionImportHandlers(service *services.PensionImportService) *PensionImportHandlers {
	return &PensionImportHandlers{service: service}
}

// Preview parsea el archivo y devuelve la propuesta sin escribir registros.
func (h *PensionImportHandlers) Preview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportSize+1024)

	in := services.PensionImportPreviewInput{}
	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxImportSize); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_request", "No se pudo leer el archivo.")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			respondError(w, http.StatusBadRequest, "missing_file", "Falta el archivo.")
			return
		}
		defer file.Close()
		content, err := io.ReadAll(io.LimitReader(file, maxImportSize))
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid_file", "No se pudo leer el archivo.")
			return
		}
		in.Filename = header.Filename
		in.MIME = header.Header.Get("Content-Type")
		in.Content = content
		in.Year = atoiOrZero(r.FormValue("year"))
		in.Month = atoiOrZero(r.FormValue("month"))
		in.ChildID = int64(atoiOrZero(r.FormValue("child_id")))
	} else {
		var req previewTextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido.")
			return
		}
		in.Text = req.Text
		in.Year = req.Year
		in.Month = req.Month
		in.ChildID = req.ChildID
	}

	preview, err := h.service.Preview(r.Context(), in)
	if err != nil {
		writeImportError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, preview)
}

// Confirm crea los registros del lote (todo o nada).
func (h *PensionImportHandlers) Confirm(w http.ResponseWriter, r *http.Request) {
	var req models.PensionImportConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido.")
		return
	}
	result, err := h.service.Confirm(r.Context(), req)
	if err != nil {
		writeImportError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// ListBatches devuelve los últimos lotes importados.
func (h *PensionImportHandlers) ListBatches(w http.ResponseWriter, r *http.Request) {
	limit := atoiOrZero(r.URL.Query().Get("limit"))
	batches, err := h.service.ListBatches(r.Context(), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, batches)
}

// TemplateCSV descarga la plantilla del formato canónico.
func (h *PensionImportHandlers) TemplateCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="pension-import-template.csv"`)
	_, _ = w.Write([]byte(h.service.TemplateCSV()))
}

// ExportCSV exporta el mes en el formato canónico.
func (h *PensionImportHandlers) ExportCSV(w http.ResponseWriter, r *http.Request) {
	year := atoiOrZero(r.URL.Query().Get("year"))
	month := atoiOrZero(r.URL.Query().Get("month"))
	if year == 0 || month == 0 {
		respondError(w, http.StatusBadRequest, "invalid_period", "Indicá year y month.")
		return
	}
	content, err := h.service.ExportCSV(r.Context(), year, month)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="pension-export.csv"`)
	_, _ = w.Write([]byte(content))
}

// ListKeywords devuelve el diccionario de palabras clave.
func (h *PensionImportHandlers) ListKeywords(w http.ResponseWriter, r *http.Request) {
	keywords, err := h.service.ListKeywords(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, keywords)
}

// CreateKeyword agrega una palabra clave.
func (h *PensionImportHandlers) CreateKeyword(w http.ResponseWriter, r *http.Request) {
	var req createKeywordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido.")
		return
	}
	keyword, err := h.service.CreateKeyword(r.Context(), req.PensionCategoryID, req.Keyword)
	if err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, keyword)
}

// DeleteKeyword elimina una palabra clave.
func (h *PensionImportHandlers) DeleteKeyword(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido.")
		return
	}
	if err := h.service.DeleteKeyword(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Palabra clave eliminada"})
}

func writeImportError(w http.ResponseWriter, err error) {
	var ie *services.ImportError
	if !errors.As(err, &ie) {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	switch ie.Code {
	case "batch_invalid":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":      ie.Code,
			"message":    ie.Message,
			"validation": ie.Validation,
		})
	case "batch_not_found":
		respondError(w, http.StatusNotFound, ie.Code, ie.Message)
	case "month_closed", "already_confirmed", "confirm_failed":
		respondError(w, http.StatusConflict, ie.Code, ie.Message)
	case "pdf_without_text", "unsupported_format", "invalid_pdf", "invalid_csv":
		respondError(w, http.StatusUnprocessableEntity, ie.Code, ie.Message)
	default:
		respondError(w, http.StatusBadRequest, ie.Code, ie.Message)
	}
}

func atoiOrZero(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}
