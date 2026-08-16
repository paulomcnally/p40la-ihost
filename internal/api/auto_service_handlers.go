package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// AutoServiceHandlers agrupa los handlers de seguros de autos.
type AutoServiceHandlers struct {
	service *services.AutoServiceService
}

// NewAutoServiceHandlers crea un nuevo AutoServiceHandlers.
func NewAutoServiceHandlers(service *services.AutoServiceService) *AutoServiceHandlers {
	return &AutoServiceHandlers{service: service}
}

type autoServiceRequest struct {
	ServiceID    int64  `json:"service_id"`
	CoverageType string `json:"coverage_type"`
}

// ListAutoServices responde con los seguros de un auto.
func (h *AutoServiceHandlers) ListAutoServices(w http.ResponseWriter, r *http.Request) {
	autoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	details, err := h.service.ListByAuto(r.Context(), autoID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, details)
}

// CreateAutoService asocia un servicio como seguro a un auto.
func (h *AutoServiceHandlers) CreateAutoService(w http.ResponseWriter, r *http.Request) {
	autoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req autoServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	as, err := h.service.Create(r.Context(), autoID, req.ServiceID, req.CoverageType)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, as)
}

// DeleteAutoService elimina un seguro de un auto.
func (h *AutoServiceHandlers) DeleteAutoService(w http.ResponseWriter, r *http.Request) {
	autoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	serviceID, err := strconv.ParseInt(r.PathValue("service_id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "service_id inválido")
		return
	}
	if err := h.service.Delete(r.Context(), autoID, serviceID); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Seguro eliminado"})
}

// ListAvailableServices responde con servicios disponibles para asociar.
func (h *AutoServiceHandlers) ListAvailableServices(w http.ResponseWriter, r *http.Request) {
	autoID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	services, err := h.service.ListAvailableServices(r.Context(), autoID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, services)
}
