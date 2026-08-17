package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type SystemSettingsHandlers struct {
	settings     *services.SystemSettingsService
	emailService *services.EmailService
	voiceMonkey  *services.VoiceMonkeyService
}

func NewSystemSettingsHandlers(settings *services.SystemSettingsService, emailService *services.EmailService, voiceMonkey *services.VoiceMonkeyService) *SystemSettingsHandlers {
	return &SystemSettingsHandlers{settings: settings, emailService: emailService, voiceMonkey: voiceMonkey}
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
	BillingGenerationHour *int    `json:"billing_generation_hour,omitempty"`
	AlertCheckHour        *int    `json:"alert_check_hour,omitempty"`
	SMTPHost              *string `json:"smtp_host,omitempty"`
	SMTPPort              *int    `json:"smtp_port,omitempty"`
	SMTPUser              *string `json:"smtp_user,omitempty"`
	SMTPPassword          *string `json:"smtp_password,omitempty"`
	SMTPFromEmail         *string `json:"smtp_from_email,omitempty"`
	SMTPFromName          *string `json:"smtp_from_name,omitempty"`
	AlertEmails           *string `json:"alert_emails,omitempty"`

	// Voice Monkey (SPEC-033). Token/device solo se envían para guardar.
	VoiceMonkeyEnabled    *bool   `json:"voicemonkey_enabled,omitempty"`
	VoiceMonkeySendAlerts *bool   `json:"voicemonkey_send_alerts,omitempty"`
	VoiceMonkeyToken      *string `json:"voicemonkey_token,omitempty"`
	VoiceMonkeyDevice     *string `json:"voicemonkey_device,omitempty"`

	// Email alerts master toggle (SPEC-037).
	EmailAlertsEnabled *bool `json:"email_alerts_enabled,omitempty"`
}

