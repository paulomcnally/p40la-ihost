package services

import (
	"context"
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func TestRenderTemplate_ReplacesPlaceholders(t *testing.T) {
	svc := NewEmailService(nil)
	html := svc.RenderTemplate(context.Background(), "Mi Título", "<p>Contenido de prueba</p>")

	if !strings.Contains(html, "Mi Título") {
		t.Errorf("título no reemplazado en template")
	}
	if !strings.Contains(html, "<p>Contenido de prueba</p>") {
		t.Errorf("contenido no reemplazado en template")
	}
	if strings.Contains(html, "{{TITLE}}") || strings.Contains(html, "{{CONTENT}}") || strings.Contains(html, "{{DATE}}") {
		t.Errorf("quedan placeholders sin reemplazar")
	}
	if !strings.Contains(html, "P40LA") {
		t.Errorf("template no contiene header P40LA")
	}
}

func TestRenderTemplate_UniqueAppearance(t *testing.T) {
	svc := NewEmailService(nil)
	a := svc.RenderTemplate(context.Background(), "Título A", "Contenido A")
	b := svc.RenderTemplate(context.Background(), "Título B", "Contenido B")

	// La estructura debe ser la misma (mismo header y footer); solo título/contenido varían.
	withoutA := strings.ReplaceAll(a, "Título A", "T")
	withoutA = strings.ReplaceAll(withoutA, "Contenido A", "C")
	withoutB := strings.ReplaceAll(b, "Título B", "T")
	withoutB = strings.ReplaceAll(withoutB, "Contenido B", "C")

	if withoutA != withoutB {
		t.Errorf("la apariencia del template no es la misma entre emails:\nA: %s\nB: %s", withoutA, withoutB)
	}
}

func TestRenderTemplate_InjectPalette(t *testing.T) {
	html := renderTemplateWithPalette("T", "C", EmailPalette{
		Primary:    "#ff0000",
		Background: "#000000",
		Card:       "#111111",
		Text:       "#eeeeee",
		Muted:      "#999999",
		Border:     "#333333",
	})

	for _, want := range []string{"#ff0000", "#000000", "#111111", "#eeeeee", "#999999", "#333333"} {
		if !strings.Contains(html, want) {
			t.Errorf("paleta no inyectada: falta %s", want)
		}
	}
	if strings.Contains(html, "{{COLOR_") {
		t.Errorf("quedan placeholders de color sin reemplazar")
	}
}

func TestRenderTemplate_DefaultPaletteWhenUnconfigured(t *testing.T) {
	svc := NewEmailService(nil)
	html := svc.RenderTemplate(context.Background(), "T", "C")

	d := DefaultEmailPalette()
	if !strings.Contains(html, d.Primary) || !strings.Contains(html, d.Background) ||
		!strings.Contains(html, d.Card) || !strings.Contains(html, d.Text) ||
		!strings.Contains(html, d.Muted) || !strings.Contains(html, d.Border) {
		t.Errorf("sin paleta configurada deben usarse los defaults: %s", html)
	}
}

func TestRenderTemplate_ResponsiveTemplate(t *testing.T) {
	svc := NewEmailService(nil)
	html := svc.RenderTemplate(context.Background(), "T", "C")

	if !strings.Contains(html, `max-width:600px`) {
		t.Errorf("tabla contenedora no es fluida (falta max-width): %s", html)
	}
	if !strings.Contains(html, "@media screen and (max-width: 480px)") {
		t.Errorf("falta media query mobile: %s", html)
	}
	// Las cards de datos NO deben depender de clases del <style> (Gmail
	// recorta <style>): se estilan inline (ADR-001).
	if strings.Contains(html, "p40la-table") || strings.Contains(html, "data-label") {
		t.Errorf("la plantilla no debe contener reglas/clases de tablas de datos (cards inline): %s", html)
	}
}

func TestStripHTML(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"<p>Hola <b>mundo</b></p>", "Hola mundo"},
		{"<table><tr><td>A</td></tr></table>", "A"},
		{"  Hola    mundo  ", "Hola mundo"},
	}
	for _, c := range cases {
		if got := stripHTML(c.in); got != c.want {
			t.Errorf("stripHTML(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildMessage_ContainsHTMLAndPlain(t *testing.T) {
	cfg := &models.SMTPConfig{
		Host:      "smtp.example.com",
		Port:      587,
		User:      "smtp-user",
		Password:  "smtp-password",
		FromEmail: "alerts@example.com",
		FromName:  "P40LA",
	}
	msg, err := buildMessage(cfg, []string{"to@example.com"}, "Asunto", "<p>Hola <b>mundo</b></p>")
	if err != nil {
		t.Fatalf("buildMessage error: %v", err)
	}
	s := string(msg)
	if !strings.Contains(s, "From: P40LA <alerts@example.com>") {
		t.Errorf("From header incorrecto: %q", s)
	}
	if !strings.Contains(s, "To: to@example.com") {
		t.Errorf("To header incorrecto")
	}
	if !strings.Contains(s, "Subject: Asunto") {
		t.Errorf("Subject incorrecto")
	}
	if !strings.Contains(s, "text/plain") {
		t.Errorf("falta parte text/plain")
	}
	if !strings.Contains(s, "text/html") {
		t.Errorf("falta parte text/html")
	}
	if strings.Contains(s, "smtp-password") || strings.Contains(s, "smtp-user") {
		t.Errorf("el mensaje MIME no debe contener credenciales")
	}
}
