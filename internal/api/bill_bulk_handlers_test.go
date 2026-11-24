package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newBillBulkHandlerTestEnv(t *testing.T) (*BillHandlers, func(string, ...any)) {
	t.Helper()
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

	billService := services.NewBillService(storage.NewBillStorage(database), storage.NewServiceStorage(database))
	billService.SetBillHistoryStorage(storage.NewBillHistoryStorage(database))
	billService.SetCurrencyStorage(storage.NewCurrencyStorage(database))
	return NewBillHandlers(billService), mustExec
}

func TestGenerateBillsHandler(t *testing.T) {
	handler, _ := newBillBulkHandlerTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/services/1/bills/generate",
		strings.NewReader(`{"from":"2026-01","to":"2026-02","dry_run":false}`))
	req.SetPathValue("service_id", "1")
	req.Header.Set(bulkSourceHeader, "mcp")
	rec := httptest.NewRecorder()

	handler.GenerateBills(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, quiero 200 (%s)", rec.Code, rec.Body.String())
	}
	var result services.GenerateBillsResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decodificar respuesta: %v", err)
	}
	if result.Created != 2 {
		t.Fatalf("created = %d, quiero 2", result.Created)
	}
}

func TestGenerateBillsHandlerErrors(t *testing.T) {
	handler, _ := newBillBulkHandlerTestEnv(t)

	cases := []struct {
		name       string
		serviceID  string
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "rango futuro",
			serviceID:  "1",
			body:       `{"from":"2026-01","to":"2026-12"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   services.BulkErrorInvalidRange,
		},
		{
			name:       "servicio inexistente",
			serviceID:  "999",
			body:       `{"from":"2026-01","to":"2026-01"}`,
			wantStatus: http.StatusNotFound,
			wantCode:   services.BulkErrorServiceNotFound,
		},
		{
			name:       "servicio inválido",
			serviceID:  "abc",
			body:       `{"from":"2026-01","to":"2026-01"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_service_id",
		},
		{
			name:       "origen inválido",
			serviceID:  "1",
			body:       `{"from":"2026-01","to":"2026-01"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   services.BulkErrorInvalidSource,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/services/"+tc.serviceID+"/bills/generate",
				strings.NewReader(tc.body))
			req.SetPathValue("service_id", tc.serviceID)
			if tc.name == "origen inválido" {
				req.Header.Set(bulkSourceHeader, "otro")
			}
			rec := httptest.NewRecorder()
			handler.GenerateBills(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, quiero %d (%s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			var payload struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decodificar error: %v", err)
			}
			if payload.Error != tc.wantCode {
				t.Fatalf("error = %q, quiero %q", payload.Error, tc.wantCode)
			}
		})
	}
}

func TestPayBillsHandler(t *testing.T) {
	handler, _ := newBillBulkHandlerTestEnv(t)
	paidAt := time.Now().Format("2006-01-02")

	generate := httptest.NewRequest(http.MethodPost, "/api/services/1/bills/generate",
		strings.NewReader(`{"from":"2026-01","to":"2026-02"}`))
	generate.SetPathValue("service_id", "1")
	generateRec := httptest.NewRecorder()
	handler.GenerateBills(generateRec, generate)
	if generateRec.Code != http.StatusOK {
		t.Fatalf("preparar facturas: %d (%s)", generateRec.Code, generateRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/services/1/bills/pay",
		strings.NewReader(`{"month":"2026-01","paid_at":"`+paidAt+`"}`))
	req.SetPathValue("service_id", "1")
	rec := httptest.NewRecorder()
	handler.PayBills(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, quiero 200 (%s)", rec.Code, rec.Body.String())
	}
	var result services.PayBillsResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decodificar respuesta: %v", err)
	}
	if result.Paid != 1 {
		t.Fatalf("paid = %d, quiero 1", result.Paid)
	}
}
