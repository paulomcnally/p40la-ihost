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
	telegramBot  *services.TelegramBotService
}

func NewSystemSettingsHandlers(settings *services.SystemSettingsService, emailService *services.EmailService, voiceMonkey *services.VoiceMonkeyService) *SystemSettingsHandlers {
	return &SystemSettingsHandlers{settings: settings, emailService: emailService, voiceMonkey: voiceMonkey}
}

// SetTelegramBotService inyecta el bot de Telegram para notificarlo cuando la
// config cambia (SPEC-079, REQ-006). Se setea después de construir el handler
// porque el bot se crea en main.go con los mismos storages.
func (h *SystemSettingsHandlers) SetTelegramBotService(telegramBot *services.TelegramBotService) {
	h.telegramBot = telegramBot
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
	Timezone              *string `json:"timezone,omitempty"`
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

	// Webhooks master toggle (SPEC-069).
	WebhookEnabled *bool `json:"webhook_enabled,omitempty"`

	// Base URL de los webhooks (SPEC-069). Default: http://ihost.local:8088.
	WebhookBaseURL *string `json:"webhook_base_url,omitempty"`

	// Formato de moneda (SPEC-058).
	CurrencyThousandsSeparator *string `json:"currency_thousands_separator,omitempty"`
	CurrencyDecimalSeparator   *string `json:"currency_decimal_separator,omitempty"`
	CurrencyDecimalDigits      *int    `json:"currency_decimal_digits,omitempty"`

	// Paleta de colores de emails (SPEC-077). Hex estricto (#RRGGBB).
	EmailColorPrimary    *string `json:"email_color_primary,omitempty"`
	EmailColorBackground *string `json:"email_color_background,omitempty"`
	EmailColorCard       *string `json:"email_color_card,omitempty"`
	EmailColorText       *string `json:"email_color_text,omitempty"`
	EmailColorMuted      *string `json:"email_color_muted,omitempty"`
	EmailColorBorder     *string `json:"email_color_border,omitempty"`

	// Bot de Telegram (SPEC-079). Token solo se envía para guardar.
	TelegramBotEnabled         *bool   `json:"telegram_bot_enabled,omitempty"`
	TelegramBotToken           *string `json:"telegram_bot_token,omitempty"`
	TelegramBotChatIDs         *string `json:"telegram_bot_chat_ids,omitempty"`
	TelegramBotSeparatorLength *int    `json:"telegram_bot_separator_length,omitempty"`
	TelegramBotShowMonths      *int    `json:"telegram_bot_show_months,omitempty"`
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

	timezone, err := h.settings.GetTimezone(r.Context())
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

	webhookEnabled, err := h.settings.GetWebhookEnabled(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	webhookBaseURL, err := h.settings.GetWebhookBaseURL(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	currencyFormat, err := h.settings.GetCurrencyFormat(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	palette, err := h.settings.GetEmailPalette(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	telegramBot, err := h.settings.GetTelegramBotConfigPublic(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// SMTPConfigPublic.User es siempre "" (info sensible, no se expone).
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"billing_generation_hour":      hour,
		"alert_check_hour":             alertCheckHour,
		"timezone":                     timezone,
		"smtp_host":                    smtp.Host,
		"smtp_port":                    smtp.Port,
		"smtp_user":                    smtp.User,
		"smtp_from_email":              smtp.FromEmail,
		"smtp_from_name":               smtp.FromName,
		"smtp_configured":              smtp.Configured,
		"alert_emails":                 joinEmails(alertEmails),
		"voicemonkey_enabled":          vm.Enabled,
		"voicemonkey_send_alerts":      vm.SendAlerts,
		"voicemonkey_configured":       vm.Configured,
		"email_alerts_enabled":         emailAlertsEnabled,
		"webhook_enabled":              webhookEnabled,
		"webhook_base_url":             webhookBaseURL,
		"currency_thousands_separator": currencyFormat.ThousandsSeparator,
		"currency_decimal_separator":   currencyFormat.DecimalSeparator,
		"currency_decimal_digits":      currencyFormat.DecimalDigits,
		"email_color_primary":          palette.Primary,
		"email_color_background":       palette.Background,
		"email_color_card":             palette.Card,
		"email_color_text":             palette.Text,
		"email_color_muted":            palette.Muted,
		"email_color_border":           palette.Border,
		"telegram_bot_enabled":          telegramBot.Enabled,
		"telegram_bot_configured":       telegramBot.Configured,
		"telegram_bot_separator_length": telegramBot.SeparatorLength,
		"telegram_bot_show_months":      telegramBot.ShowMonths,
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

	// Zona horaria (SPEC-078). Se valida en el service (nombre IANA); un valor
	// inválido se rechaza con 400 sin persistir.
	if req.Timezone != nil {
		if err := h.settings.SetTimezone(r.Context(), *req.Timezone); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_timezone", err.Error())
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

	// Webhooks master toggle (SPEC-069).
	if req.WebhookEnabled != nil {
		if err := h.settings.SetWebhookEnabled(r.Context(), *req.WebhookEnabled); err != nil {
			respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
	}

	// Base URL de los webhooks (SPEC-069).
	if req.WebhookBaseURL != nil {
		if err := h.settings.SetWebhookBaseURL(r.Context(), *req.WebhookBaseURL); err != nil {
			respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
	}

	// Formato de moneda (SPEC-058). Se valida en el service (whitelist).
	if req.CurrencyThousandsSeparator != nil {
		if err := h.settings.SetCurrencyThousandsSeparator(r.Context(), *req.CurrencyThousandsSeparator); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_currency_format", err.Error())
			return
		}
	}
	if req.CurrencyDecimalSeparator != nil {
		if err := h.settings.SetCurrencyDecimalSeparator(r.Context(), *req.CurrencyDecimalSeparator); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_currency_format", err.Error())
			return
		}
	}
	if req.CurrencyDecimalDigits != nil {
		if err := h.settings.SetCurrencyDecimalDigits(r.Context(), *req.CurrencyDecimalDigits); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_currency_format", err.Error())
			return
		}
	}

	// Paleta de colores de emails (SPEC-077). Si algún hex enviado es
	// inválido, se rechaza todo con 400 sin persistir nada.
	if req.EmailColorPrimary != nil || req.EmailColorBackground != nil ||
		req.EmailColorCard != nil || req.EmailColorText != nil ||
		req.EmailColorMuted != nil || req.EmailColorBorder != nil {
		palette := services.EmailPalette{}
		if req.EmailColorPrimary != nil {
			palette.Primary = *req.EmailColorPrimary
		}
		if req.EmailColorBackground != nil {
			palette.Background = *req.EmailColorBackground
		}
		if req.EmailColorCard != nil {
			palette.Card = *req.EmailColorCard
		}
		if req.EmailColorText != nil {
			palette.Text = *req.EmailColorText
		}
		if req.EmailColorMuted != nil {
			palette.Muted = *req.EmailColorMuted
		}
		if req.EmailColorBorder != nil {
			palette.Border = *req.EmailColorBorder
		}
		if err := h.settings.SetEmailPalette(r.Context(), palette); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_color", err.Error())
			return
		}
	}

	// Bot de Telegram (SPEC-079): cada campo se persiste solo si viene en el
	// request (updates parciales no se pisan). Tras cualquier cambio se
	// notifica al servicio para reconfigurar el polling en runtime (REQ-006).
	telegramBotChanged := false
	if req.TelegramBotEnabled != nil {
		if err := h.settings.SetTelegramBotEnabled(r.Context(), *req.TelegramBotEnabled); err != nil {
			respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		telegramBotChanged = true
	}
	if req.TelegramBotToken != nil {
		if err := h.settings.SetTelegramBotToken(r.Context(), *req.TelegramBotToken); err != nil {
			respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		telegramBotChanged = true
	}
	if req.TelegramBotChatIDs != nil {
		if err := h.settings.SetTelegramBotChatIDs(r.Context(), splitEmails(*req.TelegramBotChatIDs)); err != nil {
			respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		telegramBotChanged = true
	}
	if req.TelegramBotSeparatorLength != nil {
		if err := h.settings.SetTelegramBotSeparatorLength(r.Context(), *req.TelegramBotSeparatorLength); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		telegramBotChanged = true
	}
	if req.TelegramBotShowMonths != nil {
		if err := h.settings.SetTelegramBotShowMonths(r.Context(), *req.TelegramBotShowMonths); err != nil {
			respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		telegramBotChanged = true
	}
	if telegramBotChanged && h.telegramBot != nil {
		h.telegramBot.NotifyConfigChanged()
	}

	hour, _ := h.settings.GetBillingGenerationHour(r.Context())
	timezone, _ := h.settings.GetTimezone(r.Context())
	smtp, _ := h.settings.GetSMTPConfigPublic(r.Context())
	vm, _ := h.settings.GetVoiceMonkeyConfigPublic(r.Context())
	emailAlertsEnabled, _ := h.settings.GetEmailAlertsEnabled(r.Context())
	webhookEnabled, _ := h.settings.GetWebhookEnabled(r.Context())
	webhookBaseURL, _ := h.settings.GetWebhookBaseURL(r.Context())
	currencyFormat, _ := h.settings.GetCurrencyFormat(r.Context())
	telegramBot, _ := h.settings.GetTelegramBotConfigPublic(r.Context())
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"billing_generation_hour":      hour,
		"timezone":                     timezone,
		"smtp_configured":              smtp.Configured,
		"voicemonkey_configured":       vm.Configured,
		"email_alerts_enabled":         emailAlertsEnabled,
		"webhook_enabled":              webhookEnabled,
		"webhook_base_url":             webhookBaseURL,
		"currency_thousands_separator": currencyFormat.ThousandsSeparator,
		"currency_decimal_separator":   currencyFormat.DecimalSeparator,
		"currency_decimal_digits":      currencyFormat.DecimalDigits,
		"telegram_bot_enabled":          telegramBot.Enabled,
		"telegram_bot_configured":       telegramBot.Configured,
		"telegram_bot_separator_length": telegramBot.SeparatorLength,
		"telegram_bot_show_months":      telegramBot.ShowMonths,
		"message":                       "Configuración actualizada",
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

// ResetEmailPalette limpia los 6 colores de emails y vuelve a los defaults
// (botón "Restablecer a valores por defecto", REQ-013).
func (h *SystemSettingsHandlers) ResetEmailPalette(w http.ResponseWriter, r *http.Request) {
	if err := h.settings.ResetEmailPalette(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	palette, err := h.settings.GetEmailPalette(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"email_color_primary":    palette.Primary,
		"email_color_background": palette.Background,
		"email_color_card":       palette.Card,
		"email_color_text":       palette.Text,
		"email_color_muted":      palette.Muted,
		"email_color_border":     palette.Border,
		"message":                "Paleta de emails restablecida a los valores por defecto",
	})
}

// DeleteTelegramBot limpia la configuración del bot de Telegram y resetea el
// toggle a OFF (botón "Reconfigurar", SPEC-079). Detiene el polling.
func (h *SystemSettingsHandlers) DeleteTelegramBot(w http.ResponseWriter, r *http.Request) {
	if err := h.settings.ClearTelegramBot(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if h.telegramBot != nil {
		h.telegramBot.NotifyConfigChanged()
	}

	tgBot, err := h.settings.GetTelegramBotConfigPublic(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"telegram_bot_enabled":    tgBot.Enabled,
		"telegram_bot_configured": tgBot.Configured,
		"message":                 "Configuración del bot de Telegram eliminada",
	})
}

// previewEmailRequest es la paleta candidata del preview. Todos los campos
// son opcionales; los ausentes se completan con los defaults (REQ-014).
type previewEmailRequest struct {
	EmailColorPrimary    *string `json:"email_color_primary,omitempty"`
	EmailColorBackground *string `json:"email_color_background,omitempty"`
	EmailColorCard       *string `json:"email_color_card,omitempty"`
	EmailColorText       *string `json:"email_color_text,omitempty"`
	EmailColorMuted      *string `json:"email_color_muted,omitempty"`
	EmailColorBorder     *string `json:"email_color_border,omitempty"`
}

// PreviewEmail renderiza un email de ejemplo con una paleta candidata SIN
// persistirla (REQ-014). Devuelve el HTML completo listo para un <iframe>.
func (h *SystemSettingsHandlers) PreviewEmail(w http.ResponseWriter, r *http.Request) {
	var req previewEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_request", "Cuerpo JSON inválido")
		return
	}

	palette := services.DefaultEmailPalette()
	apply := func(field **string, target *string) {
		if *field != nil && services.ValidHexColor(**field) {
			*target = **field
		}
	}
	apply(&req.EmailColorPrimary, &palette.Primary)
	apply(&req.EmailColorBackground, &palette.Background)
	apply(&req.EmailColorCard, &palette.Card)
	apply(&req.EmailColorText, &palette.Text)
	apply(&req.EmailColorMuted, &palette.Muted)
	apply(&req.EmailColorBorder, &palette.Border)

	html := h.emailService.RenderPreviewHTML(r.Context(), palette)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"html": html,
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
