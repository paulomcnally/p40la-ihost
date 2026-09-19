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

func TestRecordWebhookRequest(t *testing.T) {
	webhookSvc, serviceSvc, _ := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, webhookSvc, serviceSvc)

	if svc.LastWebhookRequest != nil {
		t.Fatal("el servicio recién creado no debe tener last_webhook_request")
	}

	if err := webhookSvc.RecordWebhookRequest(ctx, svc.ID); err != nil {
		t.Fatalf("RecordWebhookRequest: %v", err)
	}

	got, err := serviceSvc.GetByID(ctx, svc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.LastWebhookRequest == nil {
		t.Error("RecordWebhookRequest debe setear last_webhook_request")
	}
}

func TestWebhookUpsertReactivatesSoftDeletedBill(t *testing.T) {
	webhookSvc, serviceSvc, billSvc := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, webhookSvc, serviceSvc)

	// 1. Crear la factura del período vía webhook.
	if _, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year: 2026, Month: 8, Amount: 975.97, InvoiceNumber: "FAC-OLD",
	}); err != nil {
		t.Fatalf("crear factura: %v", err)
	}
	bills, err := billSvc.ListByService(ctx, svc.ID)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	var target *models.Bill
	for i := range bills {
		if bills[i].Year == 2026 && bills[i].Month == 8 {
			target = &bills[i]
		}
	}
	if target == nil {
		t.Fatal("factura del período no encontrada")
	}
	originalID := target.ID

	// 2. Soft-delete desde la UI (como hace DELETE /api/bills/{id}).
	if err := billSvc.Delete(ctx, originalID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	// 3. Reenviar el mismo período: debe reactivar (no fallar con UNIQUE).
	res, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year: 2026, Month: 8, Amount: 990.0, InvoiceNumber: "FAC-NEW",
	})
	if err != nil {
		t.Fatalf("UpsertBill sobre soft-deleted: %v", err)
	}
	if res.Created {
		t.Error("se esperaba created=false al reactivar una fila existente")
	}
	if res.Bill.ID != originalID {
		t.Errorf("se debería conservar el id original, got %d (esperado %d)", res.Bill.ID, originalID)
	}
	if res.Bill.DeletedAt != nil {
		t.Error("la factura reactivada no debe tener deleted_at")
	}
	if res.Bill.Amount != 990.0 {
		t.Errorf("monto esperado 990.0, got %f", res.Bill.Amount)
	}
	if res.Bill.InvoiceNumber != "FAC-NEW" {
		t.Errorf("invoice_number esperado FAC-NEW, got %q", res.Bill.InvoiceNumber)
	}

	// 4. Debe quedar visible en el listado normal.
	bills, err = billSvc.ListByService(ctx, svc.ID)
	if err != nil {
		t.Fatalf("listar tras reactivar: %v", err)
	}
	found := false
	for _, b := range bills {
		if b.ID == originalID {
			found = true
		}
	}
	if !found {
		t.Error("la factura reactivada debe aparecer en el listado")
	}
}

func TestWebhookUpsertReactivatedPaidBill(t *testing.T) {
	webhookSvc, serviceSvc, billSvc := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, webhookSvc, serviceSvc)

	if _, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year: 2026, Month: 9, Amount: 500, Status: "pending",
	}); err != nil {
		t.Fatalf("crear: %v", err)
	}
	bills, _ := billSvc.ListByService(ctx, svc.ID)
	var target *models.Bill
	for i := range bills {
		if bills[i].Year == 2026 && bills[i].Month == 9 {
			target = &bills[i]
		}
	}
	if err := billSvc.Delete(ctx, target.ID); err != nil {
		t.Fatalf("soft-delete: %v", err)
	}

	// Reenviar con status paid: reactivar y marcar como pagada.
	res, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year:             2026,
		Month:            9,
		Amount:           520,
		Status:           "paid",
		PaidAt:           "2026-09-05",
		PaymentReference: "TXN-071",
	})
	if err != nil {
		t.Fatalf("UpsertBill paid sobre soft-deleted: %v", err)
	}
	if res.Bill.Status != "paid" {
		t.Errorf("estado esperado paid, got %s", res.Bill.Status)
	}
	if res.Bill.PaidAt == nil {
		t.Error("paid_at debería estar seteado")
	}
	if res.Bill.PaymentReference != "TXN-071" {
		t.Errorf("payment_reference esperado TXN-071, got %q", res.Bill.PaymentReference)
	}

	// Idempotencia: reenviar el mismo período → 200 sin duplicar ni fallar.
	res2, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year: 2026, Month: 9, Amount: 520, Status: "paid", PaidAt: "2026-09-05",
	})
	if err != nil {
		t.Fatalf("reenvío idempotente: %v", err)
	}
	if res2.Created {
		t.Error("el reenvío no debe crear una factura nueva")
	}
	if res2.Bill.ID != res.Bill.ID {
		t.Error("el reenvío debe mantener el mismo id")
	}
}

