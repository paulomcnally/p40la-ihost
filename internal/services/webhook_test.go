package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newTestWebhook(t *testing.T) (*WebhookService, *ServiceService, *BillService) {
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
	historyStorage := storage.NewBillHistoryStorage(database)

	serviceSvc := NewServiceService(serviceStorage, homeStorage, currencyStorage, billStorage)
	billSvc := NewBillService(billStorage, serviceStorage)
	billSvc.SetBillHistoryStorage(historyStorage)
	settingsSvc := NewSystemSettingsService(systemSettingsStorage)
	webhookSvc := NewWebhookService(systemSettingsStorage, settingsSvc, serviceStorage, billStorage)
	webhookSvc.SetBillHistoryStorage(historyStorage)

	if err := serviceSvc.EnsureWebhookUUIDs(ctx); err != nil {
		t.Fatalf("backfill uuid: %v", err)
	}
	return webhookSvc, serviceSvc, billSvc
}

func createTestService(t *testing.T, webhookSvc *WebhookService, serviceSvc *ServiceService) *models.Service {
	t.Helper()
	ctx := context.Background()

	home, err := serviceSvc.homes.Create(ctx, "Casa Webhook", "")
	if err != nil {
		t.Fatalf("crear hogar: %v", err)
	}
	currencies, err := serviceSvc.currencies.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}

	svc, err := serviceSvc.Create(ctx, &models.Service{
		HomeID:          home.ID,
		Name:            "Internet Webhook",
		Institution:     "Claro",
		CurrencyID:      currencies[0].ID,
		Frequency:       FrequencyMonthly,
		SuggestedAmount: 45,
		Active:          true,
		IconKey:         "internet",
	})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return svc
}

func TestWebhookKeyGeneratedAndReused(t *testing.T) {
	webhookSvc, _, _ := newTestWebhook(t)
	ctx := context.Background()

	key1, err := webhookSvc.GetOrCreateWebhookKey(ctx)
	if err != nil {
		t.Fatalf("GetOrCreateWebhookKey: %v", err)
	}
	if len(key1) != 64 {
		t.Errorf("api_key esperada de 64 hex chars, got %d", len(key1))
	}
	key2, err := webhookSvc.GetOrCreateWebhookKey(ctx)
	if err != nil {
		t.Fatalf("segunda GetOrCreateWebhookKey: %v", err)
	}
	if key1 != key2 {
		t.Error("la api_key debe reutilizarse (no regenerarse)")
	}
	if !webhookSvc.ValidateWebhookKey(ctx, key1) {
		t.Error("la api_key generada debe validar")
	}
	if webhookSvc.ValidateWebhookKey(ctx, "wrong-key") {
		t.Error("una api_key incorrecta no debe validar")
	}
	if webhookSvc.ValidateWebhookKey(ctx, "") {
		t.Error("una api_key vacía no debe validar")
	}
}

func TestWebhookURLDefaultAndCustomBase(t *testing.T) {
	webhookSvc, _, _ := newTestWebhook(t)
	ctx := context.Background()

	// Default: http://ihost.local:8088
	url, err := webhookSvc.WebhookURL(ctx, "abc-123")
	if err != nil {
		t.Fatalf("WebhookURL default: %v", err)
	}
	if url != "http://ihost.local:8088/webhooks/abc-123" {
		t.Errorf("URL default esperada ihost.local, got %q", url)
	}

	// Base URL personalizada sin slash final.
	if err := webhookSvc.settings.SetWebhookBaseURL(ctx, "http://localhost:9000"); err != nil {
		t.Fatalf("SetWebhookBaseURL: %v", err)
	}
	url, err = webhookSvc.WebhookURL(ctx, "abc-123")
	if err != nil {
		t.Fatalf("WebhookURL custom: %v", err)
	}
	if url != "http://localhost:9000/webhooks/abc-123" {
		t.Errorf("URL custom esperada localhost:9000, got %q", url)
	}

	// Base URL con slash final se normaliza.
	if err := webhookSvc.settings.SetWebhookBaseURL(ctx, "https://ihost.local:8443/"); err != nil {
		t.Fatalf("SetWebhookBaseURL con slash: %v", err)
	}
	url, _ = webhookSvc.WebhookURL(ctx, "xyz")
	if url != "https://ihost.local:8443/webhooks/xyz" {
		t.Errorf("URL con slash final esperada sin doble slash, got %q", url)
	}
}

func TestWebhookKeyRegenerateInvalidatesOld(t *testing.T) {
	webhookSvc, _, _ := newTestWebhook(t)
	ctx := context.Background()

	oldKey, _ := webhookSvc.GetOrCreateWebhookKey(ctx)
	newKey, err := webhookSvc.RegenerateWebhookKey(ctx)
	if err != nil {
		t.Fatalf("RegenerateWebhookKey: %v", err)
	}
	if oldKey == newKey {
		t.Error("la api_key regenerada debe ser distinta")
	}
	if !webhookSvc.ValidateWebhookKey(ctx, newKey) {
		t.Error("la nueva api_key debe validar")
	}
	if webhookSvc.ValidateWebhookKey(ctx, oldKey) {
		t.Error("la api_key anterior debe quedar inválida")
	}
}

