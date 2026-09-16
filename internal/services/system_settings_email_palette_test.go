package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newPaletteSettingsService(t *testing.T) *SystemSettingsService {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
}

func TestEmailPalette_DefaultsWhenUnconfigured(t *testing.T) {
	s := newPaletteSettingsService(t)
	ctx := context.Background()

	p, err := s.GetEmailPalette(ctx)
	if err != nil {
		t.Fatalf("get palette: %v", err)
	}
	d := DefaultEmailPalette()
	if p != d {
		t.Errorf("paleta sin configurar debe ser la default, got %+v", p)
	}
}

func TestEmailPalette_SetAndGet(t *testing.T) {
	s := newPaletteSettingsService(t)
	ctx := context.Background()

	want := EmailPalette{
		Primary:    "#aa0000",
		Background: "#101010",
		Card:       "#202020",
		Text:       "#f0f0f0",
		Muted:      "#808080",
		Border:     "#303030",
	}
	if err := s.SetEmailPalette(ctx, want); err != nil {
		t.Fatalf("set palette: %v", err)
	}

	got, err := s.GetEmailPalette(ctx)
	if err != nil {
		t.Fatalf("get palette: %v", err)
	}
	if got != want {
		t.Errorf("paleta guardada incorrecta, got %+v want %+v", got, want)
	}
}

func TestEmailPalette_PartialUpdate(t *testing.T) {
	s := newPaletteSettingsService(t)
	ctx := context.Background()

	if err := s.SetEmailPalette(ctx, EmailPalette{Primary: "#123456"}); err != nil {
		t.Fatalf("set partial: %v", err)
	}

	got, err := s.GetEmailPalette(ctx)
	if err != nil {
		t.Fatalf("get palette: %v", err)
	}
	if got.Primary != "#123456" {
		t.Errorf("primary no actualizado: %q", got.Primary)
	}
	d := DefaultEmailPalette()
	if got.Background != d.Background || got.Card != d.Card || got.Text != d.Text ||
		got.Muted != d.Muted || got.Border != d.Border {
		t.Errorf("los campos no enviados deben seguir en default, got %+v", got)
	}
}

func TestEmailPalette_InvalidHexRejected(t *testing.T) {
	s := newPaletteSettingsService(t)
	ctx := context.Background()

	err := s.SetEmailPalette(ctx, EmailPalette{Primary: "azul"})
	if err == nil {
		t.Fatal("color inválido no rechazado")
	}

	err = s.SetEmailPalette(ctx, EmailPalette{Primary: "#fff"})
	if err == nil {
		t.Fatal("hex de 3 dígitos no rechazado")
	}

	err = s.SetEmailPalette(ctx, EmailPalette{Primary: "#gggggg"})
	if err == nil {
		t.Fatal("hex con caracteres no válidos no rechazado")
	}

	// Nada debe persistir tras el error.
	got, err := s.GetEmailPalette(ctx)
	if err != nil {
		t.Fatalf("get palette: %v", err)
	}
	if got.Primary != DefaultEmailPalette().Primary {
		t.Errorf("color inválido no debe persistirse, primary = %q", got.Primary)
	}
}

func TestEmailPalette_Reset(t *testing.T) {
	s := newPaletteSettingsService(t)
	ctx := context.Background()

	if err := s.SetEmailPalette(ctx, EmailPalette{Primary: "#aa0000", Border: "#303030"}); err != nil {
		t.Fatalf("set palette: %v", err)
	}

	if err := s.ResetEmailPalette(ctx); err != nil {
		t.Fatalf("reset palette: %v", err)
	}

	got, err := s.GetEmailPalette(ctx)
	if err != nil {
		t.Fatalf("get palette: %v", err)
	}
	if got != DefaultEmailPalette() {
		t.Errorf("tras reset debe volver a defaults, got %+v", got)
	}
}

func TestValidHexColor(t *testing.T) {
	valid := []string{"#007aff", "#FFFFFF", "#abcdef", "#AbCdEf"}
	for _, v := range valid {
		if !ValidHexColor(v) {
			t.Errorf("validHexColor(%q) = false, esperado true", v)
		}
	}
	invalid := []string{"", "007aff", "#fff", "#gggggg", "azul", "#007af", "#007afff"}
	for _, v := range invalid {
		if ValidHexColor(v) {
			t.Errorf("validHexColor(%q) = true, esperado false", v)
		}
	}
}