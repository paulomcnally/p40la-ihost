package api

import (
	"net/http"
	"os"
	"path/filepath"

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

	// APIs de system settings
	mux.Handle("GET /api/system-settings", authMiddleware(http.HandlerFunc(handler.systemSettings.GetSystemSettings)))
	mux.Handle("PUT /api/system-settings", authMiddleware(http.HandlerFunc(handler.systemSettings.UpdateSystemSettings)))

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

	// APIs de documentos
	mux.Handle("POST /api/services/{service_id}/bills/upload", authMiddleware(http.HandlerFunc(handler.document.UploadAndAnalyze)))
	mux.Handle("POST /api/services/{service_id}/bills/from-extracted", authMiddleware(http.HandlerFunc(handler.document.CreateBillFromExtracted)))

	// APIs de instituciones
	mux.Handle("GET /api/institutions", authMiddleware(http.HandlerFunc(handler.institution.ListInstitutions)))
	mux.Handle("GET /api/institutions/{id}", authMiddleware(http.HandlerFunc(handler.institution.GetInstitution)))
	mux.Handle("POST /api/institutions", authMiddleware(http.HandlerFunc(handler.institution.CreateInstitution)))
	mux.Handle("PUT /api/institutions/{id}", authMiddleware(http.HandlerFunc(handler.institution.UpdateInstitution)))
	mux.Handle("DELETE /api/institutions/{id}", authMiddleware(http.HandlerFunc(handler.institution.DeleteInstitution)))
	mux.Handle("PUT /api/institutions/{id}/analyzers", authMiddleware(http.HandlerFunc(handler.institution.SetAnalyzers)))
	mux.Handle("GET /api/institutions/{id}/analyzers", authMiddleware(http.HandlerFunc(handler.institution.GetAnalyzers)))

	// APIs de analyzers
	mux.Handle("GET /api/analyzers", authMiddleware(http.HandlerFunc(handler.institution.ListAnalyzers)))
	mux.Handle("GET /api/services/{id}/analyzer-options", authMiddleware(http.HandlerFunc(handler.institution.GetAnalyzerOptions)))

	// APIs de autos
	mux.Handle("GET /api/autos", authMiddleware(http.HandlerFunc(handler.auto.ListAutos)))
	mux.Handle("GET /api/autos/{id}", authMiddleware(http.HandlerFunc(handler.auto.GetAuto)))
	mux.Handle("POST /api/autos", authMiddleware(http.HandlerFunc(handler.auto.CreateAuto)))
	mux.Handle("PUT /api/autos/{id}", authMiddleware(http.HandlerFunc(handler.auto.UpdateAuto)))
	mux.Handle("DELETE /api/autos/{id}", authMiddleware(http.HandlerFunc(handler.auto.DeleteAuto)))

	// Assets estáticos (JS/CSS bundles del build de Vite)
	assetsFS := http.FileServer(http.Dir(filepath.Join(staticDir, "assets")))
	mux.Handle("GET /assets/{file...}", http.StripPrefix("/assets/", assetsFS))

	// i18n JSON files
	i18nFS := http.FileServer(http.Dir(filepath.Join(staticDir, "i18n")))
	mux.Handle("GET /i18n/{file...}", http.StripPrefix("/i18n/", i18nFS))

	// SPA fallback — sirve index.html para cualquier ruta no-API no-asset
	mux.Handle("GET /{path...}", spaHandler(staticDir))

	return mux
}

func spaHandler(staticDir string) http.HandlerFunc {
	indexFile := filepath.Join(staticDir, "index.html")

	return func(w http.ResponseWriter, r *http.Request) {
		staticPath := filepath.Join(staticDir, r.URL.Path)
		if info, err := os.Stat(staticPath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, staticPath)
			return
		}
		http.ServeFile(w, r, indexFile)
	}
}
