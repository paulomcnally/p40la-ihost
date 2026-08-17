package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// AlertsHandlers expone el catálogo de alertas y sus toggles de canal.
type AlertsHandlers struct {
	alerts   *services.AlertService
	settings *services.SystemSettingsService
}

func NewAlertsHandlers(alerts *services.AlertService, settings *services.SystemSettingsService) *AlertsHandlers {
	return &AlertsHandlers{alerts: alerts, settings: settings}
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

	// Gating de canales (SPEC-037): no permitir activar un canal sin los
	// prerequisitos configurados, espejo del frontend.
	if req.MailEnabled != nil && *req.MailEnabled {
		if err := h.validateMailGating(r.Context()); err != nil {
			respondError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
			return
		}
	}
	if req.VoiceEnabled != nil && *req.VoiceEnabled {
		if err := h.validateVoiceGating(r.Context()); err != nil {
			respondError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
			return
		}
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

// validateMailGating verifica que se cumplan los prerequisitos para activar
// el canal mail: toggle maestro on, SMTP configurado y ≥1 destinatario.
func (h *AlertsHandlers) validateMailGating(ctx context.Context) error {
	enabled, err := h.settings.GetEmailAlertsEnabled(ctx)
	if err != nil {
		return err
	}
	if !enabled {
		return errors.New("Para activar alertas por email, activá primero el interruptor de Alertas por Email")
	}

	configured, err := h.settings.IsSMTPConfigured(ctx)
	if err != nil {
		return err
	}
	if !configured {
		return errors.New("Para activar alertas por email se necesita SMTP configurado")
	}

	emails, err := h.settings.GetAlertEmails(ctx)
	if err != nil {
		return err
	}
	if len(emails) == 0 {
		return errors.New("Para activar alertas por email se necesita al menos un destinatario")
	}
	return nil
}

// validateVoiceGating verifica que Voice Monkey esté activo (enabled ∧
// configured ∧ send_alerts) para permitir activar el canal voz.
func (h *AlertsHandlers) validateVoiceGating(ctx context.Context) error {
	vm, err := h.settings.GetVoiceMonkeyConfigPublic(ctx)
	if err != nil {
		return err
	}
	if !vm.Enabled || !vm.Configured || !vm.SendAlerts {
		return errors.New("Para activar alertas por voz se necesita Voice Monkey activo, configurado y enviando alertas")
	}
	return nil
}
