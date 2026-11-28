package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	mcpapi "github.com/paulomcnally/p40la-ihost/internal/mcp/api"
	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
	"github.com/paulomcnally/p40la-ihost/internal/modules"
	homemaintenance "github.com/paulomcnally/p40la-ihost/internal/modules/home/maintenance"
	homeworkrequest "github.com/paulomcnally/p40la-ihost/internal/modules/home/workrequest"
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

	// Presupuesto (SPEC-163): la cadena real de /api/budget/* contra el router.
	mustExec("INSERT INTO category_groups (name, icon, sort_order) VALUES ('Necesidades', 'cart', 0)")
	mustExec("INSERT INTO categories (category_group_id, name, icon, sort_order) VALUES (1, 'Comida', 'food', 0)")
	mustExec("INSERT INTO accounts (name, type, currency_id, starting_balance) VALUES ('Cuenta Demo', 'checking', 1, 1000)")

	budgetGroupStorage := storage.NewCategoryGroupStorage(database)
	budgetCategoryStorage := storage.NewCategoryStorage(database)
	budgetAccountStorage := storage.NewAccountStorage(database)
	budgetMonthStorage := storage.NewBudgetMonthStorage(database)
	budgetAssignmentStorage := storage.NewAssignmentStorage(database)
	budgetRuleStorage := storage.NewRecurringRuleStorage(database)
	budgetTransactionStorage := storage.NewTransactionStorage(database)
	budgetTransferStorage := storage.NewTransferStorage(database)
	budgetHandlers := NewBudgetHandlers(
		services.NewBudgetService(budgetGroupStorage, budgetCategoryStorage, budgetMonthStorage,
			budgetAssignmentStorage, budgetRuleStorage, budgetTransactionStorage, currencyStorage),
		services.NewCategoryGroupService(budgetGroupStorage, budgetCategoryStorage),
		services.NewCategoryService(budgetCategoryStorage, budgetGroupStorage),
		services.NewAccountService(budgetAccountStorage, currencyStorage),
		services.NewBudgetTransactionService(budgetTransactionStorage, budgetAccountStorage, budgetCategoryStorage, currencyStorage),
		services.NewTransferService(budgetTransferStorage, budgetAccountStorage, currencyStorage),
	)

	apiKeyService := services.NewAPIKeyService(systemSettingsStorage)
	apiKey, _, err := apiKeyService.Regenerate(ctx)
	if err != nil {
		t.Fatalf("generar API key de prueba: %v", err)
	}

	moduleRegistry := modules.NewRegistry()
	maintenanceModule := homemaintenance.New(database)
	moduleRegistry.Register(maintenanceModule)
	moduleRegistry.Register(homeworkrequest.New(database, maintenanceModule.Service(), t.TempDir(), homeworkrequest.UploadConfig{Secret: "demo-secret-demo-secret-demo-secret", TTL: homeworkrequest.DefaultUploadTTL}))

	handler := &Handler{
		service:        serviceHandlers,
		bill:           billHandlers,
		home:           homeHandlers,
		budget:         budgetHandlers,
		webhooks:       &WebhookHandlers{},
		apiKeys:        apiKeyService,
		moduleRegistry: moduleRegistry,
	}
	var sourceSeen atomic.Bool
	router := BuildRouter(handler, nil, t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-P40LA-Source") == "mcp" {
			sourceSeen.Store(true)
		}
		router.ServeHTTP(w, r)
	}))
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

	// Reportes de vivienda (SPEC-164/165): cadena real de las tools MCP.
	workRequest, err := client.CreateHomeWorkRequest(requestCtx, 1, mcpapi.HomeWorkRequestRequest{
		Title:       "Filtración demo",
		Description: "Gotea cuando llueve",
		Category:    "repair",
		Priority:    "high",
		ReportedAt:  "2026-03-10",
	})
	if err != nil {
		t.Fatalf("CreateHomeWorkRequest: %v", err)
	}
	if workRequest.Status != "reported" || workRequest.Title != "Filtración demo" || workRequest.AttachmentCount != 0 {
		t.Fatalf("reporte inesperado: %+v", workRequest)
	}

	workRequests, err := client.ListHomeWorkRequests(requestCtx, 1, mcpapi.HomeWorkRequestFilter{Status: "reported", Category: "repair"})
	if err != nil {
		t.Fatalf("ListHomeWorkRequests: %v", err)
	}
	if len(workRequests.Items) != 1 || workRequests.PendingCount != 1 || workRequests.Items[0].ID != workRequest.ID {
		t.Fatalf("listado de reportes inesperado: %+v", workRequests)
	}

	workRequestDetail, err := client.GetHomeWorkRequest(requestCtx, 1, workRequest.ID)
	if err != nil {
		t.Fatalf("GetHomeWorkRequest: %v", err)
	}
	if workRequestDetail.ID != workRequest.ID || workRequestDetail.Status != "reported" {
		t.Fatalf("detalle de reporte inesperado: %+v", workRequestDetail)
	}

	updatedWorkRequest, err := client.UpdateHomeWorkRequest(requestCtx, 1, workRequest.ID, mcpapi.HomeWorkRequestRequest{
		Title:       "Filtración revisada demo",
		Description: "Gotea cuando llueve",
		Category:    "other",
		Priority:    "low",
		ReportedAt:  "2026-03-10",
	})
	if err != nil {
		t.Fatalf("UpdateHomeWorkRequest: %v", err)
	}
	if updatedWorkRequest.Title != "Filtración revisada demo" || updatedWorkRequest.Category != "other" || updatedWorkRequest.Status != "reported" {
		t.Fatalf("edición de reporte inesperada: %+v", updatedWorkRequest)
	}

	convertedWorkRequest, err := client.ConvertHomeWorkRequest(requestCtx, 1, workRequest.ID, mcpapi.HomeWorkRequestConvertRequest{
		Kind: "preventive",
		Area: "roof",
	})
	if err != nil {
		t.Fatalf("ConvertHomeWorkRequest: %v", err)
	}
	if convertedWorkRequest.MaintenanceID == 0 || convertedWorkRequest.WorkRequest == nil || convertedWorkRequest.WorkRequest.Status != "converted" {
		t.Fatalf("conversión inesperada: %+v", convertedWorkRequest)
	}
	if convertedWorkRequest.WorkRequest.MaintenanceID == nil || *convertedWorkRequest.WorkRequest.MaintenanceID != convertedWorkRequest.MaintenanceID {
		t.Fatalf("maintenance_id no enlazado: %+v", convertedWorkRequest.WorkRequest)
	}

	convertedMaintenance, err := client.GetHomeMaintenance(requestCtx, 1, convertedWorkRequest.MaintenanceID)
	if err != nil {
		t.Fatalf("GetHomeMaintenance convertida: %v", err)
	}
	if convertedMaintenance.Kind != "preventive" || convertedMaintenance.Area != "roof" || convertedMaintenance.Description == "" {
		t.Fatalf("reparación convertida inesperada: %+v", convertedMaintenance)
	}
	if len(convertedMaintenance.Items) != 0 {
		t.Fatalf("la reparación convertida debería estar vacía: %+v", convertedMaintenance.Items)
	}

	if _, err := client.ConvertHomeWorkRequest(requestCtx, 1, workRequest.ID, mcpapi.HomeWorkRequestConvertRequest{}); err == nil {
		t.Fatal("la segunda conversión debería fallar")
	} else {
		var apiErr *mcpapi.APIError
		if !errors.As(err, &apiErr) || apiErr.Code != "already_converted" {
			t.Fatalf("segunda conversión = %v, want already_converted", err)
		}
	}

	secondWorkRequest, err := client.CreateHomeWorkRequest(requestCtx, 1, mcpapi.HomeWorkRequestRequest{
		Title:       "Agregar canaleta demo",
		Description: "Para el agua de lluvia",
		Category:    "improvement",
		Priority:    "normal",
		ReportedAt:  "2026-03-11",
	})
	if err != nil {
		t.Fatalf("CreateHomeWorkRequest segunda: %v", err)
	}
	dismissed, err := client.SetHomeWorkRequestStatus(requestCtx, 1, secondWorkRequest.ID, mcpapi.HomeWorkRequestStatusRequest{Status: "dismissed"})
	if err != nil {
		t.Fatalf("SetHomeWorkRequestStatus dismissed: %v", err)
	}
	if dismissed.Status != "dismissed" {
		t.Fatalf("estado descartado inesperado: %+v", dismissed)
	}
	reactivated, err := client.SetHomeWorkRequestStatus(requestCtx, 1, secondWorkRequest.ID, mcpapi.HomeWorkRequestStatusRequest{Status: "reported"})
	if err != nil {
		t.Fatalf("SetHomeWorkRequestStatus reported: %v", err)
	}
	if reactivated.Status != "reported" {
		t.Fatalf("estado reactivado inesperado: %+v", reactivated)
	}

	// SPEC-165 ampliación: idempotencia de alta, link de carga, multipart,
	// deduplicación, listado y borrado de evidencia por la cadena real.
	clientRequestID := "demo-contract-evidence-1"
	evidenceReport, err := client.CreateHomeWorkRequest(requestCtx, 1, mcpapi.HomeWorkRequestRequest{
		Title:           "Filtración demo con evidencia",
		Description:     "Gotea cuando llueve",
		Category:        "repair",
		Priority:        "high",
		ReportedAt:      "2026-03-15",
		ClientRequestID: &clientRequestID,
	})
	if err != nil {
		t.Fatalf("CreateHomeWorkRequest con client_request_id: %v", err)
	}
	duplicated, err := client.CreateHomeWorkRequest(requestCtx, 1, mcpapi.HomeWorkRequestRequest{
		Title:           "Filtración demo con evidencia",
		Description:     "Gotea cuando llueve",
		Category:        "repair",
		Priority:        "high",
		ReportedAt:      "2026-03-15",
		ClientRequestID: &clientRequestID,
	})
	if err != nil {
		t.Fatalf("segunda alta idempotente: %v", err)
	}
	if duplicated.ID != evidenceReport.ID || !duplicated.Deduplicated {
		t.Fatalf("alta idempotente = %+v / %+v", evidenceReport, duplicated)
	}

	uploadLink, err := client.CreateHomeWorkRequestUploadLink(requestCtx, 1, evidenceReport.ID,
		mcpapi.HomeWorkRequestUploadLinkRequest{Count: 3, TTLMinutes: 10})
	if err != nil {
		t.Fatalf("CreateHomeWorkRequestUploadLink: %v", err)
	}
	if uploadLink.Token == "" || uploadLink.Path != "/api/work-request-uploads" || uploadLink.MaxFiles != 3 {
		t.Fatalf("link inesperado: %+v", uploadLink)
	}

	var uploadBody bytes.Buffer
	uploadWriter := multipart.NewWriter(&uploadBody)
	for name, data := range map[string][]byte{
		"foto-demo.jpg":  append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, []byte("demo")...),
		"video-demo.mp4": append([]byte("\x00\x00\x00\x18ftypmp42"), []byte("demo")...),
	} {
		part, err := uploadWriter.CreateFormFile("file", name)
		if err != nil {
			t.Fatalf("parte multipart: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("escribir parte: %v", err)
		}
	}
	if err := uploadWriter.Close(); err != nil {
		t.Fatalf("cerrar multipart: %v", err)
	}
	uploadRequest, err := http.NewRequestWithContext(requestCtx, http.MethodPost, server.URL+uploadLink.Path, &uploadBody)
	if err != nil {
		t.Fatalf("request de carga: %v", err)
	}
	uploadRequest.Header.Set("Content-Type", uploadWriter.FormDataContentType())
	uploadRequest.Header.Set("Authorization", "Bearer "+uploadLink.Token)
	uploadResponse, err := http.DefaultClient.Do(uploadRequest)
	if err != nil {
		t.Fatalf("carga multipart: %v", err)
	}
	defer uploadResponse.Body.Close()
	var uploadResult struct {
		Added           int `json:"added"`
		Duplicates      int `json:"duplicates"`
		Rejected        int `json:"rejected"`
		AttachmentCount int `json:"attachment_count"`
	}
	if err := json.NewDecoder(uploadResponse.Body).Decode(&uploadResult); err != nil {
		t.Fatalf("decodificar carga: %v", err)
	}
	if uploadResponse.StatusCode != http.StatusOK || uploadResult.Added != 2 ||
		uploadResult.Rejected != 0 || uploadResult.AttachmentCount != 2 {
		t.Fatalf("carga inesperada: status=%d %+v", uploadResponse.StatusCode, uploadResult)
	}

	evidenceDetail, err := client.GetHomeWorkRequest(requestCtx, 1, evidenceReport.ID)
	if err != nil {
		t.Fatalf("GetHomeWorkRequest con evidencia: %v", err)
	}
	if evidenceDetail.AttachmentCount != 2 || len(evidenceDetail.Attachments) != 2 {
		t.Fatalf("evidencia listada = %+v", evidenceDetail)
	}
	for _, attachment := range evidenceDetail.Attachments {
		if attachment.SHA256 == "" {
			t.Fatalf("adjunto sin sha256: %+v", attachment)
		}
	}
	detailJSON, _ := json.Marshal(evidenceDetail)
	if strings.Contains(string(detailJSON), uploadLink.Token) {
		t.Fatal("el token se filtró en el listado")
	}

	if err := client.DeleteHomeWorkRequestAttachment(requestCtx, 1, evidenceReport.ID, evidenceDetail.Attachments[0].ID); err != nil {
		t.Fatalf("DeleteHomeWorkRequestAttachment: %v", err)
	}
	evidenceDetail, err = client.GetHomeWorkRequest(requestCtx, 1, evidenceReport.ID)
	if err != nil || evidenceDetail.AttachmentCount != 1 {
		t.Fatalf("evidencia tras borrar: %v %+v", err, evidenceDetail)
	}

	if _, err := client.ConvertHomeWorkRequest(requestCtx, 1, evidenceReport.ID, mcpapi.HomeWorkRequestConvertRequest{}); err != nil {
		t.Fatalf("convertir reporte con evidencia: %v", err)
	}
	var evidenceAPIErr *mcpapi.APIError
	if _, err := client.CreateHomeWorkRequestUploadLink(requestCtx, 1, evidenceReport.ID,
		mcpapi.HomeWorkRequestUploadLinkRequest{Count: 1}); !errors.As(err, &evidenceAPIErr) || evidenceAPIErr.Code != "invalid_status" {
		t.Fatalf("upload-link sobre convertido = %v", err)
	}
	if err := client.DeleteHomeWorkRequestAttachment(requestCtx, 1, evidenceReport.ID, evidenceDetail.Attachments[0].ID); !errors.As(err, &evidenceAPIErr) || evidenceAPIErr.Code != "invalid_status" {
		t.Fatalf("borrado MCP sobre convertido = %v", err)
	}

	badTokenRequest, err := http.NewRequestWithContext(requestCtx, http.MethodPost, server.URL+"/api/work-request-uploads", nil)
	if err != nil {
		t.Fatalf("request token inválido: %v", err)
	}
	badTokenRequest.Header.Set("Authorization", "Bearer demo-token-invalido")
	badTokenResponse, err := http.DefaultClient.Do(badTokenRequest)
	if err != nil {
		t.Fatalf("token inválido: %v", err)
	}
	defer badTokenResponse.Body.Close()
	if badTokenResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("token inválido status = %d", badTokenResponse.StatusCode)
	}
	if !sourceSeen.Load() {
		t.Fatal("no se observó X-P40LA-Source: mcp en la cadena real")
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

	editDemo, err := client.CreateHomeMaintenance(requestCtx, 1, mcpapi.HomeMaintenanceRequest{
		PerformedAt: "2026-03-14", Kind: "corrective", Area: "other", Description: "Reparación demo para editar",
	})
	if err != nil {
		t.Fatalf("CreateHomeMaintenance para editar: %v", err)
	}
	updatedDemo, err := client.UpdateHomeMaintenance(requestCtx, 1, editDemo.ID, mcpapi.HomeMaintenanceUpdateRequest{
		PerformedAt: "2026-03-14",
		Kind:        "corrective",
		Area:        "paint",
		Description: "Reparación demo de pintura",
	})
	if err != nil {
		t.Fatalf("UpdateHomeMaintenance: %v", err)
	}
	if updatedDemo.Area != "paint" || updatedDemo.Description != "Reparación demo de pintura" {
		t.Fatalf("cabecera editada inesperada: %+v", updatedDemo)
	}

	editLabor, err := client.CreateHomeMaintenanceItem(requestCtx, 1, editDemo.ID, mcpapi.HomeMaintenanceItemRequest{
		ItemType: "labor", Description: "Mano de obra demo", Amount: 50, CurrencyID: 1,
	})
	if err != nil {
		t.Fatalf("CreateHomeMaintenanceItem labor para editar: %v", err)
	}
	editSupplies, err := client.CreateHomeMaintenanceItem(requestCtx, 1, editDemo.ID, mcpapi.HomeMaintenanceItemRequest{
		ItemType: "supplies", Description: "Insumo demo", Quantity: 1, Unit: "unit", Amount: 30, CurrencyID: 1,
	})
	if err != nil {
		t.Fatalf("CreateHomeMaintenanceItem supplies para editar: %v", err)
	}
	updatedLabor, err := client.UpdateHomeMaintenanceItem(requestCtx, 1, editDemo.ID, editLabor.ID, mcpapi.HomeMaintenanceItemRequest{
		ItemType: "labor", Description: "Mano de obra demo", Quantity: 1, Unit: "unit", Amount: 75, CurrencyID: 1,
	})
	if err != nil {
		t.Fatalf("UpdateHomeMaintenanceItem: %v", err)
	}
	if updatedLabor.Amount != 75 || updatedLabor.Quantity != 1 || updatedLabor.Unit != "unit" {
		t.Fatalf("línea editada inesperada: %+v", updatedLabor)
	}
	editDetail, err := client.GetHomeMaintenance(requestCtx, 1, editDemo.ID)
	if err != nil {
		t.Fatalf("GetHomeMaintenance para editar: %v", err)
	}
	if len(editDetail.TotalsByCurrency) != 1 ||
		editDetail.TotalsByCurrency[0].LaborCost != 75 ||
		editDetail.TotalsByCurrency[0].SuppliesCost != 30 ||
		editDetail.TotalsByCurrency[0].Total != 105 {
		t.Fatalf("totales tras editar inesperados: %+v", editDetail.TotalsByCurrency)
	}

	if err := client.DeleteHomeMaintenanceItem(requestCtx, 1, editDemo.ID, editSupplies.ID); err != nil {
		t.Fatalf("DeleteHomeMaintenanceItem: %v", err)
	}
	editDetail, err = client.GetHomeMaintenance(requestCtx, 1, editDemo.ID)
	if err != nil {
		t.Fatalf("GetHomeMaintenance tras borrar: %v", err)
	}
	if len(editDetail.Items) != 1 || len(editDetail.TotalsByCurrency) != 1 || editDetail.TotalsByCurrency[0].Total != 75 {
		t.Fatalf("detalle tras borrar inesperado: %+v", editDetail)
	}

	_, err = client.UpdateHomeMaintenanceItem(requestCtx, 1, editDemo.ID, editLabor.ID, mcpapi.HomeMaintenanceItemRequest{
		ItemType: "labor", Description: "Mano de obra demo", Quantity: 1, Unit: "unit", Amount: -1, CurrencyID: 1,
	})
	if !errors.As(err, &apiErr) || apiErr.Code != "invalid_amount" {
		t.Fatalf("esperaba invalid_amount: %v", err)
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

	// Presupuesto (SPEC-163): categorías, cuentas, transacciones y vista mensual.
	groups, err := client.ListCategoryGroups(requestCtx)
	if err != nil {
		t.Fatalf("ListCategoryGroups: %v", err)
	}
	if len(groups) != 1 || groups[0].Name != "Necesidades" || len(groups[0].Categories) != 1 {
		t.Fatalf("grupos inesperados: %+v", groups)
	}
	category := groups[0].Categories[0]
	if category.ID != 1 || category.Name != "Comida" || category.CategoryGroupID != 1 {
		t.Fatalf("categoría inesperada: %+v", category)
	}

	budgetAccounts, err := client.ListAccounts(requestCtx)
	if err != nil {
		t.Fatalf("ListAccounts: %v", err)
	}
	if len(budgetAccounts) != 1 {
		t.Fatalf("cuentas = %d, quiero 1", len(budgetAccounts))
	}
	if account := budgetAccounts[0]; account.ID != 1 || account.Name != "Cuenta Demo" ||
		account.Type != "checking" || account.CurrencyID != 1 || account.CurrencyCode != "NIO" ||
		account.StartingBalance != 1000 || account.Balance != 1000 {
		t.Fatalf("cuenta inesperada: %+v", account)
	}

	initialTransactions, err := client.ListTransactions(requestCtx, 2026, 3)
	if err != nil {
		t.Fatalf("ListTransactions inicial: %v", err)
	}
	if len(initialTransactions) != 0 {
		t.Fatalf("transacciones iniciales = %d, quiero 0", len(initialTransactions))
	}

	categoryID := category.ID
	createdTransaction, err := client.CreateTransaction(requestCtx, mcpapi.TransactionRequest{
		AccountID:  1,
		CategoryID: &categoryID,
		CurrencyID: 1,
		Date:       "2026-03-10",
		Payee:      "Proveedor Demo",
		Memo:       "Compra demo",
		Outflow:    1200,
		Inflow:     0,
		Cleared:    false,
	})
	if err != nil {
		t.Fatalf("CreateTransaction: %v", err)
	}
	if createdTransaction.ID == 0 || createdTransaction.AccountID != 1 || createdTransaction.AccountName != "Cuenta Demo" ||
		createdTransaction.CategoryID == nil || *createdTransaction.CategoryID != 1 ||
		createdTransaction.CategoryName != "Comida" || createdTransaction.CurrencyID != 1 ||
		createdTransaction.CurrencyCode != "NIO" || createdTransaction.Date != "2026-03-10" ||
		createdTransaction.Payee != "Proveedor Demo" || createdTransaction.Memo != "Compra demo" ||
		createdTransaction.Outflow != 1200 || createdTransaction.Inflow != 0 || createdTransaction.Cleared {
		t.Fatalf("transacción creada inesperada: %+v", createdTransaction)
	}

	transactions, err := client.ListTransactions(requestCtx, 2026, 3)
	if err != nil {
		t.Fatalf("ListTransactions posterior: %v", err)
	}
	if len(transactions) != 1 || transactions[0].ID != createdTransaction.ID {
		t.Fatalf("transacciones del mes inesperadas: %+v", transactions)
	}

	view, err := client.MonthView(requestCtx, 2026, 3)
	if err != nil {
		t.Fatalf("MonthView: %v", err)
	}
	if view.Year != 2026 || view.Month != 3 || len(view.Groups) != 1 || len(view.Groups[0].Categories) != 1 {
		t.Fatalf("vista mensual inesperada: %+v", view)
	}
	row := view.Groups[0].Categories[0]
	if row.ID != 1 || row.Name != "Comida" {
		t.Fatalf("fila de categoría inesperada: %+v", row)
	}
	if row.Activity[1] != 1200 || row.Available[1] != -1200 {
		t.Fatalf("activity/available inesperados: activity=%v available=%v", row.Activity, row.Available)
	}
	if len(view.CurrencyTotals) != 1 || view.CurrencyTotals[0].CurrencyID != 1 ||
		view.CurrencyTotals[0].Code != "NIO" || view.CurrencyTotals[0].Unassigned != 0 {
		t.Fatalf("totales por moneda inesperados: %+v", view.CurrencyTotals)
	}

	_, err = client.CreateTransaction(requestCtx, mcpapi.TransactionRequest{
		AccountID:  999999,
		CurrencyID: 1,
		Date:       "2026-03-10",
		Outflow:    10,
	})
	if !errors.As(err, &apiErr) || apiErr.Code != "invalid_request" {
		t.Fatalf("esperaba invalid_request por cuenta inexistente: %v", err)
	}
}
