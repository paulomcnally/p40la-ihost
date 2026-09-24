package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Errores de dominio del cliente de automation (SPEC-092).
var (
	ErrAutomationNotConfigured = errors.New("automation no configurada en Settings")
	ErrAutomationUnauthorized  = errors.New("api_key de automation inválida")
	ErrAutomationUnreachable   = errors.New("automation inalcanzable")
)

const (
	// AutomationBaseURLKey es la key de system_settings de la base URL de
	// p40la-ihost-automation (SPEC-092).
	AutomationBaseURLKey = "automation_base_url"
	// AutomationAPIKeyKey es la key de system_settings de la api_key de
	// webhooks de p40la-ihost-automation (token del endpoint webhook:run).
	AutomationAPIKeyKey = "automation_api_key"
	// DefaultAutomationBaseURL es la base URL por defecto del proyecto
	// hermano p40la-ihost-automation (puerto 8089, distinto de 8088).
	DefaultAutomationBaseURL = "http://ihost.local:8089"

	automationClientTimeout = 30 * time.Second
)

// AutomationConfig agrupa la configuración del proyecto de automation.
type AutomationConfig struct {
	BaseURL string
	APIKey  string
}

// AutomationClient es el cliente HTTP hacia p40la-ihost-automation. Ejecuta el
// job de una cuenta (llamado externo del plugin + entrega al webhook) vía el
// endpoint POST /api/accounts/{id}/webhook:run, autenticado por token en el
// header X-Webhook-Key (SPEC-017 en el repo de automation).
type AutomationClient struct {
	settings *SystemSettingsService
	client   *http.Client
}

// NewAutomationClient crea un nuevo AutomationClient.
func NewAutomationClient(settings *SystemSettingsService) *AutomationClient {
	return &AutomationClient{
		settings: settings,
		client:   &http.Client{Timeout: automationClientTimeout},
	}
}

// IsConfigured indica si automation tiene base URL y api_key configuradas.
func (c *AutomationClient) IsConfigured(ctx context.Context) (bool, error) {
	return c.settings.IsAutomationConfigured(ctx)
}

// SyncAccount ejecuta el job de la cuenta en automation (fetch del plugin +
// envío al webhook) y devuelve la cantidad de facturas entregadas/fallidas.
func (c *AutomationClient) SyncAccount(ctx context.Context, accountID int64) (delivered, failed int, err error) {
	cfg, err := c.settings.GetAutomationConfig(ctx)
	if err != nil {
		return 0, 0, err
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.APIKey) == "" {
		return 0, 0, ErrAutomationNotConfigured
	}

	base := strings.TrimRight(cfg.BaseURL, "/")
	endpoint := fmt.Sprintf("%s/api/accounts/%d/webhook:run", base, accountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(nil))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("X-Webhook-Key", cfg.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %v", ErrAutomationUnreachable, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode == http.StatusOK {
		var result struct {
			Delivered int `json:"delivered"`
			Failed    int `json:"failed"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return 0, 0, fmt.Errorf("respuesta inválida de automation: %w", err)
		}
		return result.Delivered, result.Failed, nil
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return 0, 0, ErrAutomationUnauthorized
	}

	// Propagar el mensaje de automation (mapWebhookError / errores de dominio).
	msg := errorMessageFromBody(body)
	if msg == "" {
		msg = resp.Status
	}
	return 0, 0, fmt.Errorf("automation respondió %d: %s", resp.StatusCode, msg)
}

// errorMessageFromBody extrae el campo message de un body JSON de error de
// automation ({ "error": ..., "message": ... }). Devuelve "" si no aplica.
func errorMessageFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Message)
}
