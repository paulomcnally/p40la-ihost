// Package config carga la configuración del servidor MCP de p40la-ihost.
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL     = "http://ihost.local:8088"
	defaultTimeout     = 15 * time.Second
	defaultTimezone    = "America/Managua"
	maxTimeout         = 60 * time.Second
	apiKeyPrefix       = "p40la_"
	defaultListen      = "127.0.0.1:8090"
	defaultAdminListen = "127.0.0.1:8091"
	defaultStateDir    = "/data"
	defaultRateLimit   = 60
	defaultAuditMaxMB  = 5

	// TransportStdio usa stdin/stdout (default, compatible con SPEC-145).
	TransportStdio = "stdio"
	// TransportHTTP sirve Streamable HTTP en /mcp y requiere OAuth (SPEC-147).
	TransportHTTP = "http"
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

	// SPEC-147: modo servidor HTTP.
	Transport         string
	Listen            string
	AdminListen       string
	PublicURL         string
	OAuthSubject      string
	OAuthPasswordHash string
	StateDir          string
	RemoteWrites      bool
	RateLimit         int
	AuditMaxMB        int
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

	cfg.Transport = strings.ToLower(strings.TrimSpace(getenv("P40LA_MCP_TRANSPORT")))
	if cfg.Transport == "" {
		cfg.Transport = TransportStdio
	}
	if cfg.Transport != TransportStdio && cfg.Transport != TransportHTTP {
		return Config{}, fmt.Errorf("P40LA_MCP_TRANSPORT debe ser %q o %q", TransportStdio, TransportHTTP)
	}
	cfg.Listen = defaultListen
	cfg.AdminListen = defaultAdminListen
	cfg.StateDir = defaultStateDir
	cfg.RateLimit = defaultRateLimit
	cfg.AuditMaxMB = defaultAuditMaxMB
	cfg.OAuthSubject = "owner"

	if cfg.Transport == TransportHTTP {
		if raw := strings.TrimSpace(getenv("P40LA_MCP_LISTEN")); raw != "" {
			cfg.Listen = raw
		}
		if raw := strings.TrimSpace(getenv("P40LA_MCP_ADMIN_LISTEN")); raw != "" {
			cfg.AdminListen = raw
		}
		if err := validateLoopbackAddr("P40LA_MCP_LISTEN", cfg.Listen); err != nil {
			return Config{}, err
		}
		if err := validateLoopbackAddr("P40LA_MCP_ADMIN_LISTEN", cfg.AdminListen); err != nil {
			return Config{}, err
		}
		if cfg.Listen == cfg.AdminListen {
			return Config{}, fmt.Errorf("P40LA_MCP_LISTEN y P40LA_MCP_ADMIN_LISTEN deben ser distintos")
		}

		cfg.PublicURL = strings.TrimRight(strings.TrimSpace(getenv("P40LA_MCP_PUBLIC_URL")), "/")
		if cfg.PublicURL == "" {
			return Config{}, fmt.Errorf("P40LA_MCP_PUBLIC_URL es obligatoria en modo HTTP (ej: https://<nodo>.<tailnet>.ts.net)")
		}
		public, err := url.Parse(cfg.PublicURL)
		if err != nil || public.Host == "" {
			return Config{}, fmt.Errorf("P40LA_MCP_PUBLIC_URL debe ser una URL absoluta")
		}
		if public.Path != "" && public.Path != "/" {
			return Config{}, fmt.Errorf("P40LA_MCP_PUBLIC_URL debe ser el origen, sin path (la tool vive en /mcp)")
		}
		if public.RawQuery != "" || public.Fragment != "" {
			return Config{}, fmt.Errorf("P40LA_MCP_PUBLIC_URL no puede incluir query ni fragment")
		}
		if public.Scheme != "https" && !(public.Scheme == "http" && isLoopbackHost(public.Hostname())) {
			return Config{}, fmt.Errorf("P40LA_MCP_PUBLIC_URL debe usar https (http solo para loopback en desarrollo)")
		}

		cfg.OAuthPasswordHash = strings.TrimSpace(getenv("P40LA_MCP_OAUTH_PASSWORD_HASH"))
		if cfg.OAuthPasswordHash == "" {
			return Config{}, fmt.Errorf("P40LA_MCP_OAUTH_PASSWORD_HASH es obligatoria en modo HTTP; generá una con p40la-mcp -hash-password")
		}
		if !strings.HasPrefix(cfg.OAuthPasswordHash, "$2a$") &&
			!strings.HasPrefix(cfg.OAuthPasswordHash, "$2b$") &&
			!strings.HasPrefix(cfg.OAuthPasswordHash, "$2y$") {
			return Config{}, fmt.Errorf("P40LA_MCP_OAUTH_PASSWORD_HASH debe ser un hash bcrypt")
		}

		if raw := strings.TrimSpace(getenv("P40LA_MCP_OAUTH_SUBJECT")); raw != "" {
			cfg.OAuthSubject = raw
		}
		if raw := strings.TrimSpace(getenv("P40LA_MCP_STATE_DIR")); raw != "" {
			cfg.StateDir = raw
		}
		if raw := strings.TrimSpace(getenv("P40LA_MCP_REMOTE_WRITES")); raw != "" {
			value, err := strconv.ParseBool(raw)
			if err != nil {
				return Config{}, fmt.Errorf("P40LA_MCP_REMOTE_WRITES debe ser true o false")
			}
			cfg.RemoteWrites = value
		}
		if raw := strings.TrimSpace(getenv("P40LA_MCP_RATE_LIMIT")); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 || value > 600 {
				return Config{}, fmt.Errorf("P40LA_MCP_RATE_LIMIT debe ser un entero entre 1 y 600")
			}
			cfg.RateLimit = value
		}
		if raw := strings.TrimSpace(getenv("P40LA_MCP_AUDIT_MAX_MB")); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 || value > 100 {
				return Config{}, fmt.Errorf("P40LA_MCP_AUDIT_MAX_MB debe ser un entero entre 1 y 100")
			}
			cfg.AuditMaxMB = value
		}
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

// WritesEnabled indica si las tools pueden escribir datos. En modo HTTP exige
// además el gate remoto (SPEC-147).
func (c Config) WritesEnabled() bool {
	if c.Transport == TransportHTTP {
		return c.AllowWrites && c.RemoteWrites
	}
	return c.AllowWrites
}

// IsHTTP indica si el binario corre como servidor HTTP.
func (c Config) IsHTTP() bool {
	return c.Transport == TransportHTTP
}

// validateLoopbackAddr exige una dirección host:puerto con host loopback.
func validateLoopbackAddr(name, addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%s debe tener formato host:puerto", name)
	}
	if port == "" {
		return fmt.Errorf("%s debe incluir puerto", name)
	}
	if !isLoopbackHost(host) {
		return fmt.Errorf("%s debe escuchar en loopback (127.0.0.1 o ::1)", name)
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}
