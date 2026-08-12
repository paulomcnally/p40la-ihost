package api

import (
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// BuildRouter configura y devuelve el enrutador principal.
func BuildRouter(handler *Handler, auth *services.AuthService, staticDir string) http.Handler {
	mux := http.NewServeMux()

	// APIs
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("GET /api/setup-status", handler.SetupStatus)
	mux.HandleFunc("POST /api/setup", handler.Setup)
	mux.HandleFunc("POST /api/login", handler.Login)
	mux.HandleFunc("POST /api/logout", handler.Logout)
	mux.Handle("GET /api/me", AuthMiddleware(auth)(http.HandlerFunc(handler.Me)))

	// Páginas HTML
	mux.HandleFunc("GET /setup", setupPageHandler(auth, staticDir))
	mux.HandleFunc("GET /login", loginPageHandler(auth, staticDir))
	mux.HandleFunc("GET /dashboard", dashboardPageHandler(auth, staticDir))

	// Assets estáticos
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("GET /css/", http.StripPrefix("/css/", http.FileServer(http.Dir(staticDir+"/css"))))
	mux.Handle("GET /js/", http.StripPrefix("/js/", http.FileServer(http.Dir(staticDir+"/js"))))

	// Raíz con lógica de redirección
	mux.Handle("GET /{$}", EntryRedirectMiddleware(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Este handler nunca se ejecuta porque el middleware redirige siempre.
		http.NotFound(w, r)
	})))

	// Cualquier otra ruta sirve archivos estáticos (index.html fallback opcional)
	mux.Handle("GET /", fs)

	return mux
}

func setupPageHandler(auth *services.AuthService, staticDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setup, err := auth.IsSetupComplete(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if setup {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		http.ServeFile(w, r, staticDir+"/setup.html")
	}
}

func loginPageHandler(auth *services.AuthService, staticDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setup, err := auth.IsSetupComplete(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !setup {
			http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
			return
		}
		if isAuthenticated(r, auth) {
			http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
			return
		}
		http.ServeFile(w, r, staticDir+"/login.html")
	}
}

func dashboardPageHandler(auth *services.AuthService, staticDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAuthenticated(r, auth) {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		http.ServeFile(w, r, staticDir+"/dashboard.html")
	}
}

func isAuthenticated(r *http.Request, auth *services.AuthService) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}
	user, err := auth.ValidateSession(r.Context(), cookie.Value)
	return err == nil && user != nil
}
