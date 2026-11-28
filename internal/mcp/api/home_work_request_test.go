package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

func newWorkRequestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := New(config.Config{BaseURL: server.URL, APIKey: "demo-token", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}
	return client
}

func TestHomeWorkRequestClientReads(t *testing.T) {
	var listQuery, detailPath string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/homes/3/workrequests", func(w http.ResponseWriter, r *http.Request) {
		listQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HomeWorkRequestList{
			Items:        []HomeWorkRequest{{ID: 10, HomeID: 3, Title: "Filtración demo", Status: "reported", AttachmentCount: 2}},
			PendingCount: 1,
			Truncated:    false,
		})
	})
	mux.HandleFunc("GET /api/homes/3/workrequests/10", func(w http.ResponseWriter, r *http.Request) {
		detailPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HomeWorkRequest{
			ID: 10, HomeID: 3, Title: "Filtración demo", Status: "reported",
			Attachments: []HomeWorkRequestAttachment{{ID: 1, WorkRequestID: 10, MediaType: "image", FileName: "foto-demo.jpg"}},
		})
	})
	client := newWorkRequestClient(t, mux)

	list, err := client.ListHomeWorkRequests(context.Background(), 3, HomeWorkRequestFilter{Status: "reported", Category: "repair"})
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(list.Items) != 1 || list.PendingCount != 1 || list.Items[0].AttachmentCount != 2 {
		t.Fatalf("listado inesperado: %+v", list)
	}
	for _, fragment := range []string{"status=reported", "category=repair"} {
		if !strings.Contains(listQuery, fragment) {
			t.Fatalf("query %q no contiene %q", listQuery, fragment)
		}
	}

	detail, err := client.GetHomeWorkRequest(context.Background(), 3, 10)
	if err != nil {
		t.Fatalf("detalle: %v", err)
	}
	if detailPath != "/api/homes/3/workrequests/10" || len(detail.Attachments) != 1 || detail.Attachments[0].FileName != "foto-demo.jpg" {
		t.Fatalf("detalle inesperado: path=%q %+v", detailPath, detail)
	}
}

func TestHomeWorkRequestClientWrites(t *testing.T) {
	var sources, paths []string
	var bodies []HomeWorkRequestRequest
	var convertBody HomeWorkRequestConvertRequest
	var statusBody HomeWorkRequestStatusRequest

	writeJSON := func(w http.ResponseWriter, status int, value any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(value)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/homes/3/workrequests", func(w http.ResponseWriter, r *http.Request) {
		sources = append(sources, r.Header.Get("X-P40LA-Source"))
		paths = append(paths, r.URL.Path)
		var body HomeWorkRequestRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body create: %v", err)
		}
		bodies = append(bodies, body)
		writeJSON(w, http.StatusCreated, HomeWorkRequest{ID: 10, HomeID: 3, Title: body.Title, Status: "reported"})
	})
	mux.HandleFunc("PUT /api/homes/3/workrequests/10", func(w http.ResponseWriter, r *http.Request) {
		sources = append(sources, r.Header.Get("X-P40LA-Source"))
		paths = append(paths, r.URL.Path)
		var body HomeWorkRequestRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body update: %v", err)
		}
		bodies = append(bodies, body)
		writeJSON(w, http.StatusOK, HomeWorkRequest{ID: 10, HomeID: 3, Title: body.Title, Status: "reported"})
	})
	mux.HandleFunc("POST /api/homes/3/workrequests/10/convert", func(w http.ResponseWriter, r *http.Request) {
		sources = append(sources, r.Header.Get("X-P40LA-Source"))
		paths = append(paths, r.URL.Path)
		if err := json.NewDecoder(r.Body).Decode(&convertBody); err != nil {
			t.Errorf("body convert: %v", err)
		}
		writeJSON(w, http.StatusOK, HomeWorkRequestConvertResult{
			MaintenanceID: 42,
			WorkRequest:   &HomeWorkRequest{ID: 10, HomeID: 3, Status: "converted"},
		})
	})
	mux.HandleFunc("POST /api/homes/3/workrequests/10/status", func(w http.ResponseWriter, r *http.Request) {
		sources = append(sources, r.Header.Get("X-P40LA-Source"))
		paths = append(paths, r.URL.Path)
		if err := json.NewDecoder(r.Body).Decode(&statusBody); err != nil {
			t.Errorf("body status: %v", err)
		}
		writeJSON(w, http.StatusOK, HomeWorkRequest{ID: 10, HomeID: 3, Status: statusBody.Status})
	})
	client := newWorkRequestClient(t, mux)

	request := HomeWorkRequestRequest{
		Title: "Agregar canaleta demo", Description: "Para el agua de lluvia",
		Category: "improvement", Priority: "normal", ReportedAt: "2026-11-25",
	}
	created, err := client.CreateHomeWorkRequest(context.Background(), 3, request)
	if err != nil || created.ID != 10 {
		t.Fatalf("crear: %v %+v", err, created)
	}
	updated, err := client.UpdateHomeWorkRequest(context.Background(), 3, 10, request)
	if err != nil || updated.Title != request.Title {
		t.Fatalf("editar: %v %+v", err, updated)
	}
	converted, err := client.ConvertHomeWorkRequest(context.Background(), 3, 10, HomeWorkRequestConvertRequest{Kind: "preventive", Area: "roof"})
	if err != nil || converted.MaintenanceID != 42 || converted.WorkRequest.Status != "converted" {
		t.Fatalf("convertir: %v %+v", err, converted)
	}
	status, err := client.SetHomeWorkRequestStatus(context.Background(), 3, 10, HomeWorkRequestStatusRequest{Status: "dismissed"})
	if err != nil || status.Status != "dismissed" {
		t.Fatalf("estado: %v %+v", err, status)
	}

	if len(sources) != 4 {
		t.Fatalf("llamadas de escritura = %d", len(sources))
	}
	for _, source := range sources {
		if source != "mcp" {
			t.Fatalf("source = %q", source)
		}
	}
	wantPaths := []string{
		"/api/homes/3/workrequests",
		"/api/homes/3/workrequests/10",
		"/api/homes/3/workrequests/10/convert",
		"/api/homes/3/workrequests/10/status",
	}
	for index, path := range wantPaths {
		if paths[index] != path {
			t.Fatalf("path %d = %q, want %q", index, paths[index], path)
		}
	}
	if len(bodies) != 2 || bodies[0].Title != request.Title || bodies[1].Category != request.Category {
		t.Fatalf("bodies = %+v", bodies)
	}
	if convertBody.Kind != "preventive" || convertBody.Area != "roof" || statusBody.Status != "dismissed" {
		t.Fatalf("convert/status inesperados: %+v %+v", convertBody, statusBody)
	}
}

