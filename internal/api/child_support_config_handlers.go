package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type childSupportConfigRequest struct {
	ChildID           int64   `json:"child_id"`
	PensionCategoryID int64   `json:"pension_category_id"`
	Amount            float64 `json:"amount"`
	Currency          string  `json:"currency"`
	IsActive          *bool   `json:"is_active"`
	AutoGenerate      *bool   `json:"auto_generate"`
}

// ChildSupportConfigHandlers agrupa los handlers de configs de pensión.
type ChildSupportConfigHandlers struct {
	service *services.ChildSupportConfigService
}

// NewChildSupportConfigHandlers crea un nuevo ChildSupportConfigHandlers.
func NewChildSupportConfigHandlers(service *services.ChildSupportConfigService) *ChildSupportConfigHandlers {
	return &ChildSupportConfigHandlers{service: service}
}

// ListConfigs responde con todas las configs.
func (h *ChildSupportConfigHandlers) ListConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, configs)
}

// GetConfig responde con una config por ID.
func (h *ChildSupportConfigHandlers) GetConfig(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	config, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if config == nil {
		respondError(w, http.StatusNotFound, "not_found", "Config no encontrada")
		return
	}
	respondJSON(w, http.StatusOK, config)
}

// CreateConfig crea una nueva config.
func (h *ChildSupportConfigHandlers) CreateConfig(w http.ResponseWriter, r *http.Request) {
	var req childSupportConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	auto := false
	if req.AutoGenerate != nil {
		auto = *req.AutoGenerate
	}
	config, err := h.service.Create(r.Context(), req.ChildID, req.PensionCategoryID, req.Amount, req.Currency, active, auto)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, config)
}

// UpdateConfig actualiza una config existente.
func (h *ChildSupportConfigHandlers) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	var req childSupportConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	if req.IsActive == nil || req.AutoGenerate == nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Los campos is_active y auto_generate son requeridos")
		return
	}
	config, err := h.service.Update(r.Context(), id, req.PensionCategoryID, req.Amount, req.Currency, *req.IsActive, *req.AutoGenerate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, config)
}

// DeleteConfig elimina una config.
func (h *ChildSupportConfigHandlers) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_id", "ID inválido")
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Config eliminada"})
}
