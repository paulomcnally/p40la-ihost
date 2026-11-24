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
		"P40LA_EMAIL":    "demo@example.com",
		"P40LA_PASSWORD": "demo-password",
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

func TestLoadRequiresCredentials(t *testing.T) {
	if _, err := Load(testEnv(map[string]string{})); err == nil {
		t.Fatal("quiero error sin credenciales")
	}
	if _, err := Load(testEnv(map[string]string{"P40LA_EMAIL": "demo@example.com"})); err == nil {
		t.Fatal("quiero error con email sin contraseña")
	}
	if _, err := Load(testEnv(map[string]string{"P40LA_SESSION_COOKIE": "demo-cookie"})); err != nil {
		t.Fatalf("la cookie debe ser suficiente: %v", err)
	}
}

func TestLoadValidation(t *testing.T) {
	base := map[string]string{"P40LA_EMAIL": "demo@example.com", "P40LA_PASSWORD": "demo-password"}

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
		"P40LA_SESSION_COOKIE":   "demo-cookie",
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
	if cfg.Email != "" || cfg.Password != "" {
		t.Fatal("la cookie debe tener prioridad sobre usuario/clave")
	}
}
