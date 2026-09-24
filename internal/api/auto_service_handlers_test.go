package api

import (
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

func newListServiceAutosTestHandlers(t *testing.T) (*AutoServiceHandlers, *services.AuthService, int64, *http.Cookie) {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	h := NewAutoServiceHandlers(services.NewAutoServiceService(storage.NewAutoServiceStorage(database)))

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
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Seguro Autos', 'ASSA', 2, 'monthly', 67.39, 1, 'insurance', 'fixed', 1)`)
	mustExec("INSERT INTO autos (year, model, brand, color, placa) VALUES (2026, 'Hilux', 'Toyota', 'Blanco', 'P123ABC')")
	mustExec(`INSERT INTO auto_services (auto_id, service_id, coverage_type, policy_number, certificate, insurer_number)
		VALUES (1, 1, 'full_cover', '02B 128265', '1', '1800 9911')`)

	return h, auth, 1, cookie
}

func TestListServiceAutosHandler(t *testing.T) {
	h, auth, serviceID, cookie := newListServiceAutosTestHandlers(t)
	handler := AuthMiddleware(auth)(http.HandlerFunc(h.ListServiceAutos))
	id := strconv.FormatInt(serviceID, 10)

	// Sin sesión → 401
	req := httptest.NewRequest(http.MethodGet, "/api/services/"+id+"/autos", nil)
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión esperaba 401, got %d", rr.Code)
	}

	// Con sesión → 200 con autos del servicio
	req = httptest.NewRequest(http.MethodGet, "/api/services/"+id+"/autos", nil)
	req.AddCookie(cookie)
	req.SetPathValue("id", id)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("con sesión esperaba 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var autos []models.ServiceAutoDetail
	if err := json.Unmarshal(rr.Body.Bytes(), &autos); err != nil {
		t.Fatalf("decodificar respuesta: %v", err)
	}
	if len(autos) != 1 {
		t.Fatalf("se esperaba 1 auto, got %d", len(autos))
	}
	if autos[0].Brand != "Toyota" || autos[0].PolicyNumber != "02B 128265" {
		t.Errorf("auto inesperado: %+v", autos[0])
	}

	// Servicio sin asociaciones → 200 con lista vacía
	req = httptest.NewRequest(http.MethodGet, "/api/services/999/autos", nil)
	req.AddCookie(cookie)
	req.SetPathValue("id", "999")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("servicio sin autos esperaba 200, got %d", rr.Code)
	}
	var empty []models.ServiceAutoDetail
	if err := json.Unmarshal(rr.Body.Bytes(), &empty); err != nil {
		t.Fatalf("decodificar respuesta vacía: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("se esperaba lista vacía, got %d", len(empty))
	}

	// ID inválido → 400
	req = httptest.NewRequest(http.MethodGet, "/api/services/abc/autos", nil)
	req.AddCookie(cookie)
	req.SetPathValue("id", "abc")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("id inválido esperaba 400, got %d", rr.Code)
	}
}
