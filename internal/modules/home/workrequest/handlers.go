package workrequest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type handlers struct {
	service *Service
}

// RegisterRoutes monta los endpoints del módulo (ARCH-MODULES).
func (h *handlers) register(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/homes/{id}/workrequests", auth(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/homes/{id}/workrequests", auth(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/homes/{id}/workrequests/{work_request_id}", auth(http.HandlerFunc(h.get)))
	mux.Handle("PUT /api/homes/{id}/workrequests/{work_request_id}", auth(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/homes/{id}/workrequests/{work_request_id}", auth(http.HandlerFunc(h.delete)))
	mux.Handle("POST /api/homes/{id}/workrequests/{work_request_id}/convert", auth(http.HandlerFunc(h.convert)))
	mux.Handle("POST /api/homes/{id}/workrequests/{work_request_id}/status", auth(http.HandlerFunc(h.setStatus)))
	mux.Handle("POST /api/homes/{id}/workrequests/{work_request_id}/attachments", auth(http.HandlerFunc(h.uploadAttachment)))
	mux.Handle("GET /api/homes/{id}/workrequests/{work_request_id}/attachments/{attachment_id}", auth(http.HandlerFunc(h.downloadAttachment)))
	mux.Handle("DELETE /api/homes/{id}/workrequests/{work_request_id}/attachments/{attachment_id}", auth(http.HandlerFunc(h.deleteAttachment)))
}

func (h *handlers) list(w http.ResponseWriter, r *http.Request) {
	homeID, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	filter := Filter{
		Status:   r.URL.Query().Get("status"),
		Category: r.URL.Query().Get("category"),
	}
	result, err := h.service.List(r.Context(), homeID, filter)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *handlers) get(w http.ResponseWriter, r *http.Request) {
	homeID, id, ok := h.ids(w, r)
	if !ok {
		return
	}
	item, err := h.service.Get(r.Context(), homeID, id)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) create(w http.ResponseWriter, r *http.Request) {
	homeID, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req Input
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.Create(r.Context(), homeID, req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, item)
}

func (h *handlers) update(w http.ResponseWriter, r *http.Request) {
	homeID, id, ok := h.ids(w, r)
	if !ok {
		return
	}
	var req Input
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.Update(r.Context(), homeID, id, req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) delete(w http.ResponseWriter, r *http.Request) {
	homeID, id, ok := h.ids(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), homeID, id); err != nil {
		respondModuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) convert(w http.ResponseWriter, r *http.Request) {
	homeID, id, ok := h.ids(w, r)
	if !ok {
		return
	}
	var req ConvertInput
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	result, err := h.service.Convert(r.Context(), homeID, id, req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *handlers) setStatus(w http.ResponseWriter, r *http.Request) {
	homeID, id, ok := h.ids(w, r)
	if !ok {
		return
	}
	var req StatusInput
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.SetStatus(r.Context(), homeID, id, req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (h *handlers) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	homeID, id, ok := h.ids(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxAttachmentBytes+2*1024*1024)
	file, header, err := r.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			respondError(w, http.StatusRequestEntityTooLarge, "request_too_large", "El archivo supera el máximo de 25 MB")
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_file", "No se encontró el archivo")
		return
	}
	defer file.Close()

	attachment, err := h.service.AddAttachment(
		r.Context(), homeID, id, strings.TrimSpace(r.FormValue("media_type")), header.Filename, file)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, attachment)
}

func (h *handlers) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	homeID, id, ok := h.ids(w, r)
	if !ok {
		return
	}
	attachmentID, err := pathID(r, "attachment_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	attachment, path, err := h.service.AttachmentPath(r.Context(), homeID, id, attachmentID)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	disposition := "inline"
	if r.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Type", attachment.MimeType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", disposition+`; filename="`+strings.ReplaceAll(attachment.FileName, `"`, "")+`"`)
	http.ServeFile(w, r, path)
}

func (h *handlers) deleteAttachment(w http.ResponseWriter, r *http.Request) {
	homeID, id, ok := h.ids(w, r)
	if !ok {
		return
	}
	attachmentID, err := pathID(r, "attachment_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.DeleteAttachment(r.Context(), homeID, id, attachmentID); err != nil {
		respondModuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) ids(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	homeID, err := pathID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return 0, 0, false
	}
	id, err := pathID(r, "work_request_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return 0, 0, false
	}
	return homeID, id, true
}

func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(dst)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, errorBody{Error: code, Message: message})
}

func respondModuleError(w http.ResponseWriter, err error) {
	var validation *ValidationError
	if errors.As(err, &validation) {
		status := http.StatusBadRequest
		switch validation.Code {
		case "attachment_limit":
			status = http.StatusConflict
		case "request_too_large":
			status = http.StatusRequestEntityTooLarge
		}
		respondError(w, status, validation.Code, validation.Message)
		return
	}
	switch {
	case errors.Is(err, ErrHomeNotFound):
		respondError(w, http.StatusNotFound, "home_not_found", "Vivienda no encontrada")
	case errors.Is(err, ErrWorkRequestNotFound):
		respondError(w, http.StatusNotFound, "work_request_not_found", "Reporte no encontrado")
	case errors.Is(err, ErrAttachmentNotFound):
		respondError(w, http.StatusNotFound, "attachment_not_found", "Adjunto no encontrado")
	case errors.Is(err, ErrAlreadyConverted):
		respondError(w, http.StatusConflict, "already_converted", "El reporte ya fue convertido")
	default:
		respondError(w, http.StatusInternalServerError, "internal_error", "Error interno")
	}
}
