package workrequest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type handlers struct {
	service *Service
	limiter *uploadRateLimiter
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
	mux.Handle("POST /api/homes/{id}/workrequests/{work_request_id}/attachments/evidence", auth(http.HandlerFunc(h.uploadEvidence)))
	mux.Handle("POST /api/homes/{id}/workrequests/{work_request_id}/upload-link", auth(http.HandlerFunc(h.uploadLink)))
	mux.Handle("GET /api/homes/{id}/workrequests/{work_request_id}/attachments/{attachment_id}", auth(http.HandlerFunc(h.downloadAttachment)))
	mux.Handle("DELETE /api/homes/{id}/workrequests/{work_request_id}/attachments/{attachment_id}", auth(http.HandlerFunc(h.deleteAttachment)))

	// Endpoint público de carga: se autentica con el token firmado del header,
	// nunca con sesión ni API key (ADR-004, ADR-010).
	mux.Handle("POST "+UploadLinkPath, http.HandlerFunc(h.uploadByToken))
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
	var req CreateInput
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	item, err := h.service.Create(r.Context(), homeID, req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	status := http.StatusCreated
	if item.Deduplicated {
		status = http.StatusOK
	}
	respondJSON(w, status, item)
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

func (h *handlers) uploadEvidence(w http.ResponseWriter, r *http.Request) {
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

	attachment, deduplicated, err := h.service.AddEvidence(r.Context(), homeID, id, header.Filename, file)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	status := http.StatusCreated
	if deduplicated {
		status = http.StatusOK
	}
	respondJSON(w, status, map[string]any{
		"attachment":   attachment,
		"deduplicated": deduplicated,
	})
}

func (h *handlers) uploadLink(w http.ResponseWriter, r *http.Request) {
	homeID, id, ok := h.ids(w, r)
	if !ok {
		return
	}
	var req UploadLinkRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
			return
		}
	}
	link, err := h.service.MintUploadLink(r.Context(), homeID, id, req)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, link)
}

func (h *handlers) uploadByToken(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	defer func() {
		// Log fijo y sin datos de negocio: nunca token, archivo ni contenido.
		slog.Info("work_request_upload",
			"method", r.Method,
			"path", UploadLinkPath,
			"status", recorder.status,
			"duration_ms", time.Since(started).Milliseconds())
	}()
	w = recorder

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	token := bearerToken(r)
	if token == "" {
		respondError(w, http.StatusUnauthorized, "invalid_token", "Falta el token de carga")
		return
	}
	if allowed, retryAfter := h.limiter.allow(hashToken(token)); !allowed {
		seconds := int(retryAfter.Seconds() + 0.999)
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
		respondError(w, http.StatusTooManyRequests, "too_many_requests", "Demasiadas solicitudes de carga")
		return
	}

	claims, row, _, err := h.service.ValidateUploadToken(r.Context(), token)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		respondError(w, http.StatusUnsupportedMediaType, "unsupported_media", "Se requiere multipart/form-data")
		return
	}

	maxRequestBytes := int64(row.MaxFiles)*MaxAttachmentBytes + (1 << 20)
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	reader, err := r.MultipartReader()
	if err != nil {
		respondError(w, http.StatusUnsupportedMediaType, "unsupported_media", "Se requiere multipart/form-data")
		return
	}

	results := make([]UploadFileResult, 0)
	added, duplicates, rejected := 0, 0, 0
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				respondError(w, http.StatusRequestEntityTooLarge, "request_too_large", "El lote supera el tamaño máximo")
				return
			}
			respondError(w, http.StatusBadRequest, "invalid_file", "No se pudo leer el multipart")
			return
		}
		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}
		result := h.service.AddEvidenceWithToken(r.Context(), claims, part.FileName(), part)
		_ = part.Close()
		switch result.Status {
		case "added":
			added++
		case "duplicate":
			duplicates++
		default:
			rejected++
		}
		results = append(results, *result)
	}
	if len(results) == 0 {
		respondError(w, http.StatusBadRequest, "invalid_file", "No se encontró el archivo")
		return
	}
	if len(results) == 1 && results[0].Error == "request_too_large" {
		respondError(w, http.StatusRequestEntityTooLarge, "request_too_large", results[0].Message)
		return
	}

	item, err := h.service.Get(r.Context(), claims.HomeID, claims.WorkRequestID)
	if err != nil {
		respondModuleError(w, err)
		return
	}
	remaining := MaxAttachmentsPerRequest - item.AttachmentCount
	if remaining < 0 {
		remaining = 0
	}
	respondJSON(w, http.StatusOK, UploadResult{
		HomeID:          claims.HomeID,
		WorkRequestID:   claims.WorkRequestID,
		Added:           added,
		Duplicates:      duplicates,
		Rejected:        rejected,
		AttachmentCount: item.AttachmentCount,
		RemainingSlots:  remaining,
		Results:         results,
	})
}

func bearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(header) < 7 || !strings.EqualFold(header[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
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
	if r.Header.Get("X-P40LA-Source") == "mcp" {
		err = h.service.DeleteEvidenceAttachment(r.Context(), homeID, id, attachmentID)
	} else {
		err = h.service.DeleteAttachment(r.Context(), homeID, id, attachmentID)
	}
	if err != nil {
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
		if validation.Status != 0 {
			status = validation.Status
		}
		switch validation.Code {
		case "attachment_limit", "upload_token_exhausted":
			status = http.StatusConflict
		case "request_too_large":
			status = http.StatusRequestEntityTooLarge
		case "invalid_token", "expired_token":
			status = http.StatusUnauthorized
		case "unsupported_media":
			status = http.StatusUnsupportedMediaType
		case "upload_link_disabled":
			status = http.StatusServiceUnavailable
		case "too_many_requests":
			status = http.StatusTooManyRequests
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

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

type uploadRateWindow struct {
	count int
	reset time.Time
}

// uploadRateLimiter aplica una ventana fija en memoria por hash de token. No
// usa X-Forwarded-For: es falsificable sin un proxy confiable (REQ-023).
type uploadRateLimiter struct {
	mu      sync.Mutex
	windows map[string]*uploadRateWindow
	limit   int
	window  time.Duration
	now     func() time.Time
}

func newUploadRateLimiter(limit int, window time.Duration) *uploadRateLimiter {
	return &uploadRateLimiter{
		windows: make(map[string]*uploadRateWindow),
		limit:   limit,
		window:  window,
		now:     time.Now,
	}
}

func (l *uploadRateLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	window, ok := l.windows[key]
	if !ok || !now.Before(window.reset) {
		l.windows[key] = &uploadRateWindow{count: 1, reset: now.Add(l.window)}
		return true, 0
	}
	if window.count >= l.limit {
		return false, window.reset.Sub(now)
	}
	window.count++
	return true, 0
}
