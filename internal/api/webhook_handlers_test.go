package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newWebhookTestHandlers(t *testing.T) (*WebhookHandlers, *services.ServiceService, *services.WebhookService, *services.HomeService, *services.CurrencyService, *services.SystemSettingsService) {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	systemSettingsStorage := storage.NewSystemSettingsStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	homeStorage := storage.NewHomeStorage(database)
	serviceStorage := storage.NewServiceStorage(database)
	billStorage := storage.NewBillStorage(database)

	homeSvc := services.NewHomeService(homeStorage)
	currencySvc := services.NewCurrencyService(currencyStorage)
	serviceSvc := services.NewServiceService(serviceStorage, homeStorage, currencyStorage, billStorage)
	settingsSvc := services.NewSystemSettingsService(systemSettingsStorage)
	webhookSvc := services.NewWebhookService(systemSettingsStorage, settingsSvc, serviceStorage, billStorage)

	if err := serviceSvc.EnsureWebhookUUIDs(ctx); err != nil {
		t.Fatalf("backfill uuid: %v", err)
	}
	if err := settingsSvc.SetWebhookEnabled(ctx, true); err != nil {
		t.Fatalf("habilitar webhooks: %v", err)
	}

	return NewWebhookHandlers(webhookSvc, serviceSvc), serviceSvc, webhookSvc, homeSvc, currencySvc, settingsSvc
}

func TestWebhookUpsertBillHandler(t *testing.T) {
	h, serviceSvc, webhookSvc, homeSvc, currencySvc, _ := newWebhookTestHandlers(t)
	ctx := context.Background()

	home, err := homeSvc.Create(ctx, "Casa Webhook", "")
	if err != nil {
		t.Fatalf("crear hogar: %v", err)
	}
	currencies, _ := currencySvc.List(ctx)
	svc, err := serviceSvc.Create(ctx, &models.Service{
		HomeID:          home.ID,
		Name:            "Internet",
		Institution:     "Claro",
		CurrencyID:      currencies[0].ID,
		Frequency:       services.FrequencyMonthly,
		SuggestedAmount: 45,
		Active:          true,
		IconKey:         "internet",
	})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	if svc.WebhookUUID == "" {
		t.Fatal("el servicio debe tener webhook_uuid")
	}

	key, _ := webhookSvc.GetOrCreateWebhookKey(ctx)

	payload, _ := json.Marshal(models.WebhookBillPayload{
		Year: 2025, Month: 9, Amount: 99, Status: "paid", PaidAt: "2025-09-05", PaymentReference: "REF-001",
	})

	// El handler real está protegido por WebhookAuthMiddleware (api_key global).
	handler := WebhookAuthMiddleware(webhookSvc)(http.HandlerFunc(h.UpsertBill))

	// Sin api_key → 401
	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+svc.WebhookUUID, bytes.NewReader(payload))
	req.SetPathValue("uuid", svc.WebhookUUID)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("sin api_key esperaba 401, got %d", rr.Code)
	}

	// Con api_key incorrecta → 401
	req = httptest.NewRequest(http.MethodPost, "/webhooks/"+svc.WebhookUUID, bytes.NewReader(payload))
	req.Header.Set("X-Webhook-Key", "wrong")
	req.SetPathValue("uuid", svc.WebhookUUID)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("api_key incorrecta esperaba 401, got %d", rr.Code)
	}

	// Con api_key correcta → 200 y created=true
	req = httptest.NewRequest(http.MethodPost, "/webhooks/"+svc.WebhookUUID, bytes.NewReader(payload))
	req.Header.Set("X-Webhook-Key", key)
	req.SetPathValue("uuid", svc.WebhookUUID)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("con api_key esperaba 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var res models.WebhookResult
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("decodificar respuesta: %v", err)
	}
	if !res.Created {
		t.Error("se esperaba created=true")
	}
	if res.Bill.Status != "paid" {
		t.Errorf("estado esperado paid, got %s", res.Bill.Status)
	}
	if res.Bill.PaymentReference != "REF-001" {
		t.Errorf("referencia esperada REF-001, got %q", res.Bill.PaymentReference)
	}

	// uuid inexistente → 404
	req = httptest.NewRequest(http.MethodPost, "/webhooks/no-existe", bytes.NewReader(payload))
	req.Header.Set("X-Webhook-Key", key)
	req.SetPathValue("uuid", "no-existe")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("uuid inexistente esperaba 404, got %d", rr.Code)
	}

	// Body inválido → 400
	req = httptest.NewRequest(http.MethodPost, "/webhooks/"+svc.WebhookUUID, bytes.NewReader([]byte(`{"year":2025,"month":13}`)))
	req.Header.Set("X-Webhook-Key", key)
	req.SetPathValue("uuid", svc.WebhookUUID)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("body inválido esperaba 400, got %d", rr.Code)
	}
}

func TestWebhookDisabledReturns403(t *testing.T) {
	h, serviceSvc, webhookSvc, homeSvc, currencySvc, settingsSvc := newWebhookTestHandlers(t)
	ctx := context.Background()

	home, err := homeSvc.Create(ctx, "Casa Webhook", "")
	if err != nil {
		t.Fatalf("crear hogar: %v", err)
	}
	currencies, _ := currencySvc.List(ctx)
	svc, err := serviceSvc.Create(ctx, &models.Service{
		HomeID:          home.ID,
		Name:            "Internet",
		Institution:     "Claro",
		CurrencyID:      currencies[0].ID,
		Frequency:       services.FrequencyMonthly,
		SuggestedAmount: 45,
		Active:          true,
		IconKey:         "internet",
	})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}

	key, _ := webhookSvc.GetOrCreateWebhookKey(ctx)

	// Deshabilitar la feature.
	if err := settingsSvc.SetWebhookEnabled(ctx, false); err != nil {
		t.Fatalf("deshabilitar webhooks: %v", err)
	}

	payload, _ := json.Marshal(models.WebhookBillPayload{Year: 2025, Month: 9, Amount: 99})
	handler := WebhookAuthMiddleware(webhookSvc)(http.HandlerFunc(h.UpsertBill))
	req := httptest.NewRequest(http.MethodPost, "/webhooks/"+svc.WebhookUUID, bytes.NewReader(payload))
	req.Header.Set("X-Webhook-Key", key)
	req.SetPathValue("uuid", svc.WebhookUUID)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("feature deshabilitada esperaba 403, got %d", rr.Code)
	}
}

func TestWebhookKeyHandlers(t *testing.T) {
	h, _, webhookSvc, _, _, _ := newWebhookTestHandlers(t)
	ctx := context.Background()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/webhook/key", nil)
	h.GetWebhookKey(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET key esperaba 200, got %d", rr.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decodificar: %v", err)
	}
	if len(resp["api_key"]) != 64 {
		t.Errorf("api_key esperada de 64 chars, got %q", resp["api_key"])
	}

	oldKey, _ := webhookSvc.GetOrCreateWebhookKey(ctx)

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/webhook/key/regenerate", nil)
	h.RegenerateWebhookKey(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("regenerate esperaba 200, got %d", rr.Code)
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decodificar regenerate: %v", err)
	}
	if resp["api_key"] == oldKey {
		t.Error("la api_key regenerada debe ser distinta")
	}
}