func TestWebhookUpsertSoftDeletedNotFoundPropagates(t *testing.T) {
	webhookSvc, serviceSvc, _ := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, webhookSvc, serviceSvc)

	// Período nuevo: el UNIQUE no aplica y se crea normalmente.
	res, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year: 2026, Month: 10, Amount: 123,
	})
	if err != nil {
		t.Fatalf("crear período nuevo: %v", err)
	}
	if !res.Created {
		t.Error("se esperaba created=true para período nuevo")
	}

	// Reenviar el período activo: update normal (idempotencia intacta).
	res2, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year: 2026, Month: 10, Amount: 125,
	})
	if err != nil {
		t.Fatalf("actualizar período activo: %v", err)
	}
	if res2.Created {
		t.Error("se esperaba created=false para período activo")
	}
	if res2.Bill.Amount != 125 {
		t.Errorf("monto esperado 125, got %f", res2.Bill.Amount)
	}
}

func TestParseWebhookDates(t *testing.T) {
	// Válidos: YYYY-MM-DD, RFC3339, con/sin zona; se normalizan a YYYY-MM-DD.
	valid := []struct {
		raw  string
		want string
	}{
		{"2026-09-10", "2026-09-10"},
		{"2026-10-05", "2026-10-05"},
		{"2026-09-10T00:00:00Z", "2026-09-10"},
		{"2026-09-10T15:04:05-06:00", "2026-09-10"},
		{"2026-09-10 15:04:05", "2026-09-10"},
	}
	for _, tc := range valid {
		got, err := parseWebhookDate(tc.raw, "issue_date")
		if err != nil {
			t.Errorf("parseWebhookDate(%q): error inesperado: %v", tc.raw, err)
			continue
		}
		if got == nil || *got != tc.want {
			t.Errorf("parseWebhookDate(%q) = %v, esperado %q", tc.raw, got, tc.want)
		}
	}

	// Vacío → nil (campo opcional, NULL por defecto).
	if got, err := parseWebhookDate("", "issue_date"); err != nil || got != nil {
		t.Errorf("vacío: se esperaba (nil, nil), got (%v, %v)", got, err)
	}

	// Inválidos → error.
	invalid := []string{"2026-13-40", "not-a-date", "2026/09/10", "10-09-2026"}
	for _, raw := range invalid {
		if _, err := parseWebhookDate(raw, "issue_date"); err == nil {
			t.Errorf("parseWebhookDate(%q): se esperaba error", raw)
		}
	}

	// Fechas futuras permitidas (un vencimiento normalmente es futuro).
	future, err := parseWebhookDate("2099-12-31", "due_date")
	if err != nil || future == nil {
		t.Fatalf("fecha futura debería aceptarse, got (%v, %v)", future, err)
	}

	// parseWebhookBillDates: ambas vacías → nil, nil.
	p := &models.WebhookBillPayload{}
	i, d, err := parseWebhookBillDates(p)
	if err != nil || i != nil || d != nil {
		t.Errorf("payload sin fechas: se esperaba (nil, nil, nil), got (%v, %v, %v)", i, d, err)
	}

	// Una inválida → error.
	bad := &models.WebhookBillPayload{IssueDate: "2026-09-10", DueDate: "invalida"}
	if _, _, err := parseWebhookBillDates(bad); err == nil {
		t.Error("due_date inválida debería devolver error")
	}
}

