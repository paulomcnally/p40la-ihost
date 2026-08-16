package services

import (
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func TestRenderTemplate_ReplacesPlaceholders(t *testing.T) {
	svc := NewEmailService(nil)
	html := svc.RenderTemplate("Mi Título", "<p>Contenido de prueba</p>")

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
	a := svc.RenderTemplate("Título A", "Contenido A")
	b := svc.RenderTemplate("Título B", "Contenido B")

	// La estructura debe ser la misma (mismo header y footer); solo título/contenido varían.
	withoutA := strings.ReplaceAll(a, "Título A", "T")
	withoutA = strings.ReplaceAll(withoutA, "Contenido A", "C")
	withoutB := strings.ReplaceAll(b, "Título B", "T")
	withoutB = strings.ReplaceAll(withoutB, "Contenido B", "C")

	if withoutA != withoutB {
		t.Errorf("la apariencia del template no es la misma entre emails:\nA: %s\nB: %s", withoutA, withoutB)
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
