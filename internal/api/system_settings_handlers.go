package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type SystemSettingsHandlers struct {
	settings *services.SystemSettingsService
}

func NewSystemSettingsHandlers(settings *services.SystemSettingsService) *SystemSettingsHandlers {
	return &SystemSettingsHandlers{settings: settings}
}

func (h *SystemSettingsHandlers) GetBillingGenerationHour(w http.ResponseWriter, r *http.Request) {
	hour, err := h.settings.GetBillingGenerationHour(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"key":   "billing_generation_hour",
		"value": hour,
	})
}

func (h *SystemSettingsHandlers) SetBillingGenerationHour(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value int `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	if err := h.settings.SetBillingGenerationHour(r.Context(), req.Value); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"key":     "billing_generation_hour",
		"value":   req.Value,
		"message": "Hora de generación actualizada",
	})
}

func (h *SystemSettingsHandlers) GetSetting(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	setting, err := h.settings.GetSetting(r.Context(), key)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if setting == nil {
		respondError(w, http.StatusNotFound, "not_found", "Setting no encontrado")
		return
	}
	respondJSON(w, http.StatusOK, setting)
}

type settingsRequest struct {
	BillingGenerationHour *int `json:"billing_generation_hour,omitempty"`
}

func (h *SystemSettingsHandlers) GetSystemSettings(w http.ResponseWriter, r *http.Request) {
	hour, err := h.settings.GetBillingGenerationHour(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"billing_generation_hour": hour,
	})
}

func (h *SystemSettingsHandlers) UpdateSystemSettings(w http.ResponseWriter, r *http.Request) {
	var req settingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}
	if req.BillingGenerationHour != nil {
		if err := h.settings.SetBillingGenerationHour(r.Context(), *req.BillingGenerationHour); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
	}
	hour, _ := h.settings.GetBillingGenerationHour(r.Context())
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"billing_generation_hour": hour,
		"message":                 "Configuración actualizada",
	})
}

func parseIntParam(r *http.Request, param string) (int, error) {
	raw := r.URL.Query().Get(param)
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}
