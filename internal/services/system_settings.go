package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type SystemSettingsService struct {
	storage *storage.SystemSettingsStorage
}

func NewSystemSettingsService(storage *storage.SystemSettingsStorage) *SystemSettingsService {
	return &SystemSettingsService{storage: storage}
}

// ---- Billing generation hour ----

func (s *SystemSettingsService) GetBillingGenerationHour(ctx context.Context) (int, error) {
	val, err := s.storage.Get(ctx, "billing_generation_hour")
	if err != nil {
		return 0, err
	}
	if val == "" {
		return 0, nil
	}
	hour, err := strconv.Atoi(val)
	if err != nil {
		return 0, nil
	}
	return hour, nil
}

func (s *SystemSettingsService) SetBillingGenerationHour(ctx context.Context, hour int) error {
	if hour < 0 || hour > 23 {
		return nil
	}
	return s.storage.Set(ctx, "billing_generation_hour", strconv.Itoa(hour))
}

func (s *SystemSettingsService) Set(ctx context.Context, key, value string) error {
	return s.storage.Set(ctx, key, value)
}

func (s *SystemSettingsService) GetSetting(ctx context.Context, key string) (*models.SystemSetting, error) {
	return s.storage.GetSetting(ctx, key)
}

// ---- Formato de moneda (SPEC-058) ----

// Claves de formato de moneda en system_settings.
const (
	CurrencyThousandsSeparatorKey = "currency_thousands_separator"
	CurrencyDecimalSeparatorKey   = "currency_decimal_separator"
	CurrencyDecimalDigitsKey      = "currency_decimal_digits"
)

// GetCurrencyFormat devuelve el formato de moneda configurado. Si falta una
// clave o tiene un valor inválido, se usa el default Nicaragua (1,000.00).
func (s *SystemSettingsService) GetCurrencyFormat(ctx context.Context) (CurrencyFormat, error) {
	f := DefaultCurrencyFormat()

	thousands, err := s.storage.Get(ctx, CurrencyThousandsSeparatorKey)
	if err != nil {
		return f, err
	}
	if validSeparator(thousands) {
		f.ThousandsSeparator = thousands
	}

	decimalSep, err := s.storage.Get(ctx, CurrencyDecimalSeparatorKey)
	if err != nil {
		return f, err
	}
	if validSeparator(decimalSep) {
		f.DecimalSeparator = decimalSep
	}

	digits, err := s.storage.Get(ctx, CurrencyDecimalDigitsKey)
	if err != nil {
		return f, err
	}
	if d, err := strconv.Atoi(digits); err == nil && d >= 0 && d <= 4 {
		f.DecimalDigits = d
	}

	return f, nil
}

// SetCurrencyThousandsSeparator persiste el separador de miles (whitelist).
func (s *SystemSettingsService) SetCurrencyThousandsSeparator(ctx context.Context, sep string) error {
	if !validSeparator(sep) {
		return fmt.Errorf("separador de miles inválido")
	}
	return s.storage.Set(ctx, CurrencyThousandsSeparatorKey, sep)
}

// SetCurrencyDecimalSeparator persiste el separador decimal (whitelist).
func (s *SystemSettingsService) SetCurrencyDecimalSeparator(ctx context.Context, sep string) error {
	if !validSeparator(sep) {
		return fmt.Errorf("separador decimal inválido")
	}
	return s.storage.Set(ctx, CurrencyDecimalSeparatorKey, sep)
}

// SetCurrencyDecimalDigits persiste la cantidad de dígitos decimales (0-4).
func (s *SystemSettingsService) SetCurrencyDecimalDigits(ctx context.Context, digits int) error {
	if digits < 0 || digits > 4 {
		return fmt.Errorf("dígitos decimales inválidos")
	}
	return s.storage.Set(ctx, CurrencyDecimalDigitsKey, strconv.Itoa(digits))
}

// ---- SMTP configuration ----

// SMTP default port cuando no está configurado.
const defaultSMTPPort = 587