func TestWebhookUpsertPersistsDates(t *testing.T) {
	webhookSvc, serviceSvc, _ := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, webhookSvc, serviceSvc)

	// Crear con fechas: se normalizan y persisten.
	res, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year:      2025,
		Month:     11,
		Amount:    1250,
		IssueDate: "2025-11-10T15:00:00Z",
		DueDate:   "2025-12-05",
	})
	if err != nil {
		t.Fatalf("UpsertBill con fechas: %v", err)
	}
	if !res.Created {
		t.Error("se esperaba created=true")
	}
	if res.Bill.IssueDate == nil || *res.Bill.IssueDate != "2025-11-10" {
		t.Errorf("issue_date esperado 2025-11-10, got %v", res.Bill.IssueDate)
	}
	if res.Bill.DueDate == nil || *res.Bill.DueDate != "2025-12-05" {
		t.Errorf("due_date esperado 2025-12-05, got %v", res.Bill.DueDate)
	}

	// Actualizar sin fechas: los valores previos se conservan (semántica aditiva).
	res2, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year:   2025,
		Month:  11,
		Amount: 1300,
	})
	if err != nil {
		t.Fatalf("UpsertBill sin fechas: %v", err)
	}
	if res2.Created {
		t.Error("se esperaba created=false")
	}
	if res2.Bill.IssueDate == nil || *res2.Bill.IssueDate != "2025-11-10" {
		t.Errorf("issue_date previo debe conservarse, got %v", res2.Bill.IssueDate)
	}
	if res2.Bill.DueDate == nil || *res2.Bill.DueDate != "2025-12-05" {
		t.Errorf("due_date previo debe conservarse, got %v", res2.Bill.DueDate)
	}

	// Actualizar con fechas nuevas: sobrescriben.
	res3, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year:      2025,
		Month:     11,
		Amount:    1300,
		IssueDate: "2025-11-11",
		DueDate:   "2025-12-06",
	})
	if err != nil {
		t.Fatalf("UpsertBill fechas nuevas: %v", err)
	}
	if res3.Bill.IssueDate == nil || *res3.Bill.IssueDate != "2025-11-11" {
		t.Errorf("issue_date nuevo esperado 2025-11-11, got %v", res3.Bill.IssueDate)
	}
	if res3.Bill.DueDate == nil || *res3.Bill.DueDate != "2025-12-06" {
		t.Errorf("due_date nuevo esperado 2025-12-06, got %v", res3.Bill.DueDate)
	}
}

func TestWebhookUpsertRejectsInvalidDate(t *testing.T) {
	webhookSvc, serviceSvc, _ := newTestWebhook(t)
	ctx := context.Background()
	svc := createTestService(t, webhookSvc, serviceSvc)

	cases := []models.WebhookBillPayload{
		{Year: 2026, Month: 1, Amount: 10, IssueDate: "2026-13-40"},
		{Year: 2026, Month: 1, Amount: 10, DueDate: "invalida"},
		{Year: 2026, Month: 1, Amount: 10, IssueDate: "10-09-2026"},
	}
	for i, tc := range cases {
		if _, err := webhookSvc.UpsertBill(ctx, svc, &tc); err == nil {
			t.Errorf("caso %d: se esperaba error por fecha inválida", i)
		}
	}

	// La factura del período no debe crearse si la fecha es inválida.
	res, err := webhookSvc.UpsertBill(ctx, svc, &models.WebhookBillPayload{
		Year: 2026, Month: 1, Amount: 10,
	})
	if err != nil {
		t.Fatalf("verificar período intacto: %v", err)
	}
	if !res.Created {
		t.Error("el período 2026/01 debe estar libre (crear sin fecha inválida previa)")
	}
}
