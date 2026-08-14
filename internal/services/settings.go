package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

const (
	SettingLanguage = "language"
)

var allowedLanguages = map[string]bool{
	"es": true,
	"en": true,
}

// AppSettingsService contiene la lógica de negocio para configuraciones de la aplicación.
type AppSettingsService struct {
	storage *storage.SettingsStorage
}

// NewAppSettingsService crea un nuevo AppSettingsService.
func NewAppSettingsService(st *storage.SettingsStorage) *AppSettingsService {
	return &AppSettingsService{storage: st}
}

// GetLanguage devuelve el idioma configurado, por defecto 'es'.
func (s *AppSettingsService) GetLanguage(ctx context.Context) (string, error) {
	value, err := s.storage.Get(ctx, SettingLanguage)
	if err != nil {
		return "", fmt.Errorf("obtener idioma: %w", err)
	}
	if value == "" || !allowedLanguages[value] {
		return "es", nil
	}
	return value, nil
}

// SetLanguage cambia el idioma de la aplicación.
func (s *AppSettingsService) SetLanguage(ctx context.Context, lang string) error {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if !allowedLanguages[lang] {
		return fmt.Errorf("idioma no soportado")
	}
	return s.storage.Set(ctx, SettingLanguage, lang)
}

// GetAll devuelve todas las configuraciones.
func (s *AppSettingsService) GetAll(ctx context.Context) (map[string]string, error) {
	settings, err := s.storage.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(settings))
	for _, st := range settings {
		result[st.Key] = st.Value
	}
	lang, err := s.GetLanguage(ctx)
	if err != nil {
		return nil, err
	}
	result[SettingLanguage] = lang
	return result, nil
}
