package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/config"
	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

const testAutoRecordPDF = "%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n"

type autoRecordAPI struct {
	maintenance        *AutoMaintenanceHandlers
	inspection         *AutoInspectionHandlers
	attachment         *AutoAttachmentHandlers
	auth               *services.AuthService
	cookie             *http.Cookie
	autoID             int64
	workshopID         int64
	centerID           int64
	currencyID         int64
	maintenanceService *services.AutoMaintenanceService
}

func newAutoRecordAPI(t *testing.T) *autoRecordAPI {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	autoStorage := storage.NewAutoStorage(database)
	institutionStorage := storage.NewInstitutionStorage(database)
	categoryStorage := storage.NewInstitutionCategoryStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	maintenanceStorage := storage.NewAutoMaintenanceStorage(database)
	inspectionStorage := storage.NewAutoInspectionStorage(database)
	attachmentStorage := storage.NewAutoAttachmentStorage(database)

	auto, err := autoStorage.Create(ctx, 2020, "Corolla", "Toyota", "white", "vehicle", "", "", "", "")
	if err != nil {
		t.Fatalf("crear auto: %v", err)
	}
	workshopCategory, err := categoryStorage.GetByKey(ctx, "workshop")
	if err != nil || workshopCategory == nil {
		t.Fatalf("categoría workshop: %v", err)
	}
	centerCategory, err := categoryStorage.GetByKey(ctx, "mechanical_inspection")
	if err != nil || centerCategory == nil {
		t.Fatalf("categoría mechanical_inspection: %v", err)
	}
	workshop, err := institutionStorage.Create(ctx, &models.Institution{Name: "Taller Demo", CategoryID: &workshopCategory.ID})
	if err != nil {
		t.Fatalf("crear taller: %v", err)
	}
	center, err := institutionStorage.Create(ctx, &models.Institution{Name: "Centro Demo", CategoryID: &centerCategory.ID})
	if err != nil {
		t.Fatalf("crear centro: %v", err)
	}
	currencies, err := currencyStorage.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("listar monedas: %v", err)
	}

	attachmentService := services.NewAutoAttachmentService(attachmentStorage, autoStorage, t.TempDir())
	maintenanceService := services.NewAutoMaintenanceService(maintenanceStorage, autoStorage, institutionStorage, currencyStorage, attachmentService)
	inspectionService := services.NewAutoInspectionService(inspectionStorage, autoStorage, institutionStorage, attachmentService)

	cfg := &config.Config{BcryptCost: 10, SessionDuration: time.Hour, SecureCookie: false}
	auth := services.NewAuthService(storage.NewUserStorage(database), storage.NewSettingsStorage(database), cfg)
	_, cookie, err := auth.CreateFirstUser(ctx, "test@test.com", "Password123", "Password123")
	if err != nil {
		t.Fatalf("crear usuario: %v", err)
	}

	return &autoRecordAPI{
		maintenance:        NewAutoMaintenanceHandlers(maintenanceService),
		inspection:         NewAutoInspectionHandlers(inspectionService),
		attachment:         NewAutoAttachmentHandlers(attachmentService),
		auth:               auth,
		cookie:             cookie,
		autoID:             auto.ID,
		workshopID:         workshop.ID,
		centerID:           center.ID,
		currencyID:         currencies[0].ID,
		maintenanceService: maintenanceService,
	}
}

func (a *autoRecordAPI) createMaintenance(t *testing.T) *models.AutoMaintenanceDetail {
	t.Helper()
	item, err := a.maintenanceService.Create(context.Background(), a.autoID, models.AutoMaintenanceInput{
		PerformedAt:   "2026-10-15",
		Mileage:       42000,
		InstitutionID: &a.workshopID,
		LaborCost:     45,
		SuppliesCost:  80,
		CurrencyID:    a.currencyID,
	})
	if err != nil {
		t.Fatalf("crear mantenimiento: %v", err)
	}
	return item
}