func (h *SystemSettingsHandlers) GetSystemSettings(w http.ResponseWriter, r *http.Request) {
	hour, err := h.settings.GetBillingGenerationHour(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	alertCheckHour, err := h.settings.GetAlertCheckHour(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	smtp, err := h.settings.GetSMTPConfigPublic(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	alertEmails, err := h.settings.GetAlertEmails(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	vm, err := h.settings.GetVoiceMonkeyConfigPublic(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	emailAlertsEnabled, err := h.settings.GetEmailAlertsEnabled(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// SMTPConfigPublic.User es siempre "" (info sensible, no se expone).
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"billing_generation_hour": hour,
		"alert_check_hour":        alertCheckHour,
		"smtp_host":               smtp.Host,
		"smtp_port":               smtp.Port,
		"smtp_user":               smtp.User,
		"smtp_from_email":         smtp.FromEmail,
		"smtp_from_name":          smtp.FromName,
		"smtp_configured":         smtp.Configured,
		"alert_emails":            joinEmails(alertEmails),
		"voicemonkey_enabled":     vm.Enabled,
		"voicemonkey_send_alerts": vm.SendAlerts,
		"voicemonkey_configured":  vm.Configured,
		"email_alerts_enabled":    emailAlertsEnabled,
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

	if req.AlertCheckHour != nil {
		if err := h.settings.SetAlertCheckHour(r.Context(), *req.AlertCheckHour); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
	}

	// Actualiza SMTP solo con los campos enviados. user/password vacíos se mantienen.
	cfg := &models.SMTPConfig{}
	if req.SMTPHost != nil {
		cfg.Host = *req.SMTPHost
	}
	if req.SMTPPort != nil {
		cfg.Port = *req.SMTPPort
	}
	if req.SMTPUser != nil {
		cfg.User = *req.SMTPUser
	}
	if req.SMTPPassword != nil {
		cfg.Password = *req.SMTPPassword
	}
	if req.SMTPFromEmail != nil {
		cfg.FromEmail = *req.SMTPFromEmail
	}
	if req.SMTPFromName != nil {
		cfg.FromName = *req.SMTPFromName
	}
	if err := h.settings.SetSMTPConfig(r.Context(), cfg); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	if req.AlertEmails != nil {
		emails := splitEmails(*req.AlertEmails)
		if err := h.settings.SetAlertEmails(r.Context(), emails); err != nil {
			respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
	}

	// Voice Monkey: cada campo se persiste solo si viene en el request
	// (updates parciales no se pisan entre sí — REQ-021).
	if req.VoiceMonkeyEnabled != nil {
		if err := h.settings.SetVoiceMonkeyEnabled(r.Context(), *req.VoiceMonkeyEnabled); err != nil {
			respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
	}
	if req.VoiceMonkeySendAlerts != nil {
		if err := h.settings.SetVoiceMonkeySendAlerts(r.Context(), *req.VoiceMonkeySendAlerts); err != nil {
			respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
	}
	if req.VoiceMonkeyToken != nil || req.VoiceMonkeyDevice != nil {
		vmCfg := &models.VoiceMonkeyConfig{}
		if req.VoiceMonkeyToken != nil {
			vmCfg.Token = *req.VoiceMonkeyToken
		}
		if req.VoiceMonkeyDevice != nil {
			vmCfg.Device = *req.VoiceMonkeyDevice
		}
		if err := h.settings.SetVoiceMonkeyConfig(r.Context(), vmCfg); err != nil {
			respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
	}

	// Email alerts master toggle (SPEC-037).
	if req.EmailAlertsEnabled != nil {
		if err := h.settings.SetEmailAlertsEnabled(r.Context(), *req.EmailAlertsEnabled); err != nil {
			respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
	}

	hour, _ := h.settings.GetBillingGenerationHour(r.Context())
	smtp, _ := h.settings.GetSMTPConfigPublic(r.Context())
	vm, _ := h.settings.GetVoiceMonkeyConfigPublic(r.Context())
	emailAlertsEnabled, _ := h.settings.GetEmailAlertsEnabled(r.Context())
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"billing_generation_hour": hour,
		"smtp_configured":         smtp.Configured,
		"voicemonkey_configured":  vm.Configured,
		"email_alerts_enabled":    emailAlertsEnabled,
		"message":                 "Configuración actualizada",
	})
}

// TestEmail envía un email de prueba a los destinatarios configurados.
func (h *SystemSettingsHandlers) TestEmail(w http.ResponseWriter, r *http.Request) {
	emails, err := h.settings.GetAlertEmails(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if len(emails) == 0 {
		respondError(w, http.StatusBadRequest, "no_recipients", "Configure destinatarios antes de probar")
		return
	}

	configured, err := h.settings.IsSMTPConfigured(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if !configured {
		respondError(w, http.StatusBadRequest, "smtp_not_configured", "Configure SMTP antes de probar")
		return
	}

	if err := h.emailService.SendTest(r.Context(), emails); err != nil {
		respondError(w, http.StatusInternalServerError, "smtp_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Email de prueba enviado",
		"recipients": joinEmails(emails),
	})
}

// TestVoice anuncia un mensaje de prueba por Voice Monkey (TTS).
func (h *SystemSettingsHandlers) TestVoice(w http.ResponseWriter, r *http.Request) {
	configured, err := h.settings.IsVoiceMonkeyConfigured(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if !configured {
		respondError(w, http.StatusBadRequest, "voicemonkey_not_configured", "Configure Voice Monkey (token y device) antes de probar")
		return
	}

	if err := h.voiceMonkey.SendTest(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, "voicemonkey_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Aviso de voz enviado",
	})
}

// DeleteVoiceMonkey limpia la configuración de Voice Monkey y resetea los
// toggles a OFF (botón "Reconfigurar", REQ-020).
func (h *SystemSettingsHandlers) DeleteVoiceMonkey(w http.ResponseWriter, r *http.Request) {
	if err := h.settings.ClearVoiceMonkey(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	vm, err := h.settings.GetVoiceMonkeyConfigPublic(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"voicemonkey_enabled":     vm.Enabled,
		"voicemonkey_send_alerts": vm.SendAlerts,
		"voicemonkey_configured":  vm.Configured,
		"message":                 "Configuración de Voice Monkey eliminada",
	})
}

// DeleteSMTP limpia la configuración SMTP (botón "Reconfigurar", SPEC-034).
func (h *SystemSettingsHandlers) DeleteSMTP(w http.ResponseWriter, r *http.Request) {
	if err := h.settings.ClearSMTP(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	smtp, err := h.settings.GetSMTPConfigPublic(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"smtp_configured": smtp.Configured,
		"message":         "Configuración SMTP eliminada",
	})
}

func joinEmails(emails []string) string {
	if len(emails) == 0 {
		return ""
	}
	return strings.Join(emails, ",")
}

func splitEmails(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func parseIntParam(r *http.Request, param string) (int, error) {
	raw := r.URL.Query().Get(param)
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}
