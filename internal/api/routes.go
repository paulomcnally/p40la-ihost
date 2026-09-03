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
	mux.Handle("POST /api/system-settings/test-email", authMiddleware(http.HandlerFunc(handler.systemSettings.TestEmail)))
	mux.Handle("POST /api/system-settings/test-voice", authMiddleware(http.HandlerFunc(handler.systemSettings.TestVoice)))
	mux.Handle("DELETE /api/system-settings/voicemonkey", authMiddleware(http.HandlerFunc(handler.systemSettings.DeleteVoiceMonkey)))
	mux.Handle("DELETE /api/system-settings/smtp", authMiddleware(http.HandlerFunc(handler.systemSettings.DeleteSMTP)))

	// APIs de alertas (catálogo + toggles de canal)
	mux.Handle("GET /api/alerts", authMiddleware(http.HandlerFunc(handler.alerts.ListAlerts)))
	mux.Handle("PUT /api/alerts/{key}", authMiddleware(http.HandlerFunc(handler.alerts.UpdateAlert)))

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
	mux.Handle("POST /api/bills/{id}/pay", authMiddleware(http.HandlerFunc(handler.bill.PayBill)))
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

	// APIs de seguros de autos
	mux.Handle("GET /api/autos/{id}/services", authMiddleware(http.HandlerFunc(handler.autoService.ListAutoServices)))
	mux.Handle("POST /api/autos/{id}/services", authMiddleware(http.HandlerFunc(handler.autoService.CreateAutoService)))
	mux.Handle("DELETE /api/autos/{id}/services/{service_id}", authMiddleware(http.HandlerFunc(handler.autoService.DeleteAutoService)))
	mux.Handle("GET /api/autos/{id}/available-services", authMiddleware(http.HandlerFunc(handler.autoService.ListAvailableServices)))

	// APIs de categorías de instituciones
	mux.Handle("GET /api/institution-categories", authMiddleware(http.HandlerFunc(handler.institutionCategory.ListCategories)))
	mux.Handle("GET /api/institution-categories/{id}", authMiddleware(http.HandlerFunc(handler.institutionCategory.GetCategory)))
	mux.Handle("POST /api/institution-categories", authMiddleware(http.HandlerFunc(handler.institutionCategory.CreateCategory)))
	mux.Handle("PUT /api/institution-categories/{id}", authMiddleware(http.HandlerFunc(handler.institutionCategory.UpdateCategory)))
	mux.Handle("DELETE /api/institution-categories/{id}", authMiddleware(http.HandlerFunc(handler.institutionCategory.DeleteCategory)))

// APIs de notificaciones
	mux.Handle("GET /api/notifications", authMiddleware(http.HandlerFunc(handler.notification.ListNotifications)))
	mux.Handle("GET /api/notifications/{id}", authMiddleware(http.HandlerFunc(handler.notification.GetNotification)))
	mux.Handle("POST /api/notifications", authMiddleware(http.HandlerFunc(handler.notification.CreateNotification)))
	mux.Handle("PUT /api/notifications/{id}", authMiddleware(http.HandlerFunc(handler.notification.UpdateNotification)))
	mux.Handle("DELETE /api/notifications/{id}", authMiddleware(http.HandlerFunc(handler.notification.DeleteNotification)))

	// APIs de hijos (pensión alimenticia)
	mux.Handle("GET /api/children", authMiddleware(http.HandlerFunc(handler.child.ListChildren)))
	mux.Handle("GET /api/children/{id}", authMiddleware(http.HandlerFunc(handler.child.GetChild)))
	mux.Handle("POST /api/children", authMiddleware(http.HandlerFunc(handler.child.CreateChild)))
	mux.Handle("PUT /api/children/{id}", authMiddleware(http.HandlerFunc(handler.child.UpdateChild)))
	mux.Handle("DELETE /api/children/{id}", authMiddleware(http.HandlerFunc(handler.child.DeleteChild)))