// GetSMTPConfig devuelve la configuración SMTP completa (incluye credenciales).
// Uso interno: EmailService. NUNCA exponer esta config en responses de API.
func (s *SystemSettingsService) GetSMTPConfig(ctx context.Context) (*models.SMTPConfig, error) {
	host, err := s.storage.Get(ctx, "smtp_host")
	if err != nil {
		return nil, err
	}
	portStr, err := s.storage.Get(ctx, "smtp_port")
	if err != nil {
		return nil, err
	}
	user, err := s.storage.Get(ctx, "smtp_user")
	if err != nil {
		return nil, err
	}
	password, err := s.storage.Get(ctx, "smtp_password")
	if err != nil {
		return nil, err
	}
	fromEmail, err := s.storage.Get(ctx, "smtp_from_email")
	if err != nil {
		return nil, err
	}
	fromName, err := s.storage.Get(ctx, "smtp_from_name")
	if err != nil {
		return nil, err
	}

	port := defaultSMTPPort
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
			port = p
		}
	}

	return &models.SMTPConfig{
		Host:      host,
		Port:      port,
		User:      user,
		Password:  password,
		FromEmail: fromEmail,
		FromName:  fromName,
	}, nil
}

// GetSMTPConfigPublic devuelve la config SMTP SIN credenciales (user/password).
// Se usa en responses de API. Incluye flag Configured.
func (s *SystemSettingsService) GetSMTPConfigPublic(ctx context.Context) (*models.SMTPConfigPublic, error) {
	cfg, err := s.GetSMTPConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &models.SMTPConfigPublic{
		Host:       cfg.Host,
		Port:       cfg.Port,
		User:       "",
		FromEmail:  cfg.FromEmail,
		FromName:   cfg.FromName,
		Configured: s.isConfigured(cfg),
	}, nil
}

// SetSMTPConfig guarda la config SMTP. Los campos vacíos no se sobrescriben
// si ya tienen un valor guardado (permite ediciones parciales del host,
// manteniendo user/password intactos cuando no se envían).
func (s *SystemSettingsService) SetSMTPConfig(ctx context.Context, cfg *models.SMTPConfig) error {
	if err := s.setIfNonEmpty(ctx, "smtp_host", cfg.Host); err != nil {
		return err
	}
	if cfg.Port > 0 {
		if err := s.storage.Set(ctx, "smtp_port", strconv.Itoa(cfg.Port)); err != nil {
			return err
		}
	}
	if err := s.setIfNonEmpty(ctx, "smtp_user", cfg.User); err != nil {
		return err
	}
	if err := s.setIfNonEmpty(ctx, "smtp_password", cfg.Password); err != nil {
		return err
	}
	if err := s.setIfNonEmpty(ctx, "smtp_from_email", cfg.FromEmail); err != nil {
		return err
	}
	if err := s.setIfNonEmpty(ctx, "smtp_from_name", cfg.FromName); err != nil {
		return err
	}
	return nil
}

// IsSMTPConfigured indica si hay user y password guardados.
func (s *SystemSettingsService) IsSMTPConfigured(ctx context.Context) (bool, error) {
	cfg, err := s.GetSMTPConfig(ctx)
	if err != nil {
		return false, err
	}
	return s.isConfigured(cfg), nil
}

func (s *SystemSettingsService) isConfigured(cfg *models.SMTPConfig) bool {
	return cfg.User != "" && cfg.Password != "" && cfg.Host != ""
}

// ClearSMTP limpia toda la configuración SMTP (host, port, user, password,
// from_email, from_name) — botón "Reconfigurar" (SPEC-034).
func (s *SystemSettingsService) ClearSMTP(ctx context.Context) error {
	for _, key := range []string{
		"smtp_host", "smtp_port", "smtp_user",
		"smtp_password", "smtp_from_email", "smtp_from_name",
	} {
		if err := s.storage.Set(ctx, key, ""); err != nil {
			return err
		}
	}
	return nil
}

// setIfNonEmpty guarda el valor solo si no está vacío. Usado para no
// sobrescribir credenciales existentes con strings vacíos en updates parciales.
func (s *SystemSettingsService) setIfNonEmpty(ctx context.Context, key, value string) error {
	if value == "" {
		return nil
	}
	return s.storage.Set(ctx, key, value)
}

