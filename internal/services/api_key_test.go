package services

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newAPIKeyTestService(t *testing.T) *APIKeyService {
	t.Helper()
	database, err := db.OpenDB(filepath.Join(t.TempDir(), "test.db"), "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewAPIKeyService(storage.NewSystemSettingsStorage(database))
}

func TestAPIKeyGenerateValidateRotateRevoke(t *testing.T) {
	ctx := context.Background()
	service := newAPIKeyTestService(t)

	status, err := service.Status(ctx)
	if err != nil {
		t.Fatalf("status inicial: %v", err)
	}
	if status.Configured {
		t.Fatal("no debería haber key configurada")
	}

	key, created, err := service.Regenerate(ctx)
	if err != nil {
		t.Fatalf("regenerar: %v", err)
	}
	if !strings.HasPrefix(key, APIKeyPrefix) || len(key) < 20 {
		t.Fatalf("key inválida: %q", key)
	}
	if !created.Configured || created.Prefix == "" {
		t.Fatalf("estado creado inválido: %+v", created)
	}
	if strings.Contains(created.Prefix, key[len(created.Prefix):]) {
		t.Fatal("el prefijo no debe contener el resto de la key")
	}

	valid, err := service.Validate(ctx, key)
	if err != nil || !valid {
		t.Fatalf("validar key nueva: valid=%v err=%v", valid, err)
	}
	valid, err = service.Validate(ctx, key+"x")
	if err != nil || valid {
		t.Fatalf("una key incorrecta no debe validar: valid=%v err=%v", valid, err)
	}
	valid, _ = service.Validate(ctx, "")
	if valid {
		t.Fatal("una key vacía no debe validar")
	}

	status, err = service.Status(ctx)
	if err != nil {
		t.Fatalf("status con key: %v", err)
	}
	if !status.Configured || status.Prefix != created.Prefix || status.CreatedAt == nil {
		t.Fatalf("status inesperado: %+v", status)
	}

	newKey, _, err := service.Regenerate(ctx)
	if err != nil {
		t.Fatalf("rotar: %v", err)
	}
	if newKey == key {
		t.Fatal("la rotación debe generar una key distinta")
	}
	if valid, _ := service.Validate(ctx, key); valid {
		t.Fatal("la key anterior debe quedar inválida")
	}
	if valid, _ := service.Validate(ctx, newKey); !valid {
		t.Fatal("la key nueva debe validar")
	}

	if err := service.Revoke(ctx); err != nil {
		t.Fatalf("revocar: %v", err)
	}
	if valid, _ := service.Validate(ctx, newKey); valid {
		t.Fatal("la key revocada no debe validar")
	}
	status, _ = service.Status(ctx)
	if status.Configured {
		t.Fatal("status debe quedar no configurado")
	}
}

func TestAPIKeyNotStoredInPlaintext(t *testing.T) {
	ctx := context.Background()
	database, err := db.OpenDB(filepath.Join(t.TempDir(), "test.db"), "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	settings := storage.NewSystemSettingsStorage(database)
	service := NewAPIKeyService(settings)

	key, _, err := service.Regenerate(ctx)
	if err != nil {
		t.Fatalf("regenerar: %v", err)
	}
	raw, err := settings.Get(ctx, mcpAPIKeyHashSetting)
	if err != nil {
		t.Fatalf("leer hash: %v", err)
	}
	if raw == "" || raw == key || strings.Contains(raw, key) {
		t.Fatalf("la key no debe guardarse en claro: %q", raw)
	}
	var count int
	if err := database.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM system_settings WHERE value = ?`, key).Scan(&count); err != nil {
		t.Fatalf("buscar key en claro: %v", err)
	}
	if count != 0 {
		t.Fatal("la key completa no debe persistirse")
	}
}
