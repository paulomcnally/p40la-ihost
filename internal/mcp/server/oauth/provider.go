// Package oauth implementa un proveedor OAuth 2.1 mínimo de usuario único para
// el servidor MCP remoto (SPEC-147). No emite JWT: usa tokens opacos hasheados.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/server/limiter"
)

const (
	scopeRead  = "mcp:read"
	scopeWrite = "mcp:write"
)

// Config es la configuración del proveedor OAuth.
type Config struct {
	Issuer       string
	Subject      string
	PasswordHash string
	StateDir     string
}

// Provider expone los endpoints OAuth y valida bearer tokens.
type Provider struct {
	issuer       string
	resource     string
	subject      string
	passwordHash string
	store        *store

	now func() time.Time

	loginLimit    *limiter.Limiter
	registerLimit *limiter.Limiter
	tokenLimit    *limiter.Limiter
}

// New crea el proveedor y carga (o inicializa) el estado persistente.
func New(cfg Config) (*Provider, error) {
	issuer := strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/")
	parsed, err := url.Parse(issuer)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("issuer OAuth inválido")
	}
	if cfg.Subject == "" {
		cfg.Subject = "owner"
	}
	if cfg.PasswordHash == "" {
		return nil, fmt.Errorf("password hash obligatorio")
	}
	st, err := newStore(cfg.StateDir)
	if err != nil {
		return nil, err
	}
	return &Provider{
		issuer:        issuer,
		resource:      issuer + "/mcp",
		subject:       cfg.Subject,
		passwordHash:  cfg.PasswordHash,
		store:         st,
		now:           time.Now,
		loginLimit:    limiter.New(5, time.Minute),
		registerLimit: limiter.New(10, time.Hour),
		tokenLimit:    limiter.New(20, time.Minute),
	}, nil
}

// Issuer devuelve el origen público.
func (p *Provider) Issuer() string { return p.issuer }

// Resource devuelve la URL canónica del endpoint MCP.
func (p *Provider) Resource() string { return p.resource }

// Handler monta los endpoints OAuth y de descubrimiento.
func (p *Provider) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", p.protectedResource)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource/mcp", p.protectedResource)
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", p.authServerMetadata)
	mux.HandleFunc("POST /register", p.register)
	mux.HandleFunc("GET /authorize", p.authorizeGet)
	mux.HandleFunc("POST /authorize", p.authorizePost)
	mux.HandleFunc("POST /token", p.token)
	return mux
}

// ValidateToken comprueba un bearer token y devuelve el sujeto.
func (p *Provider) ValidateToken(_ context.Context, raw string) (string, bool) {
	if !validTokenFormat(raw) {
		return "", false
	}
	entry, ok := p.store.validateAccess(raw, p.now())
	if !ok {
		return "", false
	}
	return entry.Subject, true
}

// RevokeAll invalida todos los tokens y códigos emitidos.
func (p *Provider) RevokeAll() error {
	return p.store.revokeAll()
}

func (p *Provider) protectedResource(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 p.resource,
		"authorization_servers":    []string{p.issuer},
		"scopes_supported":         []string{scopeRead, scopeWrite},
		"bearer_methods_supported": []string{"header"},
	})
}

func (p *Provider) authServerMetadata(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                p.issuer,
		"authorization_endpoint":                p.issuer + "/authorize",
		"token_endpoint":                        p.issuer + "/token",
		"registration_endpoint":                 p.issuer + "/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      []string{scopeRead, scopeWrite},
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// clientIP prioriza X-Forwarded-For (Funnel/proxy) y cae a RemoteAddr.
func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func validTokenFormat(raw string) bool {
	if len(raw) < 32 || len(raw) > 128 {
		return false
	}
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}
