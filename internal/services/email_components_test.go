package services

import (
	"strings"
	"testing"
)

func TestRenderEmailCard_Basic(t *testing.T) {
	html := RenderEmailCard([]EmailField{
		{Label: "Institución", Value: "Claro"},
		{Label: "Monto", Value: "C$1,500.00"},
	}, DefaultEmailPalette())

	if !strings.Contains(html, `border-radius:12px`) {
		t.Errorf("card sin borde redondeado: %s", html)
	}
	if !strings.Contains(html, ">Institución</td>") || !strings.Contains(html, ">Claro</td>") {
		t.Errorf("par label:valor no renderizado: %s", html)
	}
	if !strings.Contains(html, `text-transform:uppercase`) {
		t.Errorf("label sin estilo uppercase: %s", html)
	}
	if !strings.Contains(html, `text-align:right`) {
		t.Errorf("valor sin alineación derecha: %s", html)
	}
}

func TestRenderEmailCard_Escapes(t *testing.T) {
	html := RenderEmailCard([]EmailField{
		{Label: "Servicio <x>", Value: `<script>alert(1)</script>`},
	}, DefaultEmailPalette())

	if strings.Contains(html, `<script>alert(1)</script>`) {
		t.Errorf("valor no escapado")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("valor mal escapado")
	}
	if !strings.Contains(html, "&lt;x&gt;") {
		t.Errorf("label no escapado")
	}
}

func TestRenderEmailCard_HTMLField(t *testing.T) {
	html := RenderEmailCard([]EmailField{
		{Label: "Antigüedad", Value: `<span style="background-color:#ff9500;">Hace 10 días</span>`, HTML: true},
	}, DefaultEmailPalette())

	if !strings.Contains(html, `<span style="background-color:#ff9500;">Hace 10 días</span>`) {
		t.Errorf("campo HTML no renderizado como HTML: %s", html)
	}
}

func TestRenderEmailCard_Bold(t *testing.T) {
	html := RenderEmailCard([]EmailField{
		{Label: "Total del día", Value: "C$1,500.00", Bold: true},
	}, DefaultEmailPalette())

	if !strings.Contains(html, `font-weight:700`) {
		t.Errorf("valor total sin negrita: %s", html)
	}
}

func TestRenderEmailCard_UsesPalette(t *testing.T) {
	p := DefaultEmailPalette()
	p.Border = "#334455"
	p.Text = "#112233"
	p.Muted = "#667788"
	html := RenderEmailCard([]EmailField{{Label: "A", Value: "B"}}, p)

	if !strings.Contains(html, "#334455") || !strings.Contains(html, "#112233") || !strings.Contains(html, "#667788") {
		t.Errorf("paleta no aplicada en card: %s", html)
	}
}