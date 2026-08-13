package api

import (
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// BuildRouter configura y devuelve el enrutador principal.
func BuildRouter(handler *Handler, auth *services.AuthService, staticDir string) http.Handler {
	mux := http.NewServeMux()

	authMiddleware := AuthMiddleware(auth)

	// APIs públicas
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("GET /api/setup-status", handler.SetupStatus)
	mux.HandleFunc("POST /api/setup", handler.Setup)
	mux.HandleFunc("POST /api/login", handler.Login)
	mux.HandleFunc("POST /api/logout", handler.Logout)
	mux.Handle("GET /api/me", authMiddleware(http.HandlerFunc(handler.Me)))

	// APIs de configuraciones
	mux.Handle("GET /api/settings", authMiddleware(http.HandlerFunc(handler.settings.GetSettings)))
	mux.Handle("POST /api/settings/language", authMiddleware(http.HandlerFunc(handler.settings.SetLanguage)))

	// APIs de monedas
	mux.Handle("GET /api/currencies", authMiddleware(http.HandlerFunc(handler.currency.ListCurrencies)))
	mux.Handle("POST /api/currencies", authMiddleware(http.HandlerFunc(handler.currency.CreateCurrency)))
	mux.Handle("PUT /api/currencies/{id}", authMiddleware(http.HandlerFunc(handler.currency.UpdateCurrency)))
	mux.Handle("DELETE /api/currencies/{id}", authMiddleware(http.HandlerFunc(handler.currency.DeleteCurrency)))

	// APIs de hogares
	mux.Handle("GET /api/homes", authMiddleware(http.HandlerFunc(handler.home.ListHomes)))
	mux.Handle("GET /api/homes/count", authMiddleware(http.HandlerFunc(handler.home.CountHomes)))
	mux.Handle("GET /api/homes/{id}", authMiddleware(http.HandlerFunc(handler.home.GetHome)))
	mux.Handle("POST /api/homes", authMiddleware(http.HandlerFunc(handler.home.CreateHome)))
	mux.Handle("PUT /api/homes/{id}", authMiddleware(http.HandlerFunc(handler.home.UpdateHome)))
	mux.Handle("DELETE /api/homes/{id}", authMiddleware(http.HandlerFunc(handler.home.DeleteHome)))

	// APIs de servicios
	mux.Handle("GET /api/services", authMiddleware(http.HandlerFunc(handler.service.ListServices)))
	mux.Handle("GET /api/services/{id}", authMiddleware(http.HandlerFunc(handler.service.GetService)))
	mux.Handle("POST /api/services", authMiddleware(http.HandlerFunc(handler.service.CreateService)))
	mux.Handle("PUT /api/services/{id}", authMiddleware(http.HandlerFunc(handler.service.UpdateService)))
	mux.Handle("DELETE /api/services/{id}", authMiddleware(http.HandlerFunc(handler.service.DeleteService)))

	// APIs de facturas
	mux.Handle("GET /api/services/{service_id}/bills", authMiddleware(http.HandlerFunc(handler.bill.ListBills)))
	mux.Handle("GET /api/bills/{id}", authMiddleware(http.HandlerFunc(handler.bill.GetBill)))
	mux.Handle("POST /api/bills", authMiddleware(http.HandlerFunc(handler.bill.CreateBill)))
	mux.Handle("PUT /api/bills/{id}", authMiddleware(http.HandlerFunc(handler.bill.UpdateBill)))
	mux.Handle("DELETE /api/bills/{id}", authMiddleware(http.HandlerFunc(handler.bill.DeleteBill)))

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
