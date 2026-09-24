package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

const testAutomationKey = "automation-key-para-tests-123456"

// newSyncTestSetup levanta el stack de servicios contra una DB en memoria y un
// fake server de automation. Devuelve handlers, servicios, el server y la key.
func newSyncTestSetup(t *testing.T, automationHandler http.HandlerFunc) (*ServiceHandlers, *services.ServiceService, *services.CurrencyService, *services.SystemSettingsService, *services.HomeService, *httptest.Server, string) {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	systemSettingsStorage := storage.NewSystemSettingsStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	homeStorage := storage.NewHomeStorage(database)
	serviceStorage := storage.NewServiceStorage(database)
	billStorage := storage.NewBillStorage(database)
	institutionStorage := storage.NewInstitutionStorage(database)

	homeSvc := services.NewHomeService(homeStorage)
	currencySvc := services.NewCurrencyService(currencyStorage)
	serviceSvc := services.NewServiceService(serviceStorage, homeStorage, currencyStorage, billStorage)
	settingsSvc := services.NewSystemSettingsService(systemSettingsStorage)

	automationClient := services.NewAutomationClient(settingsSvc)
	handlers := NewServiceHandlers(serviceSvc, homeSvc, institutionStorage, automationClient)

	fake := httptest.NewServer(http.HandlerFunc(automationHandler))
	t.Cleanup(fake.Close)

	return handlers, serviceSvc, currencySvc, settingsSvc, homeSvc, fake, testAutomationKey
}

// syncCreateService crea un servicio de prueba con las monedas de seed y un
// automation_account_id opcional.
func syncCreateService(t *testing.T, serviceSvc *services.ServiceService, currencySvc *services.CurrencyService, homeSvc *services.HomeService, automationAccountID *int64) (*models.Service, *models.Home) {
	t.Helper()
	ctx := context.Background()
	home, err := homeSvc.Create(ctx, "Casa Sync", "")
	if err != nil {
		t.Fatalf("crear hogar: %v", err)
	}
	currencies, err := currencySvc.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("monedas de seed: %v", err)
	}
	svc, err := serviceSvc.Create(ctx, &models.Service{
		HomeID:              home.ID,
		Name:                "Internet Sync",
		CurrencyID:          currencies[0].ID,
		Frequency:           "monthly",
		SuggestedAmount:     100,
		Active:              true,
		IconKey:             "internet",
		BillingType:         "variable",
		AutomationAccountID: automationAccountID,
	})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return svc, home
}

func TestSyncServiceHandler(t *testing.T) {
	ctx := context.Background()

	// Fake automation: exige la api_key y responde {delivered, failed}.
	fakeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Webhook-Key") != testAutomationKey {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized", "message": "api_key de webhook inválida"})
			return
		}
		if r.URL.Path != "/api/accounts/7/webhook:run" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not_found", "message": "cuenta inexistente"})
			return
		}
		json.NewEncoder(w).Encode(map[string]int{"delivered": 2, "failed": 1})
	})
	handlers, serviceSvc, currencySvc, settingsSvc, homeSvc, fake, apiKey := newSyncTestSetup(t, fakeHandler)

	accountID := int64(7)
	svc, _ := syncCreateService(t, serviceSvc, currencySvc, homeSvc, &accountID)

	// Automation sin configurar → 400
	if err := settingsSvc.SetAutomationConfig(ctx, "", ""); err != nil {
		t.Fatalf("limpiar config automation: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/services/"+strconv.FormatInt(svc.ID, 10)+"/sync", nil)
	req.SetPathValue("id", strconv.FormatInt(svc.ID, 10))
	rec := httptest.NewRecorder()
	handlers.SyncService(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("sin config automation: status %d, want 400; body %s", rec.Code, rec.Body.String())
	}

	// Configurado y válido → 200 {delivered: 2, failed: 1}
	if err := settingsSvc.SetAutomationConfig(ctx, fake.URL, apiKey); err != nil {
		t.Fatalf("configurar automation: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/services/"+strconv.FormatInt(svc.ID, 10)+"/sync", nil)
	req.SetPathValue("id", strconv.FormatInt(svc.ID, 10))
	rec = httptest.NewRecorder()
	handlers.SyncService(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("sync válido: status %d, want 200; body %s", rec.Code, rec.Body.String())
	}
	var result map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("parsear respuesta: %v", err)
	}
	if result["delivered"] != 2 || result["failed"] != 1 {
		t.Errorf("delivered/failed = %v, want 2/1", result)
	}
}

func TestSyncServiceHandlerErrors(t *testing.T) {
	ctx := context.Background()

	// Automation que devuelve 401 (api_key inválida) para probar propagación.
	fakeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized", "message": "api_key de webhook inválida"})
	})
	handlers, serviceSvc, currencySvc, settingsSvc, homeSvc, fake, apiKey := newSyncTestSetup(t, fakeHandler)
	if err := settingsSvc.SetAutomationConfig(ctx, fake.URL, apiKey); err != nil {
		t.Fatalf("configurar automation: %v", err)
	}

	// Servicio inexistente → 404
	req := httptest.NewRequest(http.MethodPost, "/api/services/99999/sync", nil)
	req.SetPathValue("id", "99999")
	rec := httptest.NewRecorder()
	handlers.SyncService(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("servicio inexistente: status %d, want 404; body %s", rec.Code, rec.Body.String())
	}

	// Servicio sin automation_account_id → 400
	svc, _ := syncCreateService(t, serviceSvc, currencySvc, homeSvc, nil)
	req = httptest.NewRequest(http.MethodPost, "/api/services/"+strconv.FormatInt(svc.ID, 10)+"/sync", nil)
	req.SetPathValue("id", strconv.FormatInt(svc.ID, 10))
	rec = httptest.NewRecorder()
	handlers.SyncService(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("sin automation_account_id: status %d, want 400; body %s", rec.Code, rec.Body.String())
	}

	// Automation devuelve 401 → se propaga como 401
	accountID := int64(7)
	svc2, _ := syncCreateService(t, serviceSvc, currencySvc, homeSvc, &accountID)
	req = httptest.NewRequest(http.MethodPost, "/api/services/"+strconv.FormatInt(svc2.ID, 10)+"/sync", nil)
	req.SetPathValue("id", strconv.FormatInt(svc2.ID, 10))
	rec = httptest.NewRecorder()
	handlers.SyncService(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("automation 401: status %d, want 401; body %s", rec.Code, rec.Body.String())
	}
}

func TestSyncServiceHandlerUnreachable(t *testing.T) {
	ctx := context.Background()
	// Server que se cierra al arrancar → automation inalcanzable → 502.
	fakeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handlers, serviceSvc, currencySvc, settingsSvc, homeSvc, fake, _ := newSyncTestSetup(t, fakeHandler)
	unreachableURL := fake.URL
	fake.Close() // cortar el server → conexión rechazada
	if err := settingsSvc.SetAutomationConfig(ctx, unreachableURL, "cualquier-key"); err != nil {
		t.Fatalf("configurar automation: %v", err)
	}

	accountID := int64(7)
	svc, _ := syncCreateService(t, serviceSvc, currencySvc, homeSvc, &accountID)

	req := httptest.NewRequest(http.MethodPost, "/api/services/"+strconv.FormatInt(svc.ID, 10)+"/sync", nil)
	req.SetPathValue("id", strconv.FormatInt(svc.ID, 10))
	rec := httptest.NewRecorder()
	handlers.SyncService(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("automation inalcanzable: status %d, want 502; body %s", rec.Code, rec.Body.String())
	}
}
