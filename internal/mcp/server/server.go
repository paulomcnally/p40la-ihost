// Package server expone el servidor MCP por Streamable HTTP con OAuth 2.1,
// auditoría y límites de uso (SPEC-147).
package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/audit"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/server/limiter"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/server/oauth"
)

const (
	maxMCPBody      = 1 << 20
	shutdownTimeout = 5 * time.Second
)

type ctxKey string

const (
	subjectKey   ctxKey = "subject"
	requestIDKey ctxKey = "request_id"
)

// Server agrupa los listeners HTTP público y administrativo.
type Server struct {
	cfg     config.Config
	version string
	logger  *slog.Logger
	mcp     *mcp.Server
	oauth   *oauth.Provider
	audit   *audit.Logger
	limiter *limiter.Limiter
	started time.Time
}

// New construye el servidor HTTP. Registra la auditoría en el servidor MCP.
func New(cfg config.Config, mcpServer *mcp.Server, logger *slog.Logger, version string) (*Server, error) {
	if !cfg.IsHTTP() {
		return nil, fmt.Errorf("el servidor HTTP requiere P40LA_MCP_TRANSPORT=http")
	}
	if logger == nil {
		logger = slog.Default()
	}
	auditLogger, err := audit.New(cfg.StateDir, cfg.AuditMaxMB)
	if err != nil {
		return nil, err
	}
	provider, err := oauth.New(oauth.Config{
		Issuer:       cfg.PublicURL,
		Subject:      cfg.OAuthSubject,
		PasswordHash: cfg.OAuthPasswordHash,
		StateDir:     cfg.StateDir,
	})
	if err != nil {
		_ = auditLogger.Close()
		return nil, err
	}
	s := &Server{
		cfg:     cfg,
		version: version,
		logger:  logger,
		mcp:     mcpServer,
		oauth:   provider,
		audit:   auditLogger,
		limiter: limiter.New(cfg.RateLimit, time.Minute),
		started: time.Now(),
	}
	mcpServer.AddReceivingMiddleware(s.auditMiddleware())
	return s, nil
}

// Run levanta los dos listeners hasta que ctx se cancele.
func (s *Server) Run(ctx context.Context) error {
	mainListener, err := net.Listen("tcp", s.cfg.Listen)
	if err != nil {
		return fmt.Errorf("escuchar MCP en %s: %w", s.cfg.Listen, err)
	}
	adminListener, err := net.Listen("tcp", s.cfg.AdminListen)
	if err != nil {
		_ = mainListener.Close()
		return fmt.Errorf("escuchar admin en %s: %w", s.cfg.AdminListen, err)
	}

	public := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}
	admin := &http.Server{
		Handler:           s.adminHandler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	errCh := make(chan error, 2)
	go func() {
		if err := public.Serve(mainListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		if err := admin.Serve(adminListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	s.logger.Info("p40la-mcp HTTP iniciado",
		"version", s.version,
		"listen", s.cfg.Listen,
		"admin_listen", s.cfg.AdminListen,
		"public_url", s.cfg.PublicURL,
		"writes_local", s.cfg.AllowWrites,
		"writes_remote", s.cfg.RemoteWrites,
		"rate_limit", s.cfg.RateLimit,
	)

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	_ = public.Shutdown(shutdownCtx)
	_ = admin.Shutdown(shutdownCtx)
	return s.audit.Close()
}

// Handler devuelve el router público (MCP + OAuth).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return s.mcp },
		&mcp.StreamableHTTPOptions{
			Logger: s.logger,
			// El servidor escucha en loopback detrás de Funnel: el Host público
			// es válido y lo valida allowedHost (ADR-006).
			DisableLocalhostProtection: true,
		},
	)
	mux.Handle("/mcp", s.requireBearer(s.allowedHost(s.rateLimit(s.limitBody(mcpHandler)))))
	mux.Handle("/", s.oauth.Handler())
	return s.securityHeaders(s.logRequests(mux))
}

func (s *Server) adminHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /admin/revoke", s.revoke)
	return s.securityHeaders(mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"version":       s.version,
		"uptime_s":      int(time.Since(s.started).Seconds()),
		"writes_local":  s.cfg.AllowWrites,
		"writes_remote": s.cfg.RemoteWrites,
	})
}

func (s *Server) revoke(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackRequest(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if err := s.oauth.RevokeAll(); err != nil {
		s.logger.Error("revocar tokens", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (s *Server) requireBearer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := bearerToken(r)
		if raw == "" {
			s.unauthorized(w)
			return
		}
		subject, ok := s.oauth.ValidateToken(r.Context(), raw)
		if !ok {
			s.unauthorized(w)
			return
		}
		ctx := context.WithValue(r.Context(), subjectKey, subject)
		ctx = context.WithValue(ctx, requestIDKey, newRequestID())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate",
		fmt.Sprintf(`Bearer resource_metadata="%s/.well-known/oauth-protected-resource"`, s.cfg.PublicURL))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             "invalid_token",
		"error_description": "se requiere un bearer token válido",
	})
}

// allowedHost acepta el Host público configurado y loopback. Reemplaza la
// protección anti-DNS-rebinding del SDK, que rechaza el Host de Funnel porque
// el listener está en 127.0.0.1 (ADR-006).
func (s *Server) allowedHost(next http.Handler) http.Handler {
	publicHost := ""
	if u, err := url.Parse(s.cfg.PublicURL); err == nil {
		publicHost = strings.ToLower(u.Hostname())
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := normalizeHost(r.Host)
		allowed := publicHost != "" && host == publicHost
		if !allowed && host == "localhost" {
			allowed = true
		}
		if ip := net.ParseIP(host); !allowed && ip != nil && ip.IsLoopback() {
			allowed = true
		}
		if !allowed {
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": "forbidden_host", "error_description": "host no permitido",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func normalizeHost(raw string) string {
	if host, _, err := net.SplitHostPort(raw); err == nil {
		return strings.ToLower(host)
	}
	return strings.ToLower(strings.Trim(raw, "[]"))
}

func (s *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := hashRaw(bearerToken(r))
		if key == "" {
			key = clientIP(r)
		}
		if allowed, retry := s.limiter.Allow(key); !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retry.Seconds())+1))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{
				"error": "too_many_requests", "error_description": "límite de uso alcanzado",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > maxMCPBody {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{
				"error": "payload_too_large", "error_description": "el cuerpo supera 1 MiB",
			})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxMCPBody)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Cache-Control", "no-store")
		if s.cfg.PublicURL != "" && len(s.cfg.PublicURL) > 5 && s.cfg.PublicURL[:5] == "https" {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Flush y Unwrap preservan el streaming SSE del transporte MCP.
func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		s.logger.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if len(header) <= 7 || !strings.EqualFold(header[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}

func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if idx := strings.IndexByte(forwarded, ','); idx > 0 {
			return strings.TrimSpace(forwarded[:idx])
		}
		return strings.TrimSpace(forwarded)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func hashRaw(raw string) string {
	if raw == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
