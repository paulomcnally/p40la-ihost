package api

import (
	"encoding/json"
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// AlertsHandlers expone el catálogo de alertas y sus toggles de canal.
type AlertsHandlers struct {
	alerts *services.AlertService
}

func NewAlertsHandlers(alerts *services.AlertService) *AlertsHandlers {
	return &AlertsHandlers{alerts: alerts}
}

// ListAlerts devuelve todas las alertas con sus flags de canal.
func (h *AlertsHandlers) ListAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.alerts.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, alerts)
}

type updateAlertRequest struct {
	MailEnabled  *bool `json:"mail_enabled,omitempty"`
	VoiceEnabled *bool `json:"voice_enabled,omitempty"`
}

// UpdateAlert actualiza mail_enabled / voice_enabled de una alerta.
func (h *AlertsHandlers) UpdateAlert(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	var req updateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	alert, err := h.alerts.GetByKey(r.Context(), key)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if alert == nil {
		respondError(w, http.StatusNotFound, "not_found", "Alerta no encontrada")
		return
	}

	if err := h.alerts.SetFlags(r.Context(), key, req.MailEnabled, req.VoiceEnabled); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"key":           key,
		"mail_enabled":  derefBool(req.MailEnabled, alert.MailEnabled),
		"voice_enabled": derefBool(req.VoiceEnabled, alert.VoiceEnabled),
		"message":       "Alerta actualizada",
	})
}

func derefBool(v *bool, def bool) bool {
	if v != nil {
		return *v
	}
	return def
}
