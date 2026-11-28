package workrequest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type evidenceFile struct {
	name string
	data []byte
}

func uploadWithToken(t *testing.T, url, token string, files ...evidenceFile) (*http.Response, map[string]any) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, file := range files {
		part, err := writer.CreateFormFile("file", file.name)
		if err != nil {
			t.Fatalf("crear parte: %v", err)
		}
		if _, err := part.Write(file.data); err != nil {
			t.Fatalf("escribir parte: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cerrar multipart: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatalf("request de carga: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("carga: %v", err)
	}
	defer resp.Body.Close()
	var payload map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&payload)
	return resp, payload
}

func requestJSONWithHeaders(t *testing.T, method, url string, body any, headers map[string]string) (*http.Response, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
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

func TestClientRequestIDIdempotencyHTTP(t *testing.T) {
	server, database, homeID := newTestServer(t)
	base := fmt.Sprintf("%s/api/homes/%d/workrequests", server.URL, homeID)
	body := map[string]any{
		"title": "Filtración demo idempotente", "description": "Gotea cuando llueve",
		"category": "repair", "priority": "high", "reported_at": "2026-11-26",
		"client_request_id": "demo-msg-1",
	}

	resp, first := requestJSON(t, http.MethodPost, base, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("primer alta = %d (%v)", resp.StatusCode, first)
	}
	resp, second := requestJSON(t, http.MethodPost, base, body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("segunda alta = %d (%v)", resp.StatusCode, second)
	}
	if first["id"] != second["id"] || second["deduplicated"] != true {
		t.Fatalf("idempotencia = %v / %v", first, second)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM home_work_requests WHERE client_request_id = 'demo-msg-1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("filas con client_request_id = %d", count)
	}

	resp, invalid := requestJSON(t, http.MethodPost, base, map[string]any{
		"title": "Reporte demo", "reported_at": "2026-11-26", "client_request_id": "espacio inválido",
	})
	if resp.StatusCode != http.StatusBadRequest || invalid["error"] != "invalid_client_request_id" {
		t.Fatalf("client_request_id inválido = %d (%v)", resp.StatusCode, invalid)
	}
}

func TestEvidenceUploadLinkFlowAndDeduplication(t *testing.T) {
	server, database, homeID := newTestServer(t)
	base := fmt.Sprintf("%s/api/homes/%d/workrequests", server.URL, homeID)
	_, created := requestJSON(t, http.MethodPost, base, validInput())
	id := int64(created["id"].(float64))

	resp, link := requestJSON(t, http.MethodPost, fmt.Sprintf("%s/%d/upload-link", base, id), UploadLinkRequest{Count: 3})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload-link = %d (%v)", resp.StatusCode, link)
	}
	token, _ := link["token"].(string)
	if token == "" || link["max_files"] != float64(3) || link["remaining_slots"] != float64(10) {
		t.Fatalf("link inesperado: %v", link)
	}
	uploadURL := server.URL + link["path"].(string)

	resp, result := uploadWithToken(t, uploadURL, token,
		evidenceFile{"foto-demo.jpg", jpegData()},
		evidenceFile{"video-demo.mp4", mp4Data()},
	)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("carga = %d (%v)", resp.StatusCode, result)
	}
	if result["added"] != float64(2) || result["duplicates"] != float64(0) || result["attachment_count"] != float64(2) {
		t.Fatalf("resultado de carga inesperado: %v", result)
	}

	resp, duplicate := uploadWithToken(t, uploadURL, token,
		evidenceFile{"foto-demo.jpg", jpegData()},
		evidenceFile{"video-demo.mp4", mp4Data()},
	)
	if resp.StatusCode != http.StatusOK || duplicate["duplicates"] != float64(2) || duplicate["added"] != float64(0) {
		t.Fatalf("duplicados inesperados = %d (%v)", resp.StatusCode, duplicate)
	}
	var rows int
	if err := database.QueryRow(`SELECT COUNT(*) FROM home_work_request_attachments WHERE work_request_id = ?`, id).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 2 {
		t.Fatalf("adjuntos en DB = %d", rows)
	}

	_, detail := requestJSON(t, http.MethodGet, fmt.Sprintf("%s/%d", base, id), nil)
	attachments := detail["attachments"].([]any)
	if len(attachments) != 2 {
		t.Fatalf("detalle con adjuntos = %v", attachments)
	}
	for _, raw := range attachments {
		attachment := raw.(map[string]any)
		if attachment["sha256"] == "" {
			t.Fatalf("adjunto sin sha256: %v", attachment)
		}
	}

	firstAttachmentID := int64(attachments[0].(map[string]any)["id"].(float64))
	resp, _ = requestJSONWithHeaders(t, http.MethodDelete,
		fmt.Sprintf("%s/%d/attachments/%d", base, id, firstAttachmentID), nil,
		map[string]string{"X-P40LA-Source": "mcp"})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("borrado MCP = %d", resp.StatusCode)
	}
}