func TestHomeWorkRequestClientEvidence(t *testing.T) {
	var sources []string
	var linkBody HomeWorkRequestUploadLinkRequest
	var uploadedName string
	var uploadedData []byte
	deleted := false

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/homes/3/workrequests/10/upload-link", func(w http.ResponseWriter, r *http.Request) {
		sources = append(sources, r.Header.Get("X-P40LA-Source"))
		if err := json.NewDecoder(r.Body).Decode(&linkBody); err != nil {
			t.Errorf("body link: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(HomeWorkRequestUploadLink{
			Token: "demo-token", Path: "/api/work-request-uploads", ExpiresAt: "2026-11-27T12:10:00Z",
			MaxBytes: 25 << 20, MaxFiles: 2, RemainingSlots: 8,
		})
	})
	mux.HandleFunc("POST /api/homes/3/workrequests/10/attachments/evidence", func(w http.ResponseWriter, r *http.Request) {
		sources = append(sources, r.Header.Get("X-P40LA-Source"))
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("form file: %v", err)
			return
		}
		defer file.Close()
		uploadedName = header.Filename
		uploadedData, _ = io.ReadAll(file)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(HomeWorkRequestEvidenceResult{
			Attachment: HomeWorkRequestAttachment{ID: 21, WorkRequestID: 10, FileName: header.Filename, SHA256: "demo"},
		})
	})
	mux.HandleFunc("DELETE /api/homes/3/workrequests/10/attachments/21", func(w http.ResponseWriter, r *http.Request) {
		sources = append(sources, r.Header.Get("X-P40LA-Source"))
		deleted = true
		w.WriteHeader(http.StatusNoContent)
	})
	client := newWorkRequestClient(t, mux)

	link, err := client.CreateHomeWorkRequestUploadLink(context.Background(), 3, 10, HomeWorkRequestUploadLinkRequest{Count: 2, TTLMinutes: 10})
	if err != nil || link.Token != "demo-token" || link.Path != "/api/work-request-uploads" {
		t.Fatalf("link: %v %+v", err, link)
	}
	if linkBody.Count != 2 || linkBody.TTLMinutes != 10 {
		t.Fatalf("body del link = %+v", linkBody)
	}

	result, err := client.AddHomeWorkRequestEvidence(context.Background(), 3, 10, "foto-demo.jpg", []byte("demo"))
	if err != nil || result.Attachment.ID != 21 || result.Attachment.SHA256 != "demo" {
		t.Fatalf("evidencia: %v %+v", err, result)
	}
	if uploadedName != "foto-demo.jpg" || string(uploadedData) != "demo" {
		t.Fatalf("archivo enviado = %q %q", uploadedName, string(uploadedData))
	}

	if err := client.DeleteHomeWorkRequestAttachment(context.Background(), 3, 10, 21); err != nil || !deleted {
		t.Fatalf("borrar: %v deleted=%v", err, deleted)
	}
	if len(sources) != 3 {
		t.Fatalf("llamadas = %d", len(sources))
	}
	for _, source := range sources {
		if source != "mcp" {
			t.Fatalf("source = %q", source)
		}
	}
}

func TestHomeWorkRequestClientAPIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/homes/3/workrequests/99", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"work_request_not_found","message":"Reporte no encontrado"}`))
	})
	client := newWorkRequestClient(t, mux)
	_, err := client.GetHomeWorkRequest(context.Background(), 3, 99)
	if err == nil {
		t.Fatal("esperaba error")
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != "work_request_not_found" || apiErr.Status != http.StatusNotFound {
		t.Fatalf("error inesperado: %#v", err)
	}
}
