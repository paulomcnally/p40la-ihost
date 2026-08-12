package services

import (
	"context"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/config"
	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newTestAuth(t *testing.T) *AuthService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := &config.Config{
		BcryptCost:      10,
		SessionDuration: 1 * time.Hour,
		SecureCookie:    false,
	}
	return NewAuthService(storage.NewUserStorage(database), storage.NewSettingsStorage(database), cfg)
}

func TestCreateFirstUser(t *testing.T) {
	auth := newTestAuth(t)
	ctx := context.Background()

	user, cookie, err := auth.CreateFirstUser(ctx, "admin@example.com", "Password123", "Password123")
	if err != nil {
		t.Fatalf("crear primer usuario: %v", err)
	}
	if user.Email != "admin@example.com" {
		t.Errorf("email esperado admin@example.com, got %s", user.Email)
	}
	if cookie == nil || cookie.Value == "" {
		t.Error("se esperaba cookie de sesión")
	}

	completed, _ := auth.IsSetupComplete(ctx)
	if !completed {
		t.Error("setup debería estar completo")
	}

	// No debe permitir segundo usuario.
	if _, _, err := auth.CreateFirstUser(ctx, "otro@example.com", "Password123", "Password123"); err == nil {
		t.Error("debería rechazar segundo usuario")
	}
}

func TestCreateFirstUserValidation(t *testing.T) {
	auth := newTestAuth(t)
	ctx := context.Background()

	cases := []struct {
		name     string
		email    string
		password string
		confirm  string
	}{
		{"email vacío", "", "Password123", "Password123"},
		{"email inválido", "no-es-email", "Password123", "Password123"},
		{"contraseña corta", "a@b.com", "Pass1", "Pass1"},
		{"sin número", "a@b.com", "Password", "Password"},
		{"sin letra", "a@b.com", "12345678", "12345678"},
		{"confirmación distinta", "a@b.com", "Password123", "Password124"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := auth.CreateFirstUser(ctx, tc.email, tc.password, tc.confirm); err == nil {
				t.Errorf("%s: se esperaba error", tc.name)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	auth := newTestAuth(t)
	ctx := context.Background()

	if _, _, err := auth.CreateFirstUser(ctx, "admin@example.com", "Password123", "Password123"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	user, cookie, err := auth.Login(ctx, "admin@example.com", "Password123", false)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if user.Email != "admin@example.com" {
		t.Errorf("email esperado admin@example.com, got %s", user.Email)
	}
	if cookie == nil {
		t.Error("se esperaba cookie")
	}

	if _, _, err := auth.Login(ctx, "admin@example.com", "mal", false); err == nil {
		t.Error("login con contraseña incorrecta debería fallar")
	}
	if _, _, err := auth.Login(ctx, "otro@example.com", "Password123", false); err == nil {
		t.Error("login con email inexistente debería fallar")
	}
}

func TestSession(t *testing.T) {
	auth := newTestAuth(t)
	ctx := context.Background()

	if _, _, err := auth.CreateFirstUser(ctx, "admin@example.com", "Password123", "Password123"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, cookie, err := auth.Login(ctx, "admin@example.com", "Password123", false)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	user, err := auth.ValidateSession(ctx, cookie.Value)
	if err != nil {
		t.Fatalf("validar sesión: %v", err)
	}
	if user.Email != "admin@example.com" {
		t.Errorf("email esperado admin@example.com, got %s", user.Email)
	}

	// Cookie inválida.
	if _, err := auth.ValidateSession(ctx, "invalid"); err == nil {
		t.Error("cookie inválida debería fallar")
	}

	// Logout limpia cookie.
	logoutCookie := auth.Logout()
	if logoutCookie.MaxAge != -1 {
		t.Errorf("logout cookie MaxAge esperado -1, got %d", logoutCookie.MaxAge)
	}
}
