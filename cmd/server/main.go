package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/paulomcnally/p40la-ihost/internal/analyzers/all"
	"github.com/paulomcnally/p40la-ihost/internal/api"
	"github.com/paulomcnally/p40la-ihost/internal/config"
	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func main() {
	var (
		healthcheck = flag.Bool("healthcheck", false, "Verificar estado de salud del servidor")
		versionFlag = flag.Bool("version", false, "Mostrar versión")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	if *versionFlag {
		fmt.Println(cfg.Version)
		os.Exit(0)
	}

	initLogger(cfg.LogLevel)

	if *healthcheck {
		if err := runHealthcheck(cfg.Port); err != nil {
			slog.Error("healthcheck falló", "error", err)
			os.Exit(1)
		}
		slog.Info("healthcheck ok")
		os.Exit(0)
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		slog.Error("crear directorio de datos", "error", err)
		os.Exit(1)
	}

	database, err := db.OpenDB(
		filepath.Join(cfg.DataDir, "app.db"),
		"./migrations",
	)
	if err != nil {
		slog.Error("abrir base de datos", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	userStorage := storage.NewUserStorage(database)
	settingsStorage := storage.NewSettingsStorage(database)
	systemSettingsStorage := storage.NewSystemSettingsStorage(database)
	alertStorage := storage.NewAlertStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	homeStorage := storage.NewHomeStorage(database)
	serviceStorage := storage.NewServiceStorage(database)
	billStorage := storage.NewBillStorage(database)
	institutionStorage := storage.NewInstitutionStorage(database)
	autoStorage := storage.NewAutoStorage(database)
	autoServiceStorage := storage.NewAutoServiceStorage(database)
	institutionCategoryStorage := storage.NewInstitutionCategoryStorage(database)
	childStorage := storage.NewChildStorage(database)

	authService := services.NewAuthService(userStorage, settingsStorage, cfg)
	appSettingsService := services.NewAppSettingsService(settingsStorage)
	systemSettingsService := services.NewSystemSettingsService(systemSettingsStorage)
	emailService := services.NewEmailService(systemSettingsService)
	voiceMonkeyService := services.NewVoiceMonkeyService(systemSettingsService)
	alertService := services.NewAlertService(alertStorage)
	currencyService := services.NewCurrencyService(currencyStorage)
	homeService := services.NewHomeService(homeStorage)
	serviceService := services.NewServiceService(serviceStorage, homeStorage, currencyStorage, billStorage)
	billService := services.NewBillService(billStorage, serviceStorage)
	institutionService := services.NewInstitutionService(institutionStorage)
	documentService := services.NewDocumentService(serviceStorage, billStorage, institutionStorage)
	autoService := services.NewAutoService(autoStorage)
	autoInsuranceService := services.NewAutoServiceService(autoServiceStorage)
	institutionCategoryService := services.NewInstitutionCategoryService(institutionCategoryStorage)
	childService := services.NewChildService(childStorage)

	// Seed del catálogo de alertas (idempotente, no borra toggles del usuario).
	if err := alertService.Seed(context.Background()); err != nil {
		slog.Error("seed de alertas", "error", err)
	}

	billingScheduler := services.NewBillingScheduler(serviceStorage, billStorage, systemSettingsService, emailService, currencyStorage, alertService, voiceMonkeyService)
	billingScheduler.Start()
	defer billingScheduler.Stop()

	alertScheduler := services.NewAlertScheduler(autoStorage, autoServiceStorage, emailService, systemSettingsService, alertService, voiceMonkeyService)
	alertScheduler.Start()
	defer alertScheduler.Stop()

	billSummaryScheduler := services.NewBillSummaryScheduler(billStorage, emailService, systemSettingsService, alertService, voiceMonkeyService)
	billSummaryScheduler.Start()
	defer billSummaryScheduler.Stop()

	settingsHandlers := api.NewSettingsHandlers(appSettingsService)
	systemSettingsHandlers := api.NewSystemSettingsHandlers(systemSettingsService, emailService, voiceMonkeyService)
	alertsHandlers := api.NewAlertsHandlers(alertService, systemSettingsService)
	currencyHandlers := api.NewCurrencyHandlers(currencyService)
	homeHandlers := api.NewHomeHandlers(homeService)
	serviceHandlers := api.NewServiceHandlers(serviceService, homeService, institutionStorage)
	billHandlers := api.NewBillHandlers(billService)
	institutionHandlers := api.NewInstitutionHandlers(institutionService, documentService)
	documentHandlers := api.NewDocumentHandlers(documentService)
	autoHandlers := api.NewAutoHandlers(autoService)
	autoServiceHandlers := api.NewAutoServiceHandlers(autoInsuranceService)
	institutionCategoryHandlers := api.NewInstitutionCategoryHandlers(institutionCategoryService)
	childHandlers := api.NewChildHandlers(childService)

	handler := api.NewHandler(authService, settingsHandlers, systemSettingsHandlers, alertsHandlers, currencyHandlers, homeHandlers, serviceHandlers, billHandlers, institutionHandlers, documentHandlers, autoHandlers, autoServiceHandlers, institutionCategoryHandlers, childHandlers)

	router := api.BuildRouter(handler, authService, "./public")

	addr := ":" + cfg.Port
	slog.Info("iniciando servidor", "addr", addr, "version", cfg.Version)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("servidor detenido", "error", err)
		os.Exit(1)
	}
}

func initLogger(level slog.Level) {
	opts := &slog.HandlerOptions{Level: level}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, opts)))
}

func runHealthcheck(port string) error {
	resp, err := http.Get("http://localhost:" + port + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
