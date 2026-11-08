package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// AutoAttachmentHandlers agrupa los handlers de PDFs de mantenimientos e
// inspecciones (SPEC-124).
type AutoAttachmentHandlers struct {
	service *services.AutoAttachmentService
}

// NewAutoAttachmentHandlers crea un nuevo AutoAttachmentHandlers.
func NewAutoAttachmentHandlers(service *services.AutoAttachmentService) *AutoAttachmentHandlers {
	return &AutoAttachmentHandlers{service: service}
}

// UploadMaintenanceAttachment sube un PDF a un mantenimiento.
func (h *AutoAttachmentHandlers) UploadMaintenanceAttachment(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, models.AutoAttachmentOwnerMaintenance, "maintenance_id")
}

// UploadInspectionAttachment sube un PDF a una inspección.
func (h *AutoAttachmentHandlers) UploadInspectionAttachment(w http.ResponseWriter, r *http.Request) {
	h.upload(w, r, models.AutoAttachmentOwnerInspection, "inspection_id")
}

// DownloadMaintenanceAttachment sirve un PDF de un mantenimiento.
func (h *AutoAttachmentHandlers) DownloadMaintenanceAttachment(w http.ResponseWriter, r *http.Request) {
	h.download(w, r, models.AutoAttachmentOwnerMaintenance, "maintenance_id")
}

// DownloadInspectionAttachment sirve un PDF de una inspección.
func (h *AutoAttachmentHandlers) DownloadInspectionAttachment(w http.ResponseWriter, r *http.Request) {
	h.download(w, r, models.AutoAttachmentOwnerInspection, "inspection_id")
}

// DeleteMaintenanceAttachment elimina un PDF de un mantenimiento.
func (h *AutoAttachmentHandlers) DeleteMaintenanceAttachment(w http.ResponseWriter, r *http.Request) {
	h.delete(w, r, models.AutoAttachmentOwnerMaintenance, "maintenance_id")
}

// DeleteInspectionAttachment elimina un PDF de una inspección.
func (h *AutoAttachmentHandlers) DeleteInspectionAttachment(w http.ResponseWriter, r *http.Request) {
	h.delete(w, r, models.AutoAttachmentOwnerInspection, "inspection_id")
}

func (h *AutoAttachmentHandlers) upload(w http.ResponseWriter, r *http.Request, ownerType, ownerValue string) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	ownerID, err := parsePathID(r, ownerValue)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID del registro inválido")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, services.MaxAutoAttachmentBytes+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			respondError(w, http.StatusRequestEntityTooLarge, "request_too_large", "El archivo supera el máximo de 10 MB")
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_file", "No se pudo leer el archivo")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_file", "No se encontró el archivo")
		return
	}
	defer file.Close()

	att, err := h.service.Save(r.Context(), autoID, ownerType, ownerID, services.AutoAttachmentUpload{
		FileName: header.Filename,
		Reader:   file,
	})
	if err != nil {
		respondAutoAttachmentError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, att)
}

func (h *AutoAttachmentHandlers) download(w http.ResponseWriter, r *http.Request, ownerType, ownerValue string) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	ownerID, err := parsePathID(r, ownerValue)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID del registro inválido")
		return
	}
	attachmentID, err := parsePathID(r, "attachment_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de adjunto inválido")
		return
	}

	att, path, err := h.service.Get(r.Context(), autoID, ownerType, ownerID, attachmentID)
	if err != nil {
		respondAutoAttachmentError(w, err)
		return
	}

	disposition := "inline"
	if r.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	safeName := strings.ReplaceAll(strings.ReplaceAll(att.FileName, `"`, ""), "\n", "")
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, safeName))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, path)
}

func (h *AutoAttachmentHandlers) delete(w http.ResponseWriter, r *http.Request, ownerType, ownerValue string) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	ownerID, err := parsePathID(r, ownerValue)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID del registro inválido")
		return
	}
	attachmentID, err := parsePathID(r, "attachment_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de adjunto inválido")
		return
	}

	if err := h.service.Delete(r.Context(), autoID, ownerType, ownerID, attachmentID); err != nil {
		respondAutoAttachmentError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Adjunto eliminado"})
}

func respondAutoAttachmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrAttachmentTooLarge):
		respondError(w, http.StatusRequestEntityTooLarge, "request_too_large", err.Error())
	case errors.Is(err, services.ErrAttachmentLimit):
		respondError(w, http.StatusConflict, "attachment_limit", err.Error())
	case errors.Is(err, services.ErrAttachmentNotFound):
		respondError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, services.ErrAttachmentOwnerAbsent):
		respondError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, services.ErrAttachmentInvalid):
		respondError(w, http.StatusBadRequest, "invalid_file", err.Error())
	default:
		respondError(w, http.StatusBadRequest, "invalid_file", err.Error())
	}
}