// ---- Alert emails ----

// GetAlertEmails devuelve la lista de emails destinatarios de alertas.
func (s *SystemSettingsService) GetAlertEmails(ctx context.Context) ([]string, error) {
	val, err := s.storage.Get(ctx, "alert_emails")
	if err != nil {
		return nil, err
	}
	return parseEmails(val), nil
}

// SetAlertEmails guarda la lista de emails destinatarios (comma-separated).
func (s *SystemSettingsService) SetAlertEmails(ctx context.Context, emails []string) error {
	value := strings.Join(sanitizeEmails(emails), ",")
	return s.storage.Set(ctx, "alert_emails", value)
}

// GetAlertCheckHour devuelve la hora del check de alertas.
// Fallback a billing_generation_hour si no está configurada.
func (s *SystemSettingsService) GetAlertCheckHour(ctx context.Context) (int, error) {
	val, err := s.storage.Get(ctx, "alert_check_hour")
	if err != nil {
		return 0, err
	}
	if val == "" {
		return s.GetBillingGenerationHour(ctx)
	}
	hour, err := strconv.Atoi(val)
	if err != nil {
		return 0, nil
	}
	return hour, nil
}

func (s *SystemSettingsService) SetAlertCheckHour(ctx context.Context, hour int) error {
	if hour < 0 || hour > 23 {
		return nil
	}
	return s.storage.Set(ctx, "alert_check_hour", strconv.Itoa(hour))
}

// ---- Email alerts master toggle (SPEC-037) ----

// EmailAlertsEnabledKey es la key del toggle maestro "Alertas por Email".
// Espejo de VoiceMonkeyEnabledKey.
const EmailAlertsEnabledKey = "email_alerts_enabled"

// GetEmailAlertsEnabled indica si el toggle maestro de alertas por email está on.
func (s *SystemSettingsService) GetEmailAlertsEnabled(ctx context.Context) (bool, error) {
	return s.getBoolSetting(ctx, EmailAlertsEnabledKey)
}

// SetEmailAlertsEnabled persiste el toggle maestro "Alertas por Email".
func (s *SystemSettingsService) SetEmailAlertsEnabled(ctx context.Context, enabled bool) error {
	return s.setBoolSetting(ctx, EmailAlertsEnabledKey, enabled)
}

// ---- Voice Monkey configuration (SPEC-033) ----

// Claves de config de Voice Monkey en system_settings.
const (
	VoiceMonkeyEnabledKey    = "voicemonkey_enabled"
	VoiceMonkeySendAlertsKey = "voicemonkey_send_alerts"
	VoiceMonkeyTokenKey      = "voicemonkey_token"
	VoiceMonkeyDeviceKey     = "voicemonkey_device"
)

// GetVoiceMonkeyConfig devuelve la config completa (incluye token/device).
// Uso interno: VoiceMonkeyService. NUNCA exponer en responses de API.
func (s *SystemSettingsService) GetVoiceMonkeyConfig(ctx context.Context) (*models.VoiceMonkeyConfig, error) {
	enabled, err := s.getBoolSetting(ctx, VoiceMonkeyEnabledKey)
	if err != nil {
		return nil, err
	}
	sendAlerts, err := s.getBoolSetting(ctx, VoiceMonkeySendAlertsKey)
	if err != nil {
		return nil, err
	}
	token, err := s.storage.Get(ctx, VoiceMonkeyTokenKey)
	if err != nil {
		return nil, err
	}
	device, err := s.storage.Get(ctx, VoiceMonkeyDeviceKey)
	if err != nil {
		return nil, err
	}
	return &models.VoiceMonkeyConfig{
		Enabled:    enabled,
		SendAlerts: sendAlerts,
		Token:      token,
		Device:     device,
	}, nil
}

