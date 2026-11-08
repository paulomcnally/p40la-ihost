package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// AutoMaintenanceHandlers agrupa los handlers de mantenimientos (SPEC-124).
type AutoMaintenanceHandlers struct {
	service *services.AutoMaintenanceService
}

// NewAutoMaintenanceHandlers crea un nuevo AutoMaintenanceHandlers.
func NewAutoMaintenanceHandlers(service *services.AutoMaintenanceService) *AutoMaintenanceHandlers {
	return &AutoMaintenanceHandlers{service: service}
}

// ListMaintenances responde con los mantenimientos de un auto.
func (h *AutoMaintenanceHandlers) ListMaintenances(w http.ResponseWriter, r *http.Request) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	result, err := h.service.List(r.Context(), autoID)
	if err != nil {
		respondAutoRecordError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// GetMaintenance responde con un mantenimiento y sus conceptos/adjuntos.
func (h *AutoMaintenanceHandlers) GetMaintenance(w http.ResponseWriter, r *http.Request) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	maintenanceID, err := parsePathID(r, "maintenance_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de mantenimiento inválido")
		return
	}
	result, err := h.service.Get(r.Context(), autoID, maintenanceID)
	if err != nil {
		respondAutoRecordError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// CreateMaintenanceItem agrega una línea de mano de obra o insumos.
func (h *AutoMaintenanceHandlers) CreateMaintenanceItem(w http.ResponseWriter, r *http.Request) {
	autoID, maintenanceID, ok := parseMaintenancePathIDs(w, r)
	if !ok {
		return
	}
	var input models.AutoMaintenanceItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	result, err := h.service.CreateItem(r.Context(), autoID, maintenanceID, input)
	if err != nil {
		respondAutoRecordError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, result)
}

// UpdateMaintenanceItem actualiza una línea de mano de obra o insumos.
func (h *AutoMaintenanceHandlers) UpdateMaintenanceItem(w http.ResponseWriter, r *http.Request) {
	autoID, maintenanceID, ok := parseMaintenancePathIDs(w, r)
	if !ok {
		return
	}
	itemID, err := parsePathID(r, "item_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de concepto inválido")
		return
	}
	var input models.AutoMaintenanceItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	if err := h.service.UpdateItem(r.Context(), autoID, maintenanceID, itemID, input); err != nil {
		respondAutoRecordError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Concepto actualizado"})
}

// DeleteMaintenanceItem elimina una línea de mano de obra o insumos.
func (h *AutoMaintenanceHandlers) DeleteMaintenanceItem(w http.ResponseWriter, r *http.Request) {
	autoID, maintenanceID, ok := parseMaintenancePathIDs(w, r)
	if !ok {
		return
	}
	itemID, err := parsePathID(r, "item_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de concepto inválido")
		return
	}
	if err := h.service.DeleteItem(r.Context(), autoID, maintenanceID, itemID); err != nil {
		respondAutoRecordError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Concepto eliminado"})
}

// CreateMaintenance registra un mantenimiento.
func (h *AutoMaintenanceHandlers) CreateMaintenance(w http.ResponseWriter, r *http.Request) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var input models.AutoMaintenanceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	result, err := h.service.Create(r.Context(), autoID, input)
	if err != nil {
		respondAutoRecordError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, result)
}

// UpdateMaintenance actualiza un mantenimiento.
func (h *AutoMaintenanceHandlers) UpdateMaintenance(w http.ResponseWriter, r *http.Request) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	maintenanceID, err := parsePathID(r, "maintenance_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de mantenimiento inválido")
		return
	}
	var input models.AutoMaintenanceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	result, err := h.service.Update(r.Context(), autoID, maintenanceID, input)
	if err != nil {
		respondAutoRecordError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// DeleteMaintenance elimina un mantenimiento y sus adjuntos.
func (h *AutoMaintenanceHandlers) DeleteMaintenance(w http.ResponseWriter, r *http.Request) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	maintenanceID, err := parsePathID(r, "maintenance_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de mantenimiento inválido")
		return
	}
	if err := h.service.Delete(r.Context(), autoID, maintenanceID); err != nil {
		respondAutoRecordError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Mantenimiento eliminado"})
}

func parseAutoPathID(r *http.Request) (int64, error) {
	return parsePathID(r, "id")
}

func parseMaintenancePathIDs(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return 0, 0, false
	}
	maintenanceID, err := parsePathID(r, "maintenance_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de mantenimiento inválido")
		return 0, 0, false
	}
	return autoID, maintenanceID, true
}

func parsePathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func respondAutoRecordError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrAutoNotFound):
		respondError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, services.ErrMaintenanceNotFound):
		respondError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, services.ErrInspectionNotFound):
		respondError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, services.ErrMaintenanceItemNotFound):
		respondError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, services.ErrInstitutionCategory):
		respondError(w, http.StatusBadRequest, "institution_category", err.Error())
	case errors.Is(err, services.ErrInstitutionNotFound):
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, services.ErrCurrencyNotFound):
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
	}
}
