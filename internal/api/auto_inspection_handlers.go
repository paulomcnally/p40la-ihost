package api

import (
	"encoding/json"
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// AutoInspectionHandlers agrupa los handlers de inspecciones mecánicas
// (SPEC-124).
type AutoInspectionHandlers struct {
	service *services.AutoInspectionService
}

// NewAutoInspectionHandlers crea un nuevo AutoInspectionHandlers.
func NewAutoInspectionHandlers(service *services.AutoInspectionService) *AutoInspectionHandlers {
	return &AutoInspectionHandlers{service: service}
}

// ListInspections responde con las inspecciones y el cronograma de un auto.
func (h *AutoInspectionHandlers) ListInspections(w http.ResponseWriter, r *http.Request) {
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

// GetInspection responde con una inspección y sus adjuntos.
func (h *AutoInspectionHandlers) GetInspection(w http.ResponseWriter, r *http.Request) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	inspectionID, err := parsePathID(r, "inspection_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de inspección inválido")
		return
	}
	result, err := h.service.Get(r.Context(), autoID, inspectionID)
	if err != nil {
		respondAutoRecordError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// CreateInspection registra una inspección mecánica.
func (h *AutoInspectionHandlers) CreateInspection(w http.ResponseWriter, r *http.Request) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var input models.AutoInspectionInput
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

// UpdateInspection actualiza una inspección mecánica.
func (h *AutoInspectionHandlers) UpdateInspection(w http.ResponseWriter, r *http.Request) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	inspectionID, err := parsePathID(r, "inspection_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de inspección inválido")
		return
	}
	var input models.AutoInspectionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	result, err := h.service.Update(r.Context(), autoID, inspectionID, input)
	if err != nil {
		respondAutoRecordError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

// DeleteInspection elimina una inspección y sus adjuntos.
func (h *AutoInspectionHandlers) DeleteInspection(w http.ResponseWriter, r *http.Request) {
	autoID, err := parseAutoPathID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	inspectionID, err := parsePathID(r, "inspection_id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID de inspección inválido")
		return
	}
	if err := h.service.Delete(r.Context(), autoID, inspectionID); err != nil {
		respondAutoRecordError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Inspección eliminada"})
}

// ListInspectionAlerts responde con el estado de inspección de todos los autos.
func (h *AutoInspectionHandlers) ListInspectionAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.service.Alerts(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, alerts)
}
