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

func newAutoTestHandlers(t *testing.T) (*AutoHandlers, *services.AuthService, *http.Cookie) {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	h := NewAutoHandlers(services.NewAutoService(storage.NewAutoStorage(database)))

	cfg := &config.Config{BcryptCost: 10, SessionDuration: time.Hour, SecureCookie: false}
	auth := services.NewAuthService(storage.NewUserStorage(database), storage.NewSettingsStorage(database), cfg)
	_, cookie, err := auth.CreateFirstUser(ctx, "test@test.com", "Password123", "Password123")
	if err != nil {
		t.Fatalf("crear usuario: %v", err)
	}
	return h, auth, cookie
}

func TestCreateAutoHandlerCamposOpcionales(t *testing.T) {
	h, auth, cookie := newAutoTestHandlers(t)
	handler := AuthMiddleware(auth)(http.HandlerFunc(h.CreateAuto))

	body := `{"year":2026,"brand":"Toyota","model":"Corolla","color":"WHITE"}`
	req := httptest.NewRequest(http.MethodPost, "/api/autos", bytes.NewBufferString(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("esperado 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var auto models.Auto
	if err := json.Unmarshal(rr.Body.Bytes(), &auto); err != nil {
		t.Fatalf("decodificar respuesta: %v", err)
	}
	if auto.Color != "white" {
		t.Errorf("color normalizado: esperado white, got %q", auto.Color)
	}
	if auto.Motor != "" || auto.Chasis != "" || auto.VIN != "" || auto.Placa != "" {
		t.Errorf("campos opcionales deberían quedar vacíos: %+v", auto)
	}
}

func TestCreateAutoHandlerColorInvalido(t *testing.T) {
	h, auth, cookie := newAutoTestHandlers(t)
	handler := AuthMiddleware(auth)(http.HandlerFunc(h.CreateAuto))

	body := `{"year":2026,"brand":"Toyota","model":"Corolla","color":"blanco"}`
	req := httptest.NewRequest(http.MethodPost, "/api/autos", bytes.NewBufferString(body))
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUpdateAutoHandlerLimpiaOpcionales(t *testing.T) {
	h, auth, cookie := newAutoTestHandlers(t)
	ctx := context.Background()

	auto, err := h.service.Create(ctx, 2020, "Hilux", "Toyota", "black", "truck", "2ZR1", "CH1", "VIN1", "ABC123")
	if err != nil {
		t.Fatalf("crear auto: %v", err)
	}

	body := `{"year":2020,"brand":"Toyota","model":"Hilux","color":"silver","motor":"","chasis":"","vin":"","placa":""}`
	req := httptest.NewRequest(http.MethodPut, "/api/autos/"+strconv.FormatInt(auto.ID, 10), bytes.NewBufferString(body))
	req.AddCookie(cookie)
	req.SetPathValue("id", strconv.FormatInt(auto.ID, 10))
	rr := httptest.NewRecorder()
	AuthMiddleware(auth)(http.HandlerFunc(h.UpdateAuto)).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("esperado 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var updated models.Auto
	if err := json.Unmarshal(rr.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decodificar respuesta: %v", err)
	}
	if updated.Color != "silver" || updated.Motor != "" || updated.Chasis != "" || updated.VIN != "" || updated.Placa != "" {
		t.Errorf("auto actualizado inesperado: %+v", updated)
	}
}
