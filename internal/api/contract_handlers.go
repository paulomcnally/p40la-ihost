package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// uploadContract parsea el multipart, aplica el límite y delega el guardado.
func uploadContract(
	w http.ResponseWriter,
	r *http.Request,
	save func(ctx context.Context, id int64, fileName string, reader io.Reader) error,
) {
	id, ok := parseContractID(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, services.MaxContractSize+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			respondError(w, http.StatusRequestEntityTooLarge, "request_too_large", "El archivo supera el máximo de 10 MB")
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_file", "No se pudo leer el archivo")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_file", "No se encontró el archivo")
		return
	}
	defer file.Close()

	if err := save(r.Context(), id, header.Filename, file); err != nil {
		respondContractError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"has_contract": true})
}

// downloadContract sirve el PDF inline o como descarga.
func downloadContract(
	w http.ResponseWriter,
	r *http.Request,
	path func(ctx context.Context, id int64) (string, error),
) {
	id, ok := parseContractID(w, r)
	if !ok {
		return
	}
	filePath, err := path(r.Context(), id)
	if err != nil {
		respondContractError(w, err)
		return
	}

	disposition := "inline"
	if r.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", disposition+`; filename="contract.pdf"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeFile(w, r, filePath)
}

// deleteContract elimina el contrato de una entidad.
func deleteContract(
	w http.ResponseWriter,
	r *http.Request,
	remove func(ctx context.Context, id int64) error,
) {
	id, ok := parseContractID(w, r)
	if !ok {
		return
	}
	if err := remove(r.Context(), id); err != nil {
		respondContractError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseContractID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return 0, false
	}
	return id, true
}

func respondContractError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrContractNotFound), errors.Is(err, services.ErrContractEntityNotFound):
		respondError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, services.ErrContractInvalid):
		respondError(w, http.StatusBadRequest, "invalid_file", err.Error())
	case errors.Is(err, services.ErrContractTooLarge):
		respondError(w, http.StatusRequestEntityTooLarge, "request_too_large", err.Error())
	default:
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
	}
}

// UploadContract sube o reemplaza el contrato PDF de una deuda.
func (h *DebtHandlers) UploadContract(w http.ResponseWriter, r *http.Request) {
	uploadContract(w, r, h.service.SaveContract)
}

// DownloadContract sirve el contrato PDF de una deuda.
func (h *DebtHandlers) DownloadContract(w http.ResponseWriter, r *http.Request) {
	downloadContract(w, r, h.service.ContractPath)
}

// DeleteContract elimina el contrato PDF de una deuda.
func (h *DebtHandlers) DeleteContract(w http.ResponseWriter, r *http.Request) {
	deleteContract(w, r, h.service.DeleteContract)
}

// UploadContract sube o reemplaza el contrato PDF de un servicio.
func (h *ServiceHandlers) UploadContract(w http.ResponseWriter, r *http.Request) {
	uploadContract(w, r, h.service.SaveContract)
}

// DownloadContract sirve el contrato PDF de un servicio.
func (h *ServiceHandlers) DownloadContract(w http.ResponseWriter, r *http.Request) {
	downloadContract(w, r, h.service.ContractPath)
}

// DeleteContract elimina el contrato PDF de un servicio.
func (h *ServiceHandlers) DeleteContract(w http.ResponseWriter, r *http.Request) {
	deleteContract(w, r, h.service.DeleteContract)
}