func TestAutoMaintenanceHandlerCreateAndList(t *testing.T) {
	api := newAutoRecordAPI(t)

	body := fmt.Sprintf(`{
		"performed_at":"2026-10-15","mileage":42000,"institution_id":%d,
		"labor_cost":45,"supplies_cost":80,"currency_id":%d
	}`, api.workshopID, api.currencyID)
	req := httptest.NewRequest(http.MethodPost, "/api/autos/"+strconv.FormatInt(api.autoID, 10)+"/maintenances", bytes.NewBufferString(body))
	req.AddCookie(api.cookie)
	req.SetPathValue("id", strconv.FormatInt(api.autoID, 10))
	rr := httptest.NewRecorder()
	AuthMiddleware(api.auth)(http.HandlerFunc(api.maintenance.CreateMaintenance)).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("esperado 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var created models.AutoMaintenanceDetail
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("decodificar mantenimiento: %v", err)
	}
	if created.WorkshopName != "Taller Demo" {
		t.Errorf("workshop_name: esperado Taller Demo, got %q", created.WorkshopName)
	}
	if created.TotalCost != 125 {
		t.Errorf("total_cost: esperado 125, got %v", created.TotalCost)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/autos/"+strconv.FormatInt(api.autoID, 10)+"/maintenances", nil)
	listReq.AddCookie(api.cookie)
	listReq.SetPathValue("id", strconv.FormatInt(api.autoID, 10))
	listRR := httptest.NewRecorder()
	AuthMiddleware(api.auth)(http.HandlerFunc(api.maintenance.ListMaintenances)).ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("esperado 200, got %d: %s", listRR.Code, listRR.Body.String())
	}
	var list models.AutoMaintenanceList
	if err := json.Unmarshal(listRR.Body.Bytes(), &list); err != nil {
		t.Fatalf("decodificar lista: %v", err)
	}
	if len(list.Items) != 1 || list.LatestMileage != 42000 {
		t.Errorf("lista inesperada: %+v", list)
	}
}

func TestAutoMaintenanceHandlerRequiereSesionYAuto(t *testing.T) {
	api := newAutoRecordAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/api/autos/1/maintenances", nil)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	AuthMiddleware(api.auth)(http.HandlerFunc(api.maintenance.ListMaintenances)).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión: esperado 401, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/autos/999999/maintenances", nil)
	req.AddCookie(api.cookie)
	req.SetPathValue("id", "999999")
	rr = httptest.NewRecorder()
	AuthMiddleware(api.auth)(http.HandlerFunc(api.maintenance.ListMaintenances)).ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("auto inexistente: esperado 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAutoInspectionHandlerListSchedule(t *testing.T) {
	api := newAutoRecordAPI(t)

	body := fmt.Sprintf(`{
		"performed_at":"2022-01-10","inspection_year":2022,"institution_id":%d,
		"result":"approved","notes":"Sin observaciones"
	}`, api.centerID)
	req := httptest.NewRequest(http.MethodPost, "/api/autos/"+strconv.FormatInt(api.autoID, 10)+"/inspections", bytes.NewBufferString(body))
	req.AddCookie(api.cookie)
	req.SetPathValue("id", strconv.FormatInt(api.autoID, 10))
	rr := httptest.NewRecorder()
	AuthMiddleware(api.auth)(http.HandlerFunc(api.inspection.CreateInspection)).ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("esperado 201, got %d: %s", rr.Code, rr.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/autos/"+strconv.FormatInt(api.autoID, 10)+"/inspections", nil)
	listReq.AddCookie(api.cookie)
	listReq.SetPathValue("id", strconv.FormatInt(api.autoID, 10))
	listRR := httptest.NewRecorder()
	AuthMiddleware(api.auth)(http.HandlerFunc(api.inspection.ListInspections)).ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("esperado 200, got %d: %s", listRR.Code, listRR.Body.String())
	}
	var list models.AutoInspectionList
	if err := json.Unmarshal(listRR.Body.Bytes(), &list); err != nil {
		t.Fatalf("decodificar inspecciones: %v", err)
	}
	if list.Schedule.FirstDueYear != 2022 || list.Schedule.NextDueYear != 2023 {
		t.Errorf("cronograma inesperado: %+v", list.Schedule)
	}

	alertsReq := httptest.NewRequest(http.MethodGet, "/api/autos/inspection-alerts", nil)
	alertsReq.AddCookie(api.cookie)
	alertsRR := httptest.NewRecorder()
	AuthMiddleware(api.auth)(http.HandlerFunc(api.inspection.ListInspectionAlerts)).ServeHTTP(alertsRR, alertsReq)
	if alertsRR.Code != http.StatusOK {
		t.Fatalf("alertas: esperado 200, got %d: %s", alertsRR.Code, alertsRR.Body.String())
	}
	var alerts []models.AutoInspectionAlert
	if err := json.Unmarshal(alertsRR.Body.Bytes(), &alerts); err != nil {
		t.Fatalf("decodificar alertas: %v", err)
	}
	if len(alerts) != 1 || alerts[0].NextDueYear != 2023 {
		t.Errorf("alertas inesperadas: %+v", alerts)
	}
}