// APIs de salarios (pensión alimenticia)
	mux.Handle("GET /api/salaries", authMiddleware(http.HandlerFunc(handler.salary.ListSalaries)))
	mux.Handle("GET /api/salaries/{id}", authMiddleware(http.HandlerFunc(handler.salary.GetSalary)))
	mux.Handle("POST /api/salaries", authMiddleware(http.HandlerFunc(handler.salary.CreateSalary)))
	mux.Handle("PUT /api/salaries/{id}", authMiddleware(http.HandlerFunc(handler.salary.UpdateSalary)))
	mux.Handle("DELETE /api/salaries/{id}", authMiddleware(http.HandlerFunc(handler.salary.DeleteSalary)))

	// APIs de categorías de pensión alimenticia
	mux.Handle("GET /api/pension-categories", authMiddleware(http.HandlerFunc(handler.pensionCategory.ListCategories)))
	mux.Handle("GET /api/pension-categories/{id}", authMiddleware(http.HandlerFunc(handler.pensionCategory.GetCategory)))
	mux.Handle("POST /api/pension-categories", authMiddleware(http.HandlerFunc(handler.pensionCategory.CreateCategory)))
	mux.Handle("PUT /api/pension-categories/{id}", authMiddleware(http.HandlerFunc(handler.pensionCategory.UpdateCategory)))
	mux.Handle("DELETE /api/pension-categories/{id}", authMiddleware(http.HandlerFunc(handler.pensionCategory.DeleteCategory)))

	// APIs de registros de manutención (SPEC-049)
	mux.Handle("GET /api/pension/records", authMiddleware(http.HandlerFunc(handler.supportRecord.ListRecords)))
	mux.Handle("GET /api/pension/records/{id}", authMiddleware(http.HandlerFunc(handler.supportRecord.GetRecord)))
	mux.Handle("POST /api/pension/records", authMiddleware(http.HandlerFunc(handler.supportRecord.CreateRecord)))
	mux.Handle("PUT /api/pension/records/{id}", authMiddleware(http.HandlerFunc(handler.supportRecord.UpdateRecord)))
	mux.Handle("POST /api/pension/records/{id}/mark-paid", authMiddleware(http.HandlerFunc(handler.supportRecord.MarkPaid)))
	mux.Handle("POST /api/pension/records/{id}/mark-pending", authMiddleware(http.HandlerFunc(handler.supportRecord.MarkPending)))
	mux.Handle("POST /api/pension/records/{id}/mark-rejected", authMiddleware(http.HandlerFunc(handler.supportRecord.MarkRejected)))
	mux.Handle("POST /api/pension/records/{id}/upload-proof", authMiddleware(http.HandlerFunc(handler.supportRecord.UploadProof)))
	mux.Handle("GET /api/pension/records/{id}/proof", authMiddleware(http.HandlerFunc(handler.supportRecord.DownloadProof)))

	// APIs de pagos de salario (SPEC-049)
	mux.Handle("GET /api/pension/salary-payments", authMiddleware(http.HandlerFunc(handler.salaryPayment.ListSalaryPayments)))
	mux.Handle("GET /api/pension/salary-payments/{id}", authMiddleware(http.HandlerFunc(handler.salaryPayment.GetSalaryPayment)))
	mux.Handle("POST /api/pension/salary-payments/{id}/mark-received", authMiddleware(http.HandlerFunc(handler.salaryPayment.MarkSalaryReceived)))
	mux.Handle("POST /api/pension/salary-payments/{id}/mark-pending", authMiddleware(http.HandlerFunc(handler.salaryPayment.MarkSalaryPending)))

	// APIs de cierre de mes (SPEC-049)
	mux.Handle("GET /api/pension/closing/{year}/{month}", authMiddleware(http.HandlerFunc(handler.monthClosing.GetClosingStatus)))
	mux.Handle("POST /api/pension/closing/{year}/{month}", authMiddleware(http.HandlerFunc(handler.monthClosing.CloseMonth)))
	mux.Handle("DELETE /api/pension/closing/{year}/{month}", authMiddleware(http.HandlerFunc(handler.monthClosing.ReopenMonth)))

	// APIs de configs de pensión (SPEC-051)
	mux.Handle("GET /api/pension/configs", authMiddleware(http.HandlerFunc(handler.config.ListConfigs)))
	mux.Handle("GET /api/pension/configs/{id}", authMiddleware(http.HandlerFunc(handler.config.GetConfig)))
	mux.Handle("POST /api/pension/configs", authMiddleware(http.HandlerFunc(handler.config.CreateConfig)))
	mux.Handle("PUT /api/pension/configs/{id}", authMiddleware(http.HandlerFunc(handler.config.UpdateConfig)))
	mux.Handle("DELETE /api/pension/configs/{id}", authMiddleware(http.HandlerFunc(handler.config.DeleteConfig)))

	// Generación mensual de pensión (SPEC-051)
	mux.Handle("POST /api/pension/generate", authMiddleware(http.HandlerFunc(handler.pensionDashboard.GenerateMonth)))

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
