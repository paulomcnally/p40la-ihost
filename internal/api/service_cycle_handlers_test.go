package api

import (
	"bytes"
	"context"
	"encoding/json"
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

func newServiceCycleTestHandlers(t *testing.T) (*ServiceCycleHandlers, *services.AuthService, int64, *http.Cookie) {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	svcSvc := services.NewServiceCycleService(
		storage.NewServiceStorage(database),
		storage.NewServiceCycleStorage(database),
	)
	h := NewServiceCycleHandlers(svcSvc)

	cfg := &config.Config{BcryptCost: 10, SessionDuration: time.Hour, SecureCookie: false}
	auth := services.NewAuthService(storage.NewUserStorage(database), storage.NewSettingsStorage(database), cfg)
	_, cookie, err := auth.CreateFirstUser(ctx, "test@test.com", "Password123", "Password123")
	if err != nil {
		t.Fatalf("crear usuario: %v", err)
	}

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("insertar dato de prueba (%q): %v", query, err)
		}
	}

	mustExec("INSERT INTO homes (name) VALUES ('Casa')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring, start_date, end_date, webhook_uuid)
		VALUES (1, 'Seguro', 'ASSA', 2, 'monthly', 67.39, 1, 'insurance', 'fixed', 1, '2025-10-20', '2026-10-19', 'wbh-0001')`)

	return h, auth, 1, cookie
}

func TestRenewServiceHandler(t *testing.T) {
	h, auth, serviceID, cookie := newServiceCycleTestHandlers(t)
	handler := AuthMiddleware(auth)(http.HandlerFunc(h.RenewService))
	id := strconv.FormatInt(serviceID, 10)

	body := `{"start_date":"2026-10-20","end_date":"2027-10-19","suggested_amount":70}`
	url := "/api/services/" + id + "/renew"

	// Sin sesión → 401
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión esperaba 401, got %d", rr.Code)
	}

	// Con sesión → 200 con service + cycle
	req = httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	req.AddCookie(cookie)
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("renew esperaba 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var res struct {
		Service *models.Service      `json:"service"`
		Cycle   *models.ServiceCycle `json:"cycle"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("decodificar respuesta: %v", err)
	}
	if res.Cycle == nil || res.Cycle.Sequence != 1 {
		t.Fatalf("se esperaba ciclo 1, got %+v", res.Cycle)
	}
	if res.Service == nil || res.Service.StartDate == nil || *res.Service.StartDate != "2026-10-20" {
		t.Fatalf("servicio renovado inesperado: %+v", res.Service)
	}

	// Fechas inválidas → 400
	bad := `{"start_date":"2027-10-19","end_date":"2026-10-20"}`
	req = httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(bad))
	req.AddCookie(cookie)
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("fechas inválidas esperaba 400, got %d", rr.Code)
	}

	// Servicio inexistente → 404
	req = httptest.NewRequest(http.MethodPost, "/api/services/999/renew", bytes.NewBufferString(body))
	req.AddCookie(cookie)
	req.SetPathValue("id", "999")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("servicio inexistente esperaba 404, got %d", rr.Code)
	}
}

func TestListServiceCyclesHandler(t *testing.T) {
	h, auth, serviceID, cookie := newServiceCycleTestHandlers(t)
	handler := AuthMiddleware(auth)(http.HandlerFunc(h.ListServiceCycles))
	id := strconv.FormatInt(serviceID, 10)

	// Renovar primero para tener un ciclo.
	svcHandler := AuthMiddleware(auth)(http.HandlerFunc(h.RenewService))
	req := httptest.NewRequest(http.MethodPost, "/api/services/"+id+"/renew", bytes.NewBufferString(`{"start_date":"2026-10-20","end_date":"2027-10-19"}`))
	req.AddCookie(cookie)
	req.SetPathValue("id", id)
	svcHandler.ServeHTTP(httptest.NewRecorder(), req)

	// Sin sesión → 401
	req = httptest.NewRequest(http.MethodGet, "/api/services/"+id+"/cycles", nil)
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión esperaba 401, got %d", rr.Code)
	}

	// Con sesión → 200 con 1 ciclo
	req = httptest.NewRequest(http.MethodGet, "/api/services/"+id+"/cycles", nil)
	req.AddCookie(cookie)
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list cycles esperaba 200, got %d", rr.Code)
	}
	var cycles []models.ServiceCycle
	if err := json.Unmarshal(rr.Body.Bytes(), &cycles); err != nil {
		t.Fatalf("decodificar: %v", err)
	}
	if len(cycles) != 1 || cycles[0].Sequence != 1 {
		t.Fatalf("se esperaba 1 ciclo, got %+v", cycles)
	}
}
