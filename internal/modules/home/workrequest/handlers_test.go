package workrequest

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	homemaintenance "github.com/paulomcnally/p40la-ihost/internal/modules/home/maintenance"
)

func newTestServer(t *testing.T) (*httptest.Server, *sql.DB, int64) {
	t.Helper()
	dir := t.TempDir()
	database, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "workrequests-http.db")+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("abrir db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	migrationsDir, err := filepath.Abs("../../../../migrations")
	if err != nil {
		t.Fatalf("ruta de migraciones: %v", err)
	}
	if err := db.Migrate(database, migrationsDir); err != nil {
		t.Fatalf("migrar: %v", err)
	}
	res, err := database.Exec(`INSERT INTO homes (name) VALUES ('Casa Demo')`)
	if err != nil {
		t.Fatalf("insertar vivienda: %v", err)
	}
	homeID, _ := res.LastInsertId()

	maintenanceModule := homemaintenance.New(database)
	maintenanceModule.Service().SetClock(func() time.Time { return time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC) })
	module := New(database, maintenanceModule.Service(), dir)
	module.Service().SetClock(func() time.Time { return time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC) })

	mux := http.NewServeMux()
	noAuth := func(next http.Handler) http.Handler { return next }
	maintenanceModule.RegisterRoutes(mux, noAuth)
	module.RegisterRoutes(mux, noAuth)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, database, homeID
}

func requestJSON(t *testing.T, method, url string, body any) (*http.Response, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	var payload map[string]any
	if resp.StatusCode != http.StatusNoContent {
		_ = json.NewDecoder(resp.Body).Decode(&payload)
	}
	return resp, payload
}

func uploadFile(t *testing.T, url, mediaType, fileName string, data []byte) (*http.Response, map[string]any) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("media_type", mediaType); err != nil {
		t.Fatalf("campo media_type: %v", err)
	}
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("crear archivo: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("escribir archivo: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cerrar multipart: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subir archivo: %v", err)
	}
	defer resp.Body.Close()
	var payload map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&payload)
	return resp, payload
}

func TestHandlersWorkRequestLifecycle(t *testing.T) {
	server, _, homeID := newTestServer(t)
	base := fmt.Sprintf("%s/api/homes/%d/workrequests", server.URL, homeID)

	resp, created := requestJSON(t, http.MethodPost, base, validInput())
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST status = %d (%v)", resp.StatusCode, created)
	}
	id := int64(created["id"].(float64))
	if created["status"] != StatusReported {
		t.Fatalf("estado = %v", created["status"])
	}

	resp, listed := requestJSON(t, http.MethodGet, base+"?status=reported&category=repair", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET list status = %d", resp.StatusCode)
	}
	items, _ := listed["items"].([]any)
	if len(items) != 1 || int(listed["pending_count"].(float64)) != 1 {
		t.Fatalf("listado = %v", listed)
	}

	resp, updated := requestJSON(t, http.MethodPut, fmt.Sprintf("%s/%d", base, id), Input{
		Title:       "Filtración revisada",
		Description: "Se revisó la filtración demo.",
		Category:    CategoryOther,
		Priority:    PriorityLow,
		ReportedAt:  "2026-11-25",
	})
	if resp.StatusCode != http.StatusOK || updated["status"] != StatusReported {
		t.Fatalf("PUT status = %d (%v)", resp.StatusCode, updated)
	}

	resp, attachment := uploadFile(t, fmt.Sprintf("%s/%d/attachments", base, id), MediaImage, "foto-demo.jpg", jpegData())
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload status = %d (%v)", resp.StatusCode, attachment)
	}
	attachmentID := int64(attachment["id"].(float64))
	if attachment["mime_type"] != "image/jpeg" {
		t.Fatalf("mime = %v", attachment["mime_type"])
	}

	detailURL := fmt.Sprintf("%s/%d/attachments/%d", base, id, attachmentID)
	req, _ := http.NewRequest(http.MethodGet, detailURL, nil)
	req.Header.Set("Range", "bytes=0-1")
	fileResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET adjunto: %v", err)
	}
	defer fileResp.Body.Close()
	if fileResp.StatusCode != http.StatusPartialContent {
		t.Fatalf("Range status = %d", fileResp.StatusCode)
	}
	if fileResp.Header.Get("Content-Type") != "image/jpeg" || fileResp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("headers = %v", fileResp.Header)
	}

	resp, _ = requestJSON(t, http.MethodDelete, detailURL, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE adjunto status = %d", resp.StatusCode)
	}

	resp, converted := requestJSON(t, http.MethodPost, fmt.Sprintf("%s/%d/convert", base, id), ConvertInput{Kind: "preventive", Area: "roof"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("convert status = %d (%v)", resp.StatusCode, converted)
	}
	maintenanceID := int64(converted["maintenance_id"].(float64))
	if maintenanceID == 0 {
		t.Fatalf("maintenance_id = %v", converted)
	}
	workRequest, _ := converted["work_request"].(map[string]any)
	if workRequest["status"] != StatusConverted {
		t.Fatalf("reporte convertido = %v", workRequest)
	}

	resp, _ = requestJSON(t, http.MethodGet, fmt.Sprintf("%s/api/homes/%d/maintenances/%d", server.URL, homeID, maintenanceID), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reparación creada no encontrada: %d", resp.StatusCode)
	}

	resp, conflict := requestJSON(t, http.MethodPost, fmt.Sprintf("%s/%d/convert", base, id), ConvertInput{})
	if resp.StatusCode != http.StatusConflict || conflict["error"] != "already_converted" {
		t.Fatalf("segunda conversión = %d (%v)", resp.StatusCode, conflict)
	}

	resp, invalid := uploadFile(t, fmt.Sprintf("%s/%d/attachments", base, id), MediaImage, "nota.txt", []byte("texto demo"))
	if resp.StatusCode != http.StatusBadRequest || invalid["error"] != "invalid_file" {
		t.Fatalf("upload inválido = %d (%v)", resp.StatusCode, invalid)
	}

	resp, _ = requestJSON(t, http.MethodDelete, fmt.Sprintf("%s/%d", base, id), nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE reporte status = %d", resp.StatusCode)
	}
	resp, missing := requestJSON(t, http.MethodGet, fmt.Sprintf("%s/%d", base, id), nil)
	if resp.StatusCode != http.StatusNotFound || missing["error"] != "work_request_not_found" {
		t.Fatalf("reporte eliminado = %d (%v)", resp.StatusCode, missing)
	}
}