func TestEvidenceTokenScopesAndStatus(t *testing.T) {
	server, _, homeID := newTestServer(t)
	base := fmt.Sprintf("%s/api/homes/%d/workrequests", server.URL, homeID)
	_, first := requestJSON(t, http.MethodPost, base, validInput())
	firstID := int64(first["id"].(float64))
	_, second := requestJSON(t, http.MethodPost, base, validInput())
	secondID := int64(second["id"].(float64))

	_, link := requestJSON(t, http.MethodPost, fmt.Sprintf("%s/%d/upload-link", base, firstID), UploadLinkRequest{Count: 2})
	token := link["token"].(string)
	uploadURL := server.URL + link["path"].(string)

	resp, result := uploadWithToken(t, uploadURL, token,
		evidenceFile{"foto-1.jpg", jpegData()},
		evidenceFile{"foto-2.jpg", append(jpegData(), []byte("segunda")...)},
		evidenceFile{"foto-3.jpg", append(jpegData(), []byte("tercera")...)},
	)
	if resp.StatusCode != http.StatusOK || result["added"] != float64(2) || result["rejected"] != float64(1) {
		t.Fatalf("lote con cupo = %d (%v)", resp.StatusCode, result)
	}
	results := result["results"].([]any)
	if results[2].(map[string]any)["error"] != "upload_token_exhausted" {
		t.Fatalf("tercer archivo = %v", results[2])
	}

	resp, exhausted := uploadWithToken(t, uploadURL, token, evidenceFile{"foto-4.jpg", append(jpegData(), []byte("cuarta")...)})
	if resp.StatusCode != http.StatusConflict || exhausted["error"] != "upload_token_exhausted" {
		t.Fatalf("token agotado = %d (%v)", resp.StatusCode, exhausted)
	}

	resp, tampered := uploadWithToken(t, uploadURL, token+"x", evidenceFile{"foto-5.jpg", jpegData()})
	if resp.StatusCode != http.StatusUnauthorized || tampered["error"] != "invalid_token" {
		t.Fatalf("token alterado = %d (%v)", resp.StatusCode, tampered)
	}

	_, otherLink := requestJSON(t, http.MethodPost, fmt.Sprintf("%s/%d/upload-link", base, secondID), UploadLinkRequest{Count: 1})
	foreignToken := otherLink["token"].(string)
	_, converted := requestJSON(t, http.MethodPost, fmt.Sprintf("%s/%d/convert", base, secondID), ConvertInput{})
	if converted["work_request"] == nil {
		t.Fatalf("conversión inesperada: %v", converted)
	}
	resp, invalidStatus := uploadWithToken(t, uploadURL, foreignToken, evidenceFile{"foto-6.jpg", jpegData()})
	if resp.StatusCode != http.StatusConflict || invalidStatus["error"] != "invalid_status" {
		t.Fatalf("reporte convertido = %d (%v)", resp.StatusCode, invalidStatus)
	}
}

func TestEvidenceRejectionsAndLimits(t *testing.T) {
	server, database, homeID := newTestServer(t)
	base := fmt.Sprintf("%s/api/homes/%d/workrequests", server.URL, homeID)
	_, created := requestJSON(t, http.MethodPost, base, validInput())
	id := int64(created["id"].(float64))
	_, link := requestJSON(t, http.MethodPost, fmt.Sprintf("%s/%d/upload-link", base, id), UploadLinkRequest{Count: 10})
	token := link["token"].(string)
	uploadURL := server.URL + link["path"].(string)

	resp, result := uploadWithToken(t, uploadURL, token, evidenceFile{"nota-demo.txt", []byte("texto demo")})
	if resp.StatusCode != http.StatusOK || result["rejected"] != float64(1) {
		t.Fatalf("archivo inválido = %d (%v)", resp.StatusCode, result)
	}
	results := result["results"].([]any)
	if results[0].(map[string]any)["error"] != "invalid_file" {
		t.Fatalf("resultado inválido = %v", results[0])
	}
	var rows int
	if err := database.QueryRow(`SELECT COUNT(*) FROM home_work_request_attachments WHERE work_request_id = ?`, id).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Fatalf("se guardó un archivo rechazado: %d", rows)
	}

	for index := 0; index < MaxAttachmentsPerRequest; index++ {
		_, attachment := uploadFile(t, fmt.Sprintf("%s/%d/attachments", base, id), MediaImage, "foto-demo.jpg", jpegData())
		if attachment["id"] == nil {
			t.Fatalf("adjunto %d no creado: %v", index, attachment)
		}
	}
	resp, limit := requestJSON(t, http.MethodPost, fmt.Sprintf("%s/%d/upload-link", base, id), UploadLinkRequest{})
	if resp.StatusCode != http.StatusConflict || limit["error"] != "attachment_limit" {
		t.Fatalf("límite de link = %d (%v)", resp.StatusCode, limit)
	}
}