// GetVoiceMonkeyConfigPublic devuelve la config SIN credenciales (token/device).
// Se usa en responses de API. Incluye flag Configured.
func (s *SystemSettingsService) GetVoiceMonkeyConfigPublic(ctx context.Context) (*models.VoiceMonkeyConfigPublic, error) {
	cfg, err := s.GetVoiceMonkeyConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &models.VoiceMonkeyConfigPublic{
		Enabled:    cfg.Enabled,
		SendAlerts: cfg.SendAlerts,
		Configured: s.isVoiceMonkeyConfigured(cfg),
	}, nil
}

// SetVoiceMonkeyConfig guarda solo las credenciales (token/device). Vacíos no
// se sobrescriben si ya tienen un valor guardado. Los toggles booleanos se
// persisten por separado via SetVoiceMonkeyEnabled / SetVoiceMonkeySendAlerts
// para permitir updates parciales sin pisarse (REQ-021).
func (s *SystemSettingsService) SetVoiceMonkeyConfig(ctx context.Context, cfg *models.VoiceMonkeyConfig) error {
	if err := s.setIfNonEmpty(ctx, VoiceMonkeyTokenKey, cfg.Token); err != nil {
		return err
	}
	if err := s.setIfNonEmpty(ctx, VoiceMonkeyDeviceKey, cfg.Device); err != nil {
		return err
	}
	return nil
}

// SetVoiceMonkeyEnabled persiste el toggle maestro "Activar Voice Monkey".
func (s *SystemSettingsService) SetVoiceMonkeyEnabled(ctx context.Context, enabled bool) error {
	return s.setBoolSetting(ctx, VoiceMonkeyEnabledKey, enabled)
}

// SetVoiceMonkeySendAlerts persiste el toggle "Enviar alertas".
func (s *SystemSettingsService) SetVoiceMonkeySendAlerts(ctx context.Context, sendAlerts bool) error {
	return s.setBoolSetting(ctx, VoiceMonkeySendAlertsKey, sendAlerts)
}

// ClearVoiceMonkey limpia las credenciales y resetea ambos toggles a OFF
// (botón "Reconfigurar", REQ-020).
func (s *SystemSettingsService) ClearVoiceMonkey(ctx context.Context) error {
	for _, key := range []string{VoiceMonkeyTokenKey, VoiceMonkeyDeviceKey} {
		if err := s.storage.Set(ctx, key, ""); err != nil {
			return err
		}
	}
	if err := s.setBoolSetting(ctx, VoiceMonkeyEnabledKey, false); err != nil {
		return err
	}
	return s.setBoolSetting(ctx, VoiceMonkeySendAlertsKey, false)
}

// IsVoiceMonkeyConfigured indica si hay token y device guardados.
func (s *SystemSettingsService) IsVoiceMonkeyConfigured(ctx context.Context) (bool, error) {
	cfg, err := s.GetVoiceMonkeyConfig(ctx)
	if err != nil {
		return false, err
	}
	return s.isVoiceMonkeyConfigured(cfg), nil
}

// IsVoiceMonkeyEnabled indica si el toggle maestro de Voice Monkey está on.
func (s *SystemSettingsService) IsVoiceMonkeyEnabled(ctx context.Context) (bool, error) {
	return s.getBoolSetting(ctx, VoiceMonkeyEnabledKey)
}

// IsVoiceMonkeySendingAlerts indica si el toggle "enviar alertas" está on.
func (s *SystemSettingsService) IsVoiceMonkeySendingAlerts(ctx context.Context) (bool, error) {
	return s.getBoolSetting(ctx, VoiceMonkeySendAlertsKey)
}

func (s *SystemSettingsService) isVoiceMonkeyConfigured(cfg *models.VoiceMonkeyConfig) bool {
	return cfg.Token != "" && cfg.Device != ""
}

func (s *SystemSettingsService) getBoolSetting(ctx context.Context, key string) (bool, error) {
	val, err := s.storage.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

func (s *SystemSettingsService) setBoolSetting(ctx context.Context, key string, value bool) error {
	v := "0"
	if value {
		v = "1"
	}
	return s.storage.Set(ctx, key, v)
}

// parseEmails separa una lista comma-separated en emails limpios.
func parseEmails(val string) []string {
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	return sanitizeEmails(parts)
}

func sanitizeEmails(parts []string) []string {
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
