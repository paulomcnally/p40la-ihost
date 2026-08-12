package api

import (
	"context"
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

type contextKey string

const userContextKey contextKey = "user"

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