func TestMigration0063ApplyAndRollback(t *testing.T) {
	_, database, homeID, _ := newTestService(t)
	res, err := database.Exec(`INSERT INTO home_work_requests (home_id, title, reported_at) VALUES (?, 'Reporte demo', '2026-11-26')`, homeID)
	if err != nil {
		t.Fatalf("insertar reporte: %v", err)
	}
	reportID, _ := res.LastInsertId()
	if _, err := database.Exec(`INSERT INTO home_work_request_attachments (work_request_id, media_type, file_name, size_bytes) VALUES (?, 'image', 'foto-demo.jpg', 10)`, reportID); err != nil {
		t.Fatalf("insertar adjunto: %v", err)
	}

	dir, err := filepath.Abs("../../../../migrations")
	if err != nil {
		t.Fatalf("ruta de migraciones: %v", err)
	}
	down, err := os.ReadFile(filepath.Join(dir, "0063_work_request_evidence_idempotency.down.sql"))
	if err != nil {
		t.Fatalf("leer down: %v", err)
	}
	up, err := os.ReadFile(filepath.Join(dir, "0063_work_request_evidence_idempotency.up.sql"))
	if err != nil {
		t.Fatalf("leer up: %v", err)
	}
	if _, err := database.Exec(string(down)); err != nil {
		t.Fatalf("aplicar down: %v", err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM home_work_requests WHERE id = ?`, reportID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("el reporte se perdió al revertir: count=%d err=%v", count, err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM home_work_request_attachments WHERE work_request_id = ?`, reportID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("el adjunto se perdió al revertir: count=%d err=%v", count, err)
	}
	if _, err := database.Exec(string(up)); err != nil {
		t.Fatalf("reaplicar up: %v", err)
	}
	if _, err := database.Exec(`UPDATE home_work_requests SET client_request_id = 'demo-1' WHERE id = ?`, reportID); err != nil {
		t.Fatalf("usar client_request_id tras reaplicar: %v", err)
	}
	if _, err := database.Exec(`UPDATE home_work_request_attachments SET sha256 = 'demo-hash' WHERE work_request_id = ?`, reportID); err != nil {
		t.Fatalf("usar sha256 tras reaplicar: %v", err)
	}
}

func TestUploadRateLimiter(t *testing.T) {
	now := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	limiter := newUploadRateLimiter(2, time.Minute)
	limiter.now = func() time.Time { return now }
	if allowed, _ := limiter.allow("demo"); !allowed {
		t.Fatal("primera request debería permitirse")
	}
	if allowed, _ := limiter.allow("demo"); !allowed {
		t.Fatal("segunda request debería permitirse")
	}
	if allowed, retry := limiter.allow("demo"); allowed || retry <= 0 {
		t.Fatalf("tercera request = allowed=%v retry=%v", allowed, retry)
	}
	now = now.Add(time.Minute)
	if allowed, _ := limiter.allow("demo"); !allowed {
		t.Fatal("la ventana debería reiniciarse")
	}
}

func TestUploadLinkDisabledWithoutSecret(t *testing.T) {
	service, _, homeID, _ := newTestService(t)
	ctx := context.Background()
	item, err := service.Create(ctx, homeID, CreateInput{Input: validInput()})
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	service.SetUploadConfig(UploadConfig{})
	_, err = service.MintUploadLink(ctx, homeID, item.ID, UploadLinkRequest{})
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Code != "upload_link_disabled" {
		t.Fatalf("sin secreto = %v", err)
	}
}

func TestUploadRateLimitHTTP(t *testing.T) {
	server, _, _ := newTestServer(t)
	for index := 0; index < 30; index++ {
		resp, _ := uploadWithToken(t, server.URL+UploadLinkPath, "demo-token-rate-limit")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("request %d = %d", index, resp.StatusCode)
		}
	}
	resp, _ := uploadWithToken(t, server.URL+UploadLinkPath, "demo-token-rate-limit")
	if resp.StatusCode != http.StatusTooManyRequests || resp.Header.Get("Retry-After") == "" {
		t.Fatalf("rate limit = %d retry=%q", resp.StatusCode, resp.Header.Get("Retry-After"))
	}
}

func TestUploadTokenExpiryAndTamper(t *testing.T) {
	service, _, homeID, _ := newTestService(t)
	ctx := context.Background()
	item, err := service.Create(ctx, homeID, CreateInput{Input: validInput()})
	if err != nil {
		t.Fatalf("crear: %v", err)
	}
	service.SetUploadConfig(UploadConfig{Secret: strings.Repeat("s", 32), TTL: time.Minute})
	link, err := service.MintUploadLink(ctx, homeID, item.ID, UploadLinkRequest{Count: 1})
	if err != nil {
		t.Fatalf("emitir link: %v", err)
	}
	if _, _, _, err := service.ValidateUploadToken(ctx, link.Token); err != nil {
		t.Fatalf("token válido: %v", err)
	}
	if _, _, _, err := service.ValidateUploadToken(ctx, link.Token+"x"); err == nil {
		t.Fatal("token alterado debería fallar")
	}
	service.SetClock(func() time.Time { return time.Date(2026, 11, 27, 12, 2, 0, 0, time.UTC) })
	if _, _, _, err := service.ValidateUploadToken(ctx, link.Token); err == nil {
		t.Fatal("token expirado debería fallar")
	}
}