func TestHandlersStatusAndOwnership(t *testing.T) {
	server, database, homeID := newTestServer(t)
	base := fmt.Sprintf("%s/api/homes/%d/workrequests", server.URL, homeID)

	_, created := requestJSON(t, http.MethodPost, base, validInput())
	id := int64(created["id"].(float64))

	resp, dismissed := requestJSON(t, http.MethodPost, fmt.Sprintf("%s/%d/status", base, id), StatusInput{Status: StatusDismissed})
	if resp.StatusCode != http.StatusOK || dismissed["status"] != StatusDismissed {
		t.Fatalf("descartar = %d (%v)", resp.StatusCode, dismissed)
	}
	resp, reactivated := requestJSON(t, http.MethodPost, fmt.Sprintf("%s/%d/status", base, id), StatusInput{Status: StatusReported})
	if resp.StatusCode != http.StatusOK || reactivated["status"] != StatusReported {
		t.Fatalf("reactivar = %d (%v)", resp.StatusCode, reactivated)
	}

	res, err := database.Exec(`INSERT INTO homes (name) VALUES ('Casa Demo Dos')`)
	if err != nil {
		t.Fatalf("insertar vivienda: %v", err)
	}
	otherHomeID, _ := res.LastInsertId()
	resp, missing := requestJSON(t, http.MethodGet, fmt.Sprintf("%s/api/homes/%d/workrequests/%d", server.URL, otherHomeID, id), nil)
	if resp.StatusCode != http.StatusNotFound || missing["error"] != "work_request_not_found" {
		t.Fatalf("reporte de otra vivienda = %d (%v)", resp.StatusCode, missing)
	}

	resp, invalid := requestJSON(t, http.MethodPost, fmt.Sprintf("%s/%d/status", base, id), StatusInput{Status: "converted"})
	if resp.StatusCode != http.StatusBadRequest || invalid["error"] != "invalid_status" {
		t.Fatalf("estado inválido = %d (%v)", resp.StatusCode, invalid)
	}
}

func TestHandlersUploadLimitAndSize(t *testing.T) {
	server, _, homeID := newTestServer(t)
	base := fmt.Sprintf("%s/api/homes/%d/workrequests", server.URL, homeID)

	_, created := requestJSON(t, http.MethodPost, base, validInput())
	id := int64(created["id"].(float64))

	for index := 0; index < MaxAttachmentsPerRequest; index++ {
		resp, _ := uploadFile(t, fmt.Sprintf("%s/%d/attachments", base, id), MediaImage, "foto-demo.jpg", jpegData())
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("adjunto %d status = %d", index, resp.StatusCode)
		}
	}
	resp, limit := uploadFile(t, fmt.Sprintf("%s/%d/attachments", base, id), MediaImage, "foto-demo.jpg", jpegData())
	if resp.StatusCode != http.StatusConflict || limit["error"] != "attachment_limit" {
		t.Fatalf("límite = %d (%v)", resp.StatusCode, limit)
	}

	_, other := requestJSON(t, http.MethodPost, base, validInput())
	otherID := int64(other["id"].(float64))
	big := append(jpegData(), bytes.Repeat([]byte("x"), MaxAttachmentBytes)...)
	resp, tooLarge := uploadFile(t, fmt.Sprintf("%s/%d/attachments", base, otherID), MediaImage, "grande.jpg", big)
	if resp.StatusCode != http.StatusRequestEntityTooLarge || tooLarge["error"] != "request_too_large" {
		t.Fatalf("tamaño = %d (%v)", resp.StatusCode, tooLarge)
	}
}

func TestModuleInfoAndRoutes(t *testing.T) {
	_, database, _ := newTestServer(t)
	maintenance := homemaintenance.NewService(homemaintenance.NewStorage(database))
	module := New(database, maintenance, t.TempDir())
	info := module.Info()
	if info.ID != ModuleID || info.Version != ModuleVersion {
		t.Fatalf("info = %+v", info)
	}
	if len(info.Migrations) != 1 || info.Migrations[0] != "0062_home_work_requests" {
		t.Fatalf("migraciones = %v", info.Migrations)
	}
	if !strings.HasPrefix(info.Version, "1.") {
		t.Fatalf("versión semver = %q", info.Version)
	}
}

func TestConversionDescriptionIsTruncated(t *testing.T) {
	description := strings.Repeat("d", maxDescriptionLen)
	item := &WorkRequest{Title: strings.Repeat("t", maxTitleLen), Description: description}
	got := convertedDescription(item)
	if len([]rune(got)) != maxMaintenanceDesc {
		t.Fatalf("descripción convertida = %d runas", len([]rune(got)))
	}
}
