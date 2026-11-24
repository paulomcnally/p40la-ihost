package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type contextKey string

const (
	userContextKey       contextKey = "user"
	authSourceContextKey contextKey = "auth_source"
)

// WebhookAuthMiddleware valida la api_key global de webhooks (header
// X-Webhook-Key) y, si es válida, deja pasar la solicitud. SPEC-069.
func WebhookAuthMiddleware(webhooks *services.WebhookService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := r.Header.Get("X-Webhook-Key")
			if provided == "" || !webhooks.ValidateWebhookKey(r.Context(), provided) {
				respondError(w, http.StatusUnauthorized, "unauthorized", "api_key de webhook inválida")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SessionOrAPIKeyMiddleware acepta una API key global (SPEC-145) o, si no hay
// key, una sesión normal. La key tiene todos los permisos y se identifica con
// Authorization: Bearer o X-API-Key.
func SessionOrAPIKeyMiddleware(auth *services.AuthService, apiKeys *services.APIKeyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if provided := apiKeyFromRequest(r); provided != "" {
				if apiKeys == nil {
					respondError(w, http.StatusUnauthorized, "auth_failed", "API key inválida")
					return
				}
				valid, err := apiKeys.Validate(r.Context(), provided)
				if err != nil || !valid {
					respondError(w, http.StatusUnauthorized, "auth_failed", "API key inválida")
					return
				}
				ctx := context.WithValue(r.Context(), userContextKey, &models.User{Email: "api-key"})
				ctx = context.WithValue(ctx, authSourceContextKey, models.BillSourceMCP)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			AuthMiddleware(auth)(next).ServeHTTP(w, r)
		})
	}
}

func apiKeyFromRequest(r *http.Request) string {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(authorization) >= 7 && strings.EqualFold(authorization[:7], "bearer ") {
		return strings.TrimSpace(authorization[7:])
	}
	return strings.TrimSpace(r.Header.Get("X-API-Key"))
}

// requestBillSource devuelve el origen de auditoría. Una petición autenticada
// por API key siempre es mcp; una sesión puede declarar dashboard o mcp.
func requestBillSource(r *http.Request) string {
	if source, ok := r.Context().Value(authSourceContextKey).(string); ok && source != "" {
		return source
	}
	return r.Header.Get("X-P40LA-Source")
}

// AuthMiddleware valida la sesión y, si es válida, inyecta el usuario en el contexto.
func AuthMiddleware(auth *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := auth.ValidateSession(r.Context(), cookie.Value)
			if err != nil || user == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// userFromContext extrae el usuario autenticado del contexto.
func userFromContext(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userContextKey).(*models.User)
	return user, ok
}

// EntryRedirectMiddleware redirige la raíz según el estado de setup y sesión.
func EntryRedirectMiddleware(auth *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				next.ServeHTTP(w, r)
				return
			}

			setup, err := auth.IsSetupComplete(r.Context())
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if !setup {
				http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
				return
			}

			cookie, err := r.Cookie("session")
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}
			user, err := auth.ValidateSession(r.Context(), cookie.Value)
			if err != nil || user == nil {
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}

			http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
		})
	}
}
