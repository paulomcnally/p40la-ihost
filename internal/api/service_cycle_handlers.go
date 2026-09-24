package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// ServiceCycleHandlers agrupa los handlers de ciclos de servicio (SPEC-091).
type ServiceCycleHandlers struct {
	service *services.ServiceCycleService
}

// NewServiceCycleHandlers crea un nuevo ServiceCycleHandlers.
func NewServiceCycleHandlers(service *services.ServiceCycleService) *ServiceCycleHandlers {
	return &ServiceCycleHandlers{service: service}
}

type renewServiceRequest struct {
	StartDate       string   `json:"start_date"`
	EndDate         string   `json:"end_date"`
	SuggestedAmount *float64 `json:"suggested_amount"`
}

// RenewService renueva un servicio de seguro creando un ciclo nuevo (SPEC-091).
func (h *ServiceCycleHandlers) RenewService(w http.ResponseWriter, r *http.Request) {
	serviceID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}

	var req renewServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	svc, cycle, err := h.service.Renew(r.Context(), serviceID, req.StartDate, req.EndDate, req.SuggestedAmount)
	if err != nil {
		if err.Error() == "servicio no encontrado" {
			respondError(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"service": svc,
		"cycle":   cycle,
	})
}

// ListServiceCycles responde con los ciclos de un servicio (SPEC-091).
func (h *ServiceCycleHandlers) ListServiceCycles(w http.ResponseWriter, r *http.Request) {
	serviceID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	cycles, err := h.service.ListByService(r.Context(), serviceID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, cycles)
}