func TestWebhookUpsertCreatesBill(t *testing.T) {
	webhookSvc, serviceSvc, billSvc := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, webhookSvc, serviceSvc)

	res, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year:          2025,
		Month:         12,
		Amount:        1234.56,
		InvoiceNumber: "INV-2025-12",
	})
	if err != nil {
		t.Fatalf("UpsertBill crear: %v", err)
	}
	if !res.Created {
		t.Error("se esperaba created=true en creación")
	}
	if res.Bill.Status != "pending" {
		t.Errorf("estado esperado pending, got %s", res.Bill.Status)
	}
	if res.Bill.Amount != 1234.56 {
		t.Errorf("monto esperado 1234.56, got %f", res.Bill.Amount)
	}

	bills, err := billSvc.ListByService(ctx, svc.ID)
	if err != nil {
		t.Fatalf("listar facturas: %v", err)
	}
	// createTestService genera la factura del mes actual + la del webhook.
	found := false
	for _, b := range bills {
		if b.Year == 2025 && b.Month == 12 {
			found = true
		}
	}
	if !found {
		t.Error("la factura del webhook no aparece en la lista")
	}
}

func TestWebhookUpsertUpdatesAndPays(t *testing.T) {
	webhookSvc, serviceSvc, _ := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, webhookSvc, serviceSvc)

	// Crear pendiente.
	if _, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year:   2025,
		Month:  10,
		Amount: 100,
	}); err != nil {
		t.Fatalf("crear pendiente: %v", err)
	}

	// Reenviar con status paid: debe actualizar y pagar.
	res, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year:             2025,
		Month:            10,
		Amount:           120,
		InvoiceNumber:    "INV-2025-10",
		Status:           "paid",
		PaidAt:           "2025-10-01",
		PaymentReference: "TXN-001",
	})
	if err != nil {
		t.Fatalf("UpsertBill pagar: %v", err)
	}
	if res.Created {
		t.Error("se esperaba created=false en actualización")
	}
	if res.Bill.Status != "paid" {
		t.Errorf("estado esperado paid, got %s", res.Bill.Status)
	}
	if res.Bill.PaymentReference != "TXN-001" {
		t.Errorf("payment_reference esperado TXN-001, got %q", res.Bill.PaymentReference)
	}
	if res.Bill.Amount != 120 {
		t.Errorf("monto esperado 120, got %f", res.Bill.Amount)
	}
	if res.Bill.PaidAt == nil {
		t.Error("paid_at debería estar seteado")
	}
}

func TestWebhookUpsertRevertsPaidToPending(t *testing.T) {
	webhookSvc, serviceSvc, _ := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, webhookSvc, serviceSvc)

	if _, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year: 2025, Month: 11, Amount: 50, Status: "paid", PaidAt: "2025-11-01",
	}); err != nil {
		t.Fatalf("crear pagada: %v", err)
	}

	res, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year: 2025, Month: 11, Amount: 50, Status: "pending",
	})
	if err != nil {
		t.Fatalf("revertir a pending: %v", err)
	}
	if res.Bill.Status != "pending" {
		t.Errorf("estado esperado pending tras revertir, got %s", res.Bill.Status)
	}
	if res.Bill.PaidAt != nil {
		t.Error("paid_at debería limpiarse al revertir a pending")
	}
}

func TestWebhookUpsertValidation(t *testing.T) {
	webhookSvc, serviceSvc, _ := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, webhookSvc, serviceSvc)

	cases := []models.WebhookBillPayload{
		{Year: 1800, Month: 1, Amount: 10},                                  // año inválido
		{Year: 2026, Month: 13, Amount: 10},                                 // mes inválido (mensual)
		{Year: 2026, Month: 1, Amount: -5},                                  // monto negativo
		{Year: 2026, Month: 1, Amount: 10, Status: "cancelado"},             // status inválido
		{Year: 2026, Month: 1, Amount: 10, DriveURL: "https://example.com"}, // drive_url inválido
	}
	for i, tc := range cases {
		if _, err := webhookSvc.UpsertBill(ctx, svc, &tc); err == nil {
			t.Errorf("caso %d: se esperaba error de validación", i)
		}
	}
}

