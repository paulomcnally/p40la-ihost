package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// AlertSenders agrupa los schedulers capaces de enviar una alerta de forma
// manual (SPEC-090).
type AlertSenders interface {
	SendNow() services.AlertSendResult
}

// BillCreatedSender envía el aviso de prueba de "nueva factura generada"
// (SPEC-090) sin generar facturas.
type BillCreatedSender interface {
	SendNowBillCreated() services.AlertSendResult
}

// AlertsHandlers expone el catálogo de alertas y sus toggles de canal.
type AlertsHandlers struct {
	alerts   *services.AlertService
	settings *services.SystemSettingsService
	senders  []AlertSenders
	billCtr  BillCreatedSender
}

func NewAlertsHandlers(alerts *services.AlertService, settings *services.SystemSettingsService) *AlertsHandlers {
	return &AlertsHandlers{alerts: alerts, settings: settings}
}

// SetSchedulers inyecta los schedulers que el envío manual puede disparar
// (SPEC-090). Se llama desde main.go; en tests se puede omitir.
func (h *AlertsHandlers) SetSchedulers(senders []AlertSenders, billScheduler BillCreatedSender) {
	h.senders = senders
	h.billCtr = billScheduler
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

// SendNow dispara el envío manual de las alertas (SPEC-090). Cada scheduler
// respeta los canales habilitados por su alerta y los gates maestros; no se
// escribe last_* (el automático queda intacto). No falla si un canal está roto.
func (h *AlertsHandlers) SendNow(w http.ResponseWriter, r *http.Request) {
	results := make([]services.AlertSendResult, 0, 4)
	for _, s := range h.senders {
		results = append(results, s.SendNow())
	}
	if h.billCtr != nil {
		results = append(results, h.billCtr.SendNowBillCreated())
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Alertas enviadas",
		"results": results,
	})
}

type updateAlertRequest struct {
	MailEnabled     *bool `json:"mail_enabled,omitempty"`
	VoiceEnabled    *bool `json:"voice_enabled,omitempty"`
	TelegramEnabled *bool `json:"telegram_enabled,omitempty"`
}

// UpdateAlert actualiza mail_enabled / voice_enabled / telegram_enabled de una alerta.
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
	if req.TelegramEnabled != nil && *req.TelegramEnabled {
		if err := h.validateTelegramGating(r.Context()); err != nil {
			respondError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
			return
		}
	}

	if err := h.alerts.SetFlags(r.Context(), key, req.MailEnabled, req.VoiceEnabled, req.TelegramEnabled); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"key":              key,
		"mail_enabled":     derefBool(req.MailEnabled, alert.MailEnabled),
		"voice_enabled":    derefBool(req.VoiceEnabled, alert.VoiceEnabled),
		"telegram_enabled": derefBool(req.TelegramEnabled, alert.TelegramEnabled),
		"message":          "Alerta actualizada",
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

// validateTelegramGating verifica que el bot de Telegram esté habilitado en
// settings, con token y al menos un chat_id autorizado (SPEC-088). Si el bot
// no está habilitado, el canal telegram no se puede activar ni se intenta
// enviar nada.
func (h *AlertsHandlers) validateTelegramGating(ctx context.Context) error {
	cfg, err := h.settings.GetTelegramBotConfig(ctx)
	if err != nil {
		return err
	}
	if !cfg.Enabled {
		return errors.New("Para activar alertas por Telegram, activá primero el interruptor del Bot de Telegram")
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return errors.New("Para activar alertas por Telegram se necesita el token del bot configurado")
	}
	if len(cfg.ChatIDs) == 0 {
		return errors.New("Para activar alertas por Telegram se necesita al menos un chat_id autorizado")
	}
	return nil
}
