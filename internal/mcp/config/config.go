// Package config carga la configuración del servidor MCP de p40la-ihost.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL  = "http://ihost.local:8088"
	defaultTimeout  = 15 * time.Second
	defaultTimezone = "America/Managua"
	maxTimeout      = 60 * time.Second
	apiKeyPrefix    = "p40la_"
)

// Config es la configuración efectiva del binario MCP. La API key nunca se
// imprime ni se pasa por argumentos de línea de comandos.
type Config struct {
	BaseURL     string
	APIKey      string
	AllowWrites bool
	Timeout     time.Duration
	Timezone    string
	Location    *time.Location
}

// Load lee la configuración desde las variables de entorno usando getenv.
// Si getenv es nil se usa os.Getenv.
func Load(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	cfg := Config{
		BaseURL:  strings.TrimSpace(getenv("P40LA_BASE_URL")),
		APIKey:   strings.TrimSpace(getenv("P40LA_API_KEY")),
		Timeout:  defaultTimeout,
		Timezone: defaultTimezone,
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return Config{}, fmt.Errorf("P40LA_BASE_URL debe ser una URL absoluta http o https")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return Config{}, fmt.Errorf("P40LA_BASE_URL debe usar http o https")
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	if cfg.APIKey == "" {
		return Config{}, fmt.Errorf("P40LA_API_KEY es obligatoria; generá una con scripts/create-api-key.sh")
	}
	if !strings.HasPrefix(cfg.APIKey, apiKeyPrefix) || len(cfg.APIKey) < 20 || len(cfg.APIKey) > 200 {
		return Config{}, fmt.Errorf("P40LA_API_KEY debe tener el formato p40la_...")
	}

	if raw := strings.TrimSpace(getenv("P40LA_MCP_ALLOW_WRITES")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("P40LA_MCP_ALLOW_WRITES debe ser true o false")
		}
		cfg.AllowWrites = value
	}

	if raw := strings.TrimSpace(getenv("P40LA_TIMEOUT")); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil || timeout <= 0 || timeout > maxTimeout {
			return Config{}, fmt.Errorf("P40LA_TIMEOUT debe ser una duración entre 1s y %s", maxTimeout)
		}
		cfg.Timeout = timeout
	}

	if raw := strings.TrimSpace(getenv("P40LA_TIMEZONE")); raw != "" {
		cfg.Timezone = raw
	}
	location, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return Config{}, fmt.Errorf("P40LA_TIMEZONE inválida: %s", cfg.Timezone)
	}
	cfg.Location = location
	return cfg, nil
}

// WritesEnabled indica si las tools pueden escribir datos.
func (c Config) WritesEnabled() bool {
	return c.AllowWrites
}