func TestWebhookHistoryRecordsPaidWithAmountZero(t *testing.T) {
	webhookSvc, serviceSvc, billSvc := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, webhookSvc, serviceSvc)

	// Crear pendiente con monto 350.5.
	if _, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year:   2026,
		Month:  4,
		Amount: 350.5,
	}); err != nil {
		t.Fatalf("crear pendiente: %v", err)
	}

	// Webhook con amount 0 y status paid (caso reportado por el usuario).
	res, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year:          2026,
		Month:         4,
		Amount:        0,
		InvoiceNumber: "FAC0252065392026",
		Status:        "paid",
	})
	if err != nil {
		t.Fatalf("UpsertBill paid: %v", err)
	}
	if res.Bill.Status != "paid" || res.Bill.Amount != 0 {
		t.Fatalf("estado/monto esperados paid/0, got %s/%f", res.Bill.Status, res.Bill.Amount)
	}

	history, err := billSvc.History(ctx, res.Bill.ID)
	if err != nil {
		t.Fatalf("listar historial: %v", err)
	}
	events := history
	if len(events) < 2 {
		t.Fatalf("se esperaban al menos 2 eventos, got %d", len(events))
	}
	paid := events[0]
	if paid.Action != models.BillActionPaid || paid.Source != models.BillSourceWebhook {
		t.Errorf("evento esperado paid/webhook, got %s/%s", paid.Action, paid.Source)
	}
	foundAmountChange := false
	for _, c := range paid.Changes {
		if c.Field == "amount" {
			foundAmountChange = true
			if c.Old.(float64) != 350.5 || c.New.(float64) != 0 {
				t.Errorf("diff amount incorrecto: %v -> %v", c.Old, c.New)
			}
		}
	}
	if !foundAmountChange {
		t.Errorf("el diff debería incluir amount 350.5 -> 0, got %+v", paid.Changes)
	}
}

func TestWebhookHistorySkipsNoop(t *testing.T) {
	webhookSvc, serviceSvc, billSvc := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, webhookSvc, serviceSvc)

	if _, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year: 2026, Month: 3, Amount: 100, InvoiceNumber: "INV-NOOP",
	}); err != nil {
		t.Fatalf("crear: %v", err)
	}
	// Reenviar exactamente los mismos valores: no debe crear evento updated.
	if _, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year: 2026, Month: 3, Amount: 100, InvoiceNumber: "INV-NOOP",
	}); err != nil {
		t.Fatalf("reenviar: %v", err)
	}

	bills, err := billSvc.ListByService(ctx, svc.ID)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	var target *models.Bill
	for i := range bills {
		if bills[i].Year == 2026 && bills[i].Month == 3 {
			target = &bills[i]
		}
	}
	if target == nil {
		t.Fatal("factura no encontrada")
	}
	events, err := billSvc.History(ctx, target.ID)
	if err != nil {
		t.Fatalf("listar historial: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("noop no debe crear evento, got %d", len(events))
	}
}

func TestWebhookYearlyIgnoresMonth(t *testing.T) {
	webhookSvc, serviceSvc, _ := newTestWebhook(t)
	ctx := context.Background()

	home, err := serviceSvc.homes.Create(ctx, "Casa Anual", "")
	if err != nil {
		t.Fatalf("crear hogar: %v", err)
	}
	currencies, _ := serviceSvc.currencies.List(ctx)
	svc, err := serviceSvc.Create(ctx, &models.Service{
		HomeID:          home.ID,
		Name:            "Seguro Anual",
		CurrencyID:      currencies[0].ID,
		Frequency:       FrequencyYearly,
		SuggestedAmount: 300,
		Active:          true,
		IconKey:         "insurance",
	})
	if err != nil {
		t.Fatalf("crear servicio anual: %v", err)
	}

	// El mes se ignora: la factura anual se identifica solo por año.
	res, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year: 2026, Month: 5, Amount: 320, Status: "paid", PaidAt: "2026-01-05",
	})
	if err != nil {
		t.Fatalf("UpsertBill anual: %v", err)
	}
	if res.Bill.Month != 0 {
		t.Errorf("mes esperado 0 para servicio anual, got %d", res.Bill.Month)
	}
	if res.Bill.Status != "paid" {
		t.Errorf("estado esperado paid, got %s", res.Bill.Status)
	}
}

func TestServiceWebhookUUIDAssignedAndBackfilled(t *testing.T) {
	_, serviceSvc, _ := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, nil, serviceSvc)

	if svc.WebhookUUID == "" {
		t.Error("el servicio recién creado debe tener webhook_uuid")
	}

	found, err := serviceSvc.FindByWebhookUUID(ctx, svc.WebhookUUID)
	if err != nil {
		t.Fatalf("FindByWebhookUUID: %v", err)
	}
	if found == nil || found.ID != svc.ID {
		t.Errorf("FindByWebhookUUID no devolvió el servicio correcto")
	}

	if missing, err := serviceSvc.FindByWebhookUUID(ctx, "no-existe"); err != nil || missing != nil {
		t.Errorf("uuid inexistente debería devolver nil (err=%v)", err)
	}

	regenerated, err := serviceSvc.RegenerateWebhookUUID(ctx, svc.ID)
	if err != nil {
		t.Fatalf("RegenerateWebhookUUID: %v", err)
	}
	if regenerated.WebhookUUID == svc.WebhookUUID {
		t.Error("el uuid regenerado debe ser distinto")
	}
}
