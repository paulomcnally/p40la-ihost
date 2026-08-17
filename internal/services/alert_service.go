package services

import (
	"context"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// AlertService maneja el catálogo de alertas (tabla `alerts`) y sus flags
// de canal (mail / voz). El catálogo se define en código y se siembra en
// main.go vía Seed() (INSERT OR IGNORE, no borra config del usuario).
type AlertService struct {
	storage *storage.AlertStorage
}

func NewAlertService(alertStorage *storage.AlertStorage) *AlertService {
	return &AlertService{storage: alertStorage}
}

// catalog define las alertas conocidas. Agregar una alerta nueva = sumar
// una entrada aquí (el seed la inserta en DB al arrancar).
func catalog() []models.Alert {
	return []models.Alert{
		{
			Key:         models.AlertKeyInsurance,
			Title:       "Seguros de autos vencidos",
			Description: "Recibir un email o aviso por Alexa cuando un seguro de auto vence o un auto queda sin cobertura.",
			Speech:      "Paulo, tienes autos sin seguro o con seguros vencidos.",
		},
		{
			Key:         models.AlertKeyBillCreated,
			Title:       "Nueva factura generada",
			Description: "Informativo cuando el sistema genera una factura automáticamente.",
			Speech:      "Paulo, se generó una nueva factura automática.",
		},
		{
			Key:         models.AlertKeyBillSummary,
			Title:       "Resumen diario de facturas",
			Description: "Resumen diario de todas las facturas pendientes.",
			Speech:      "Paulo, tienes {n} facturas pendientes.",
		},
	}
}

// Seed inserta las alertas del catálogo que falten. Idempotente.
func (s *AlertService) Seed(ctx context.Context) error {
	return s.storage.Seed(ctx, catalog())
}

// List devuelve todas las alertas.
func (s *AlertService) List(ctx context.Context) ([]models.Alert, error) {
	return s.storage.List(ctx)
}

// GetByKey devuelve una alerta por key.
func (s *AlertService) GetByKey(ctx context.Context, key string) (*models.Alert, error) {
	return s.storage.GetByKey(ctx, key)
}

// SetFlags actualiza mail_enabled / voice_enabled de una alerta.
func (s *AlertService) SetFlags(ctx context.Context, key string, mailEnabled, voiceEnabled *bool) error {
	return s.storage.SetFlags(ctx, key, mailEnabled, voiceEnabled)
}

// IsEnabled devuelve true si la alerta está habilitada para el canal dado.
// Si la alerta no existe, devuelve false (comportamiento seguro por defecto).
func (s *AlertService) IsEnabled(ctx context.Context, key string, channel models.AlertChannel) (bool, error) {
	a, err := s.storage.GetByKey(ctx, key)
	if err != nil {
		return false, err
	}
	if a == nil {
		return false, nil
	}
	switch channel {
	case models.AlertChannelMail:
		return a.MailEnabled, nil
	case models.AlertChannelVoice:
		return a.VoiceEnabled, nil
	default:
		return false, nil
	}
}

// Speech devuelve el texto TTS de la alerta (desde DB, con fallback al catálogo).
func (s *AlertService) Speech(ctx context.Context, key string) (string, error) {
	a, err := s.storage.GetByKey(ctx, key)
	if err != nil {
		return "", err
	}
	if a != nil {
		return a.Speech, nil
	}
	for _, c := range catalog() {
		if c.Key == key {
			return c.Speech, nil
		}
	}
	return "", nil
}
