package services

import (
	"context"
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
