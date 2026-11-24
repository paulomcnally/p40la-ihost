package api

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	mcpapi "github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// TestMCPBackendContract recorre la cadena real API key → router → handler →
// servicio → storage → JSON y la compara con los DTO que consume el MCP.
// Detecta drift de rutas, tags JSON, auth y auditoría source=mcp sin depender
// de datos de producción (SPEC-153).
func TestMCPBackendContract(t *testing.T) {
	database, err := db.OpenDB(filepath.Join(t.TempDir(), "test.db"), "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("ejecutar SQL de prueba (%q): %v", query, err)
		}
	}
	mustExec("INSERT INTO homes (name) VALUES ('Casa Demo')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Servicio Demo', 'Proveedor Demo', 1, 'monthly', 100, 1, 'internet', 'fixed', 1)`)

	billStorage := storage.NewBillStorage(database)
	serviceStorage := storage.NewServiceStorage(database)
	homeStorage := storage.NewHomeStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	billHistoryStorage := storage.NewBillHistoryStorage(database)
	systemSettingsStorage := storage.NewSystemSettingsStorage(database)

	billService := services.NewBillService(billStorage, serviceStorage)
	billService.SetBillHistoryStorage(billHistoryStorage)
	billService.SetCurrencyStorage(currencyStorage)

	serviceService := services.NewServiceService(serviceStorage, homeStorage, currencyStorage, billStorage)
	serviceHandlers := NewServiceHandlers(
		serviceService,
		services.NewHomeService(homeStorage),
		storage.NewInstitutionStorage(database),
		nil,
	)
	billHandlers := NewBillHandlers(billService)

	apiKeyService := services.NewAPIKeyService(systemSettingsStorage)
	apiKey, _, err := apiKeyService.Regenerate(ctx)
	if err != nil {
		t.Fatalf("generar API key de prueba: %v", err)
	}

	handler := &Handler{
		service:  serviceHandlers,
		bill:     billHandlers,
		webhooks: &WebhookHandlers{},
		apiKeys:  apiKeyService,
	}
	server := httptest.NewServer(BuildRouter(handler, nil, t.TempDir()))
	t.Cleanup(server.Close)

	client, err := mcpapi.New(config.Config{
		BaseURL: server.URL,
		APIKey:  apiKey,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("crear cliente MCP: %v", err)
	}
	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	service, err := client.GetService(requestCtx, 1)
	if err != nil {
		t.Fatalf("GetService: %v", err)
	}
	if service.Name != "Servicio Demo" || service.ID != 1 {
		t.Fatalf("servicio inesperado: %+v", service)
	}

	initialBills, err := client.ListBills(requestCtx, 1)
	if err != nil {
		t.Fatalf("ListBills inicial: %v", err)
	}
	if len(initialBills) != 0 {
		t.Fatalf("facturas iniciales = %d, quiero 0", len(initialBills))
	}

	generated, err := client.GenerateBills(requestCtx, 1, mcpapi.GenerateRequest{
		From:   "2026-01",
		To:     "2026-02",
		DryRun: false,
	})
	if err != nil {
		t.Fatalf("GenerateBills: %v", err)
	}
	if generated.Created != 2 || len(generated.Items) != 2 {
		t.Fatalf("generación inesperada: %+v", generated)
	}
	if generated.ServiceName != "Servicio Demo" {
		t.Fatalf("service_name = %q", generated.ServiceName)
	}

	bills, err := client.ListBills(requestCtx, 1)
	if err != nil {
		t.Fatalf("ListBills posterior: %v", err)
	}
	if len(bills) != 2 {
		t.Fatalf("facturas = %d, quiero 2", len(bills))
	}
	for _, bill := range bills {
		if bill.ServiceID != 1 || bill.Status != "pending" {
			t.Fatalf("factura inesperada: %+v", bill)
		}
	}

	paid, err := client.PayBills(requestCtx, 1, mcpapi.PayRequest{
		Month:  "2026-01",
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("PayBills dry-run: %v", err)
	}
	if !paid.DryRun || paid.Selector != "month" || len(paid.Items) != 1 {
		t.Fatalf("pago dry-run inesperado: %+v", paid)
	}

	var source string
	if err := database.QueryRowContext(ctx,
		"SELECT source FROM bill_history WHERE action = 'created' ORDER BY id DESC LIMIT 1",
	).Scan(&source); err != nil {
		t.Fatalf("leer bill_history.source: %v", err)
	}
	if source != "mcp" {
		t.Fatalf("source = %q, quiero mcp", source)
	}
}
