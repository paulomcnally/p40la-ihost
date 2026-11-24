package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// APIKeyPrefix identifica las API keys del sistema (SPEC-145).
const APIKeyPrefix = "p40la_"

const (
	mcpAPIKeyHashSetting     = "mcp_api_key_hash"
	mcpAPIKeyPrefixSetting   = "mcp_api_key_prefix"
	mcpAPIKeyCreatedSetting  = "mcp_api_key_created_at"
	mcpAPIKeyLastUsedSetting = "mcp_api_key_last_used_at"
	mcpAPIKeyLastUsedTTL     = time.Minute
)

// APIKeyStatus describe el estado de la key global sin exponer el secreto.
type APIKeyStatus struct {
	Configured bool       `json:"configured"`
	Prefix     string     `json:"prefix,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// APIKeyService gestiona la única API key global del sistema. La key completa
// se muestra una sola vez; en system_settings solo persisten el hash y el
// prefijo (SPEC-145).
type APIKeyService struct {
	settings *storage.SystemSettingsStorage
}

// NewAPIKeyService crea el servicio de API key global.
func NewAPIKeyService(settings *storage.SystemSettingsStorage) *APIKeyService {
	return &APIKeyService{settings: settings}
}

// Status devuelve el estado de la key sin el secreto.
func (s *APIKeyService) Status(ctx context.Context) (*APIKeyStatus, error) {
	hash, err := s.settings.Get(ctx, mcpAPIKeyHashSetting)
	if err != nil {
		return nil, err
	}
	status := &APIKeyStatus{Configured: hash != ""}
	if !status.Configured {
		return status, nil
	}
	status.Prefix, err = s.settings.Get(ctx, mcpAPIKeyPrefixSetting)
	if err != nil {
		return nil, err
	}
	created, err := s.settings.Get(ctx, mcpAPIKeyCreatedSetting)
	if err != nil {
		return nil, err
	}
	status.CreatedAt = parseAPIKeyTime(created)
	lastUsed, err := s.settings.Get(ctx, mcpAPIKeyLastUsedSetting)
	if err != nil {
		return nil, err
	}
	status.LastUsedAt = parseAPIKeyTime(lastUsed)
	return status, nil
}

// Regenerate crea una nueva key global, invalida la anterior y devuelve el
// secreto completo. Es la única operación que expone la key.
func (s *APIKeyService) Regenerate(ctx context.Context) (string, *APIKeyStatus, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", nil, fmt.Errorf("generar API key: %w", err)
	}
	key := APIKeyPrefix + base64.RawURLEncoding.EncodeToString(random)
	hash := hashAPIKey(key)
	prefix := key
	if len(prefix) > 14 {
		prefix = prefix[:14]
	}
	now := time.Now().UTC()
	created := now.Format(time.RFC3339)

	if err := s.settings.Set(ctx, mcpAPIKeyPrefixSetting, prefix); err != nil {
		return "", nil, err
	}
	if err := s.settings.Set(ctx, mcpAPIKeyCreatedSetting, created); err != nil {
		return "", nil, err
	}
	if err := s.settings.Set(ctx, mcpAPIKeyLastUsedSetting, ""); err != nil {
		return "", nil, err
	}
	// El hash se escribe al final: si algo falla antes, no se invalida la key
	// anterior por accidente.
	if err := s.settings.Set(ctx, mcpAPIKeyHashSetting, hash); err != nil {
		return "", nil, err
	}
	status := &APIKeyStatus{Configured: true, Prefix: prefix, CreatedAt: &now}
	return key, status, nil
}

// Revoke invalida la key vigente de inmediato.
func (s *APIKeyService) Revoke(ctx context.Context) error {
	if err := s.settings.Set(ctx, mcpAPIKeyHashSetting, ""); err != nil {
		return err
	}
	if err := s.settings.Set(ctx, mcpAPIKeyPrefixSetting, ""); err != nil {
		return err
	}
	if err := s.settings.Set(ctx, mcpAPIKeyCreatedSetting, ""); err != nil {
		return err
	}
	return s.settings.Set(ctx, mcpAPIKeyLastUsedSetting, "")
}

// Validate comprueba la key en tiempo constante y actualiza last_used_at como
// máximo una vez por minuto. No devuelve el secreto ni lo registra.
func (s *APIKeyService) Validate(ctx context.Context, provided string) (bool, error) {
	provided = strings.TrimSpace(provided)
	if provided == "" || !strings.HasPrefix(provided, APIKeyPrefix) {
		return false, nil
	}
	expected, err := s.settings.Get(ctx, mcpAPIKeyHashSetting)
	if err != nil {
		return false, err
	}
	if expected == "" {
		return false, nil
	}
	providedHash := hashAPIKey(provided)
	if subtle.ConstantTimeCompare([]byte(providedHash), []byte(expected)) != 1 {
		return false, nil
	}
	s.touchLastUsed(ctx)
	return true, nil
}

func (s *APIKeyService) touchLastUsed(ctx context.Context) {
	raw, err := s.settings.Get(ctx, mcpAPIKeyLastUsedSetting)
	if err != nil {
		return
	}
	if lastUsed := parseAPIKeyTime(raw); lastUsed != nil && time.Since(*lastUsed) < mcpAPIKeyLastUsedTTL {
		return
	}
	_ = s.settings.Set(ctx, mcpAPIKeyLastUsedSetting, time.Now().UTC().Format(time.RFC3339))
}

func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func parseAPIKeyTime(raw string) *time.Time {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &parsed
}
