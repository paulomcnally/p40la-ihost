package config

import (
	"testing"
	"time"
)

func testEnv(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(testEnv(map[string]string{
		"P40LA_API_KEY": "p40la_demo-demo-demo-demo-demo-demo-demo",
	}))
	if err != nil {
		t.Fatalf("cargar configuración: %v", err)
	}
	if cfg.BaseURL != "http://ihost.local:8088" {
		t.Fatalf("base URL = %q", cfg.BaseURL)
	}
	if cfg.Timeout != 15*time.Second {
		t.Fatalf("timeout = %s", cfg.Timeout)
	}
	if cfg.AllowWrites {
		t.Fatal("las escrituras deben estar deshabilitadas por defecto")
	}
	if cfg.Location == nil || cfg.Location.String() != "America/Managua" {
		t.Fatalf("timezone = %v", cfg.Location)
	}
}

func TestLoadRequiresAPIKey(t *testing.T) {
	if _, err := Load(testEnv(map[string]string{})); err == nil {
		t.Fatal("quiero error sin API key")
	}
	if _, err := Load(testEnv(map[string]string{"P40LA_API_KEY": "otra-cosa"})); err == nil {
		t.Fatal("quiero error con API key sin prefijo")
	}
	if _, err := Load(testEnv(map[string]string{"P40LA_API_KEY": "p40la_corta"})); err == nil {
		t.Fatal("quiero error con API key demasiado corta")
	}
}

func TestLoadValidation(t *testing.T) {
	base := map[string]string{"P40LA_API_KEY": "p40la_demo-demo-demo-demo-demo-demo-demo"}

	cases := []map[string]string{
		{"P40LA_BASE_URL": "ihost.local:8088"},
		{"P40LA_BASE_URL": "ftp://ihost.local"},
		{"P40LA_TIMEOUT": "0s"},
		{"P40LA_TIMEOUT": "2m"},
		{"P40LA_MCP_ALLOW_WRITES": "quizás"},
		{"P40LA_TIMEZONE": "No/Existe"},
	}
	for _, extra := range cases {
		values := make(map[string]string, len(base)+len(extra))
		for key, value := range base {
			values[key] = value
		}
		for key, value := range extra {
			values[key] = value
		}
		if _, err := Load(testEnv(values)); err == nil {
			t.Fatalf("quiero error para %v", extra)
		}
	}
}

func TestLoadCustom(t *testing.T) {
	cfg, err := Load(testEnv(map[string]string{
		"P40LA_BASE_URL":         "https://demo.example.com/p40la/",
		"P40LA_API_KEY":          "p40la_demo-demo-demo-demo-demo-demo-demo",
		"P40LA_MCP_ALLOW_WRITES": "true",
		"P40LA_TIMEOUT":          "20s",
		"P40LA_TIMEZONE":         "UTC",
	}))
	if err != nil {
		t.Fatalf("cargar configuración: %v", err)
	}
	if cfg.BaseURL != "https://demo.example.com/p40la" {
		t.Fatalf("base URL = %q", cfg.BaseURL)
	}
	if !cfg.WritesEnabled() {
		t.Fatal("quiero escrituras habilitadas")
	}
	if cfg.Timeout != 20*time.Second {
		t.Fatalf("timeout = %s", cfg.Timeout)
	}
	if cfg.Location.String() != "UTC" {
		t.Fatalf("timezone = %s", cfg.Location)
	}
}

const demoHash = "$2a$10$abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func httpEnv(extra map[string]string) map[string]string {
	values := map[string]string{
		"P40LA_API_KEY":                 "p40la_demo-demo-demo-demo-demo-demo-demo",
		"P40LA_MCP_TRANSPORT":           "http",
		"P40LA_MCP_PUBLIC_URL":          "https://demo-node.example.ts.net",
		"P40LA_MCP_OAUTH_PASSWORD_HASH": demoHash,
		"P40LA_MCP_STATE_DIR":           "/tmp/demo-mcp-state",
	}
	for key, value := range extra {
		values[key] = value
	}
	return values
}

func TestLoadHTTPValid(t *testing.T) {
	cfg, err := Load(testEnv(httpEnv(map[string]string{
		"P40LA_MCP_ALLOW_WRITES":  "true",
		"P40LA_MCP_REMOTE_WRITES": "false",
		"P40LA_MCP_RATE_LIMIT":    "30",
		"P40LA_MCP_AUDIT_MAX_MB":  "2",
	})))
	if err != nil {
		t.Fatalf("cargar HTTP: %v", err)
	}
	if !cfg.IsHTTP() {
		t.Fatal("transporte HTTP esperado")
	}
	if cfg.Listen != "127.0.0.1:8090" || cfg.AdminListen != "127.0.0.1:8091" {
		t.Fatalf("listeners = %q / %q", cfg.Listen, cfg.AdminListen)
	}
	if cfg.WritesEnabled() {
		t.Fatal("writes remotos deben estar deshabilitados")
	}
	cfg.RemoteWrites = true
	if !cfg.WritesEnabled() {
		t.Fatal("writes locales+remotos deben habilitarse")
	}
	if cfg.RateLimit != 30 || cfg.AuditMaxMB != 2 {
		t.Fatalf("rate=%d audit=%d", cfg.RateLimit, cfg.AuditMaxMB)
	}
}

func TestLoadHTTPValidation(t *testing.T) {
	cases := []map[string]string{
		{"P40LA_MCP_PUBLIC_URL": ""},
		{"P40LA_MCP_OAUTH_PASSWORD_HASH": ""},
		{"P40LA_MCP_OAUTH_PASSWORD_HASH": "texto"},
		{"P40LA_MCP_LISTEN": "0.0.0.0:8090"},
		{"P40LA_MCP_ADMIN_LISTEN": "192.0.2.10:8091"},
		{"P40LA_MCP_LISTEN": "127.0.0.1:8091"}, // igual al admin
		{"P40LA_MCP_PUBLIC_URL": "http://demo.example.com"},
		{"P40LA_MCP_PUBLIC_URL": "https://demo.example.com/mcp"},
		{"P40LA_MCP_REMOTE_WRITES": "quizás"},
		{"P40LA_MCP_RATE_LIMIT": "0"},
		{"P40LA_MCP_AUDIT_MAX_MB": "101"},
	}
	for _, extra := range cases {
		if _, err := Load(testEnv(httpEnv(extra))); err == nil {
			t.Fatalf("quiero error para %v", extra)
		}
	}
}

func TestLoadHTTPAllowsLoopbackDevelopment(t *testing.T) {
	cfg, err := Load(testEnv(httpEnv(map[string]string{
		"P40LA_MCP_PUBLIC_URL":   "http://127.0.0.1:8090",
		"P40LA_MCP_LISTEN":       "127.0.0.1:8090",
		"P40LA_MCP_ADMIN_LISTEN": "127.0.0.1:8091",
	})))
	if err != nil {
		t.Fatalf("loopback HTTP: %v", err)
	}
	if cfg.PublicURL != "http://127.0.0.1:8090" {
		t.Fatalf("public URL = %q", cfg.PublicURL)
	}
}
