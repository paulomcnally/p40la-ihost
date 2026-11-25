package api

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	mcpapi "github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/modules"
	homemaintenance "github.com/paulomcnally/p40la-ihost/internal/modules/home/maintenance"
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
	homeService := services.NewHomeService(homeStorage)
	serviceHandlers := NewServiceHandlers(
		serviceService,
		homeService,
		storage.NewInstitutionStorage(database),
		nil,
	)
	homeHandlers := NewHomeHandlers(homeService)
	billHandlers := NewBillHandlers(billService)

	apiKeyService := services.NewAPIKeyService(systemSettingsStorage)
	apiKey, _, err := apiKeyService.Regenerate(ctx)
	if err != nil {
		t.Fatalf("generar API key de prueba: %v", err)
	}

	moduleRegistry := modules.NewRegistry()
	moduleRegistry.Register(homemaintenance.New(database))

	handler := &Handler{
		service:        serviceHandlers,
		bill:           billHandlers,
		home:           homeHandlers,
		webhooks:       &WebhookHandlers{},
		apiKeys:        apiKeyService,
		moduleRegistry: moduleRegistry,
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

	home, err := client.UpdateHome(requestCtx, 1, mcpapi.HomeUpdateRequest{
		Name:    "Casa Demo",
		Address: "Calle Demo 123",
		Usage:   "inhabited",
	})
	if err != nil {
		t.Fatalf("UpdateHome: %v", err)
	}
	if home.ID != 1 || home.Name != "Casa Demo" || home.Address != "Calle Demo 123" || home.Usage != "inhabited" {
		t.Fatalf("hogar inesperado: %+v", home)
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

	maintenance, err := client.CreateHomeMaintenance(requestCtx, 1, mcpapi.HomeMaintenanceRequest{
		PerformedAt: "2026-03-10",
		Kind:        "corrective",
		Area:        "plumbing",
		Description: "Reparación demo",
		Item: &mcpapi.HomeMaintenanceItemRequest{
			ItemType: "labor", Description: "Mano de obra demo", Amount: 50, CurrencyID: 1,
		},
	})
	if err != nil {
		t.Fatalf("CreateHomeMaintenance: %v", err)
	}
	if maintenance.ID == 0 || len(maintenance.Items) != 1 {
		t.Fatalf("mantenimiento inesperado: %+v", maintenance)
	}
	createdItem := maintenance.Items[0]
	if createdItem.ItemType != "labor" || createdItem.Quantity != 1 || createdItem.Unit != "unit" || createdItem.CurrencyID != 1 {
		t.Fatalf("línea inesperada: %+v", createdItem)
	}

	list, err := client.ListHomeMaintenances(requestCtx, 1, mcpapi.HomeMaintenanceFilter{
		Date: "2026-03-10", ItemType: "labor", Query: "demo", IncludeItems: true,
	})
	if err != nil {
		t.Fatalf("ListHomeMaintenances: %v", err)
	}
	if len(list.Items) != 1 || len(list.Items[0].Items) != 1 || list.Truncated {
		t.Fatalf("listado inesperado: %+v", list)
	}
	if len(list.TotalsByCurrency) != 1 || list.TotalsByCurrency[0].LaborCost != 50 {
		t.Fatalf("totales inesperados: %+v", list.TotalsByCurrency)
	}

	detail, err := client.GetHomeMaintenance(requestCtx, 1, maintenance.ID)
	if err != nil {
		t.Fatalf("GetHomeMaintenance: %v", err)
	}
	if len(detail.Items) != 1 || detail.Items[0].ID != createdItem.ID {
		t.Fatalf("detalle inesperado: %+v", detail)
	}

	supplies, err := client.CreateHomeMaintenanceItem(requestCtx, 1, maintenance.ID, mcpapi.HomeMaintenanceItemRequest{
		ItemType: "supplies", Description: "Insumo demo", Quantity: 2, Unit: "kg", Amount: 30, CurrencyID: 1,
	})
	if err != nil {
		t.Fatalf("CreateHomeMaintenanceItem: %v", err)
	}
	if supplies.ItemType != "supplies" || supplies.Quantity != 2 || supplies.Unit != "kg" {
		t.Fatalf("insumo inesperado: %+v", supplies)
	}

	list, err = client.ListHomeMaintenances(requestCtx, 1, mcpapi.HomeMaintenanceFilter{
		Date: "2026-03-10", ItemType: "supplies", IncludeItems: true,
	})
	if err != nil {
		t.Fatalf("ListHomeMaintenances supplies: %v", err)
	}
	if len(list.Items) != 1 || len(list.Items[0].Items) != 2 {
		t.Fatalf("listado con insumos inesperado: %+v", list)
	}
	if len(list.TotalsByCurrency) != 1 || list.TotalsByCurrency[0].LaborCost != 50 || list.TotalsByCurrency[0].SuppliesCost != 30 {
		t.Fatalf("totales con insumos inesperados: %+v", list.TotalsByCurrency)
	}

	empty, err := client.CreateHomeMaintenance(requestCtx, 1, mcpapi.HomeMaintenanceRequest{
		PerformedAt: "2026-03-12", Kind: "preventive", Area: "roof", Description: "Revisión demo de techo",
	})
	if err != nil {
		t.Fatalf("CreateHomeMaintenance sin líneas: %v", err)
	}
	if empty.ID == 0 || len(empty.Items) != 0 {
		t.Fatalf("reparación vacía inesperada: %+v", empty)
	}

	target, err := client.CreateHomeMaintenance(requestCtx, 1, mcpapi.HomeMaintenanceRequest{
		PerformedAt: "2026-03-11", Kind: "corrective", Area: "plumbing", Description: "Reparación demo destino",
	})
	if err != nil {
		t.Fatalf("CreateHomeMaintenance destino: %v", err)
	}
	moved, err := client.MoveHomeMaintenanceItems(requestCtx, 1, maintenance.ID, mcpapi.HomeMaintenanceMoveRequest{
		ItemIDs:             []int64{createdItem.ID, supplies.ID},
		TargetMaintenanceID: &target.ID,
	})
	if err != nil {
		t.Fatalf("MoveHomeMaintenanceItems: %v", err)
	}
	if moved.MovedCount != 2 || moved.SourceMaintenanceID != maintenance.ID || moved.TargetMaintenanceID != target.ID {
		t.Fatalf("movimiento inesperado: %+v", moved)
	}
	if moved.SourceItemCount != 0 || moved.SourceDeleted || moved.TargetCreated {
		t.Fatalf("estado del movimiento inesperado: %+v", moved)
	}
	if len(moved.MovedItems) != 2 {
		t.Fatalf("líneas movidas = %d, quiero 2", len(moved.MovedItems))
	}
	for _, item := range moved.MovedItems {
		if item.MaintenanceID != target.ID {
			t.Fatalf("línea no movida: %+v", item)
		}
	}

	sourceDetail, err := client.GetHomeMaintenance(requestCtx, 1, maintenance.ID)
	if err != nil {
		t.Fatalf("GetHomeMaintenance origen: %v", err)
	}
	if len(sourceDetail.Items) != 0 {
		t.Fatalf("el origen conserva líneas: %+v", sourceDetail.Items)
	}
	targetDetail, err := client.GetHomeMaintenance(requestCtx, 1, target.ID)
	if err != nil {
		t.Fatalf("GetHomeMaintenance destino: %v", err)
	}
	if len(targetDetail.Items) != 2 {
		t.Fatalf("el destino tiene %d líneas, quiero 2", len(targetDetail.Items))
	}
	if len(targetDetail.TotalsByCurrency) != 1 || targetDetail.TotalsByCurrency[0].Total != 80 {
		t.Fatalf("totales del destino inesperados: %+v", targetDetail.TotalsByCurrency)
	}

	emptyItem, err := client.CreateHomeMaintenanceItem(requestCtx, 1, empty.ID, mcpapi.HomeMaintenanceItemRequest{
		ItemType: "labor", Description: "Mano de obra demo", Amount: 10, CurrencyID: 1,
	})
	if err != nil {
		t.Fatalf("CreateHomeMaintenanceItem en reparación vacía: %v", err)
	}
	createdTarget, err := client.MoveHomeMaintenanceItems(requestCtx, 1, empty.ID, mcpapi.HomeMaintenanceMoveRequest{
		ItemIDs: []int64{emptyItem.ID},
		Target: &mcpapi.HomeMaintenanceTargetRequest{
			PerformedAt: "2026-03-13", Kind: "corrective", Area: "other", Description: "Reparación demo creada al mover",
		},
		DeleteSourceIfEmpty: true,
	})
	if err != nil {
		t.Fatalf("MoveHomeMaintenanceItems con destino nuevo: %v", err)
	}
	if !createdTarget.TargetCreated || !createdTarget.SourceDeleted || createdTarget.Target == nil {
		t.Fatalf("destino nuevo inesperado: %+v", createdTarget)
	}

	_, err = client.CreateHomeMaintenance(requestCtx, 1, mcpapi.HomeMaintenanceRequest{
		PerformedAt: "2026-03-11", Kind: "corrective", Area: "other", Description: "Reparación demo",
		Item: &mcpapi.HomeMaintenanceItemRequest{
			ItemType: "labor", Description: "Mano de obra demo", Amount: 10, CurrencyID: 999999,
		},
	})
	var apiErr *mcpapi.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "currency_not_found" {
		t.Fatalf("esperaba currency_not_found: %v", err)
	}

	_, err = client.CreateHomeMaintenance(requestCtx, 1, mcpapi.HomeMaintenanceRequest{
		PerformedAt: "2999-01-01", Kind: "corrective", Area: "other", Description: "Reparación demo",
		Item: &mcpapi.HomeMaintenanceItemRequest{
			ItemType: "labor", Description: "Mano de obra demo", Amount: 10, CurrencyID: 1,
		},
	})
	if !errors.As(err, &apiErr) || apiErr.Code != "future_date" {
		t.Fatalf("esperaba future_date: %v", err)
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