func TestAutoAttachmentHandlerUploadDownloadDelete(t *testing.T) {
	api := newAutoRecordAPI(t)
	maintenance := api.createMaintenance(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "factura-demo.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(testAutoRecordPDF)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/autos/1/maintenances/1/attachments", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.AddCookie(api.cookie)
	uploadReq.SetPathValue("id", strconv.FormatInt(api.autoID, 10))
	uploadReq.SetPathValue("maintenance_id", strconv.FormatInt(maintenance.ID, 10))
	uploadRR := httptest.NewRecorder()
	AuthMiddleware(api.auth)(http.HandlerFunc(api.attachment.UploadMaintenanceAttachment)).ServeHTTP(uploadRR, uploadReq)
	if uploadRR.Code != http.StatusCreated {
		t.Fatalf("upload: esperado 201, got %d: %s", uploadRR.Code, uploadRR.Body.String())
	}
	var att models.AutoAttachment
	if err := json.Unmarshal(uploadRR.Body.Bytes(), &att); err != nil {
		t.Fatalf("decodificar adjunto: %v", err)
	}

	downloadReq := httptest.NewRequest(http.MethodGet, "/api/autos/1/maintenances/1/attachments/1", nil)
	downloadReq.AddCookie(api.cookie)
	downloadReq.SetPathValue("id", strconv.FormatInt(api.autoID, 10))
	downloadReq.SetPathValue("maintenance_id", strconv.FormatInt(maintenance.ID, 10))
	downloadReq.SetPathValue("attachment_id", strconv.FormatInt(att.ID, 10))
	downloadRR := httptest.NewRecorder()
	AuthMiddleware(api.auth)(http.HandlerFunc(api.attachment.DownloadMaintenanceAttachment)).ServeHTTP(downloadRR, downloadReq)
	if downloadRR.Code != http.StatusOK {
		t.Fatalf("download: esperado 200, got %d: %s", downloadRR.Code, downloadRR.Body.String())
	}
	if got := downloadRR.Header().Get("Content-Type"); got != "application/pdf" {
		t.Errorf("content-type: esperado application/pdf, got %q", got)
	}
	if got := downloadRR.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("nosniff: got %q", got)
	}
	if !bytes.HasPrefix(downloadRR.Body.Bytes(), []byte("%PDF-")) {
		t.Errorf("el cuerpo descargado no es un PDF")
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/autos/1/maintenances/1/attachments/1", nil)
	deleteReq.AddCookie(api.cookie)
	deleteReq.SetPathValue("id", strconv.FormatInt(api.autoID, 10))
	deleteReq.SetPathValue("maintenance_id", strconv.FormatInt(maintenance.ID, 10))
	deleteReq.SetPathValue("attachment_id", strconv.FormatInt(att.ID, 10))
	deleteRR := httptest.NewRecorder()
	AuthMiddleware(api.auth)(http.HandlerFunc(api.attachment.DeleteMaintenanceAttachment)).ServeHTTP(deleteRR, deleteReq)
	if deleteRR.Code != http.StatusOK {
		t.Fatalf("delete: esperado 200, got %d: %s", deleteRR.Code, deleteRR.Body.String())
	}
}

func TestAutoAttachmentHandlerRechazaNoPDF(t *testing.T) {
	api := newAutoRecordAPI(t)
	maintenance := api.createMaintenance(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "factura-demo.pdf")
	_, _ = part.Write([]byte("no es un pdf"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/autos/1/maintenances/1/attachments", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(api.cookie)
	req.SetPathValue("id", strconv.FormatInt(api.autoID, 10))
	req.SetPathValue("maintenance_id", strconv.FormatInt(maintenance.ID, 10))
	rr := httptest.NewRecorder()
	AuthMiddleware(api.auth)(http.HandlerFunc(api.attachment.UploadMaintenanceAttachment)).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAutoInspectionHandlerRechazaCategoriaIncorrecta(t *testing.T) {
	api := newAutoRecordAPI(t)

	body := fmt.Sprintf(`{
		"performed_at":"2022-01-10","inspection_year":2022,"institution_id":%d,
		"result":"approved"
	}`, api.workshopID)
	req := httptest.NewRequest(http.MethodPost, "/api/autos/"+strconv.FormatInt(api.autoID, 10)+"/inspections", bytes.NewBufferString(body))
	req.AddCookie(api.cookie)
	req.SetPathValue("id", strconv.FormatInt(api.autoID, 10))
	rr := httptest.NewRecorder()
	AuthMiddleware(api.auth)(http.HandlerFunc(api.inspection.CreateInspection)).ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, got %d: %s", rr.Code, rr.Body.String())
	}
	var response errorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decodificar error: %v", err)
	}
	if response.Error != "institution_category" {
		t.Errorf("código de error: esperado institution_category, got %q", response.Error)
	}
}
