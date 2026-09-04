package services

import (
	"strings"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func TestBuildBillCreatedEmail_FixedMonthly(t *testing.T) {
	svc := &models.Service{
		Name:        "Internet residencial",
		Institution: "Claro",
		BillingType: "fixed",
	}
	bill := &models.Bill{Month: 8, Year: 2026, Amount: 1500}

	title, html := buildBillCreatedEmail(svc, bill, "C$", DefaultCurrencyFormat())

	if title != "Nueva factura generada — Internet residencial" {
		t.Errorf("título incorrecto: %q", title)
	}
	checks := []string{
		"Hola,",
		"Agosto 2026",
		"C$1,500.00",
		"Este monto es <strong>fijo</strong>",
		"Institución:</strong> Claro",
		"Servicio:</strong> Internet residencial",
	}
	for _, c := range checks {
		if !strings.Contains(html, c) {
			t.Errorf("contenido falta %q", c)
		}
	}
}

func TestBuildBillCreatedEmail_VariableYearly(t *testing.T) {
	svc := &models.Service{
		Name:        "Seguro auto",
		Institution: "Seguros XYZ",
		BillingType: "variable",
	}
	bill := &models.Bill{Month: 0, Year: 2026, Amount: 300}

	title, html := buildBillCreatedEmail(svc, bill, "$", DefaultCurrencyFormat())

	if title != "Nueva factura generada — Seguro auto" {
		t.Errorf("título incorrecto: %q", title)
	}
	if !strings.Contains(html, "Año 2026") {
		t.Errorf("período anual no aparece")
	}
	if !strings.Contains(html, "variable (podés editarlo)") {
		t.Errorf("no indica monto variable editable")
	}
	if !strings.Contains(html, "$300.00") {
		t.Errorf("monto con símbolo no aparece")
	}
}

func TestBuildBillCreatedEmail_WithoutInstitution(t *testing.T) {
	svc := &models.Service{Name: "Luz", BillingType: "variable"}
	bill := &models.Bill{Month: 1, Year: 2026, Amount: 100}

	_, html := buildBillCreatedEmail(svc, bill, "", DefaultCurrencyFormat())

	if !strings.Contains(html, "Institución:</strong> —") {
		t.Errorf("institución ausente no se muestra como —")
	}
}

func TestFormatPeriod(t *testing.T) {
	cases := []struct {
		month, year int
		want        string
	}{
		{1, 2026, "Enero 2026"},
		{8, 2026, "Agosto 2026"},
		{12, 2026, "Diciembre 2026"},
		{0, 2026, "Año 2026"},
		{13, 2026, "13/2026"},
	}
	for _, c := range cases {
		if got := formatPeriod(c.month, c.year); got != c.want {
			t.Errorf("formatPeriod(%d, %d) = %q, want %q", c.month, c.year, got, c.want)
		}
	}
}

func TestFormatAmount(t *testing.T) {
	if got := formatAmount(1500, "C$", DefaultCurrencyFormat()); got != "C$1,500.00" {
		t.Errorf("formatAmount con símbolo = %q", got)
	}
	if got := formatAmount(300, "", DefaultCurrencyFormat()); got != "300.00" {
		t.Errorf("formatAmount sin símbolo = %q", got)
	}
}

func TestDaysSince(t *testing.T) {
	now := time.Now()
	cases := []struct {
		days int
		want int
	}{
		{0, 0},
		{1, 1},
		{15, 15},
		{30, 30},
	}
	for _, c := range cases {
		day := now.AddDate(0, 0, -c.days)
		if got := daysSince(day); got != c.want {
			t.Errorf("daysSince hace %d días = %d, want %d", c.days, got, c.want)
		}
	}
}

func TestRenderBillSummaryContent_GroupedByHome(t *testing.T) {
	now := time.Now()
	pending := []models.PendingBillDetail{
		{
			BillID: 1, ServiceName: "Internet", HomeID: 1, HomeName: "Casa Central",
			Institution: "Claro", Month: 8, Year: 2026, Amount: 1500,
			CurrencySymbol: "C$", CreatedAt: now.AddDate(0, 0, -3),
		},
		{
			BillID: 2, ServiceName: "Luz", HomeID: 2, HomeName: "Casa Playa",
			Institution: "", Month: 1, Year: 2026, Amount: 100,
			CurrencySymbol: "$", CreatedAt: now.AddDate(0, 0, -20),
		},
	}

	html := renderBillSummaryContent(pending, DefaultCurrencyFormat())

	if !strings.Contains(html, "2 facturas pendientes") {
		t.Errorf("no aparece el contador de pendientes")
	}
	if !strings.Contains(html, "Casa Central") || !strings.Contains(html, "Casa Playa") {
		t.Errorf("no aparecen las casas agrupadas")
	}
	if !strings.Contains(html, "Hace 3 días") {
		t.Errorf("badge de antigüedad 3 días no aparece")
	}
	if !strings.Contains(html, "Hace 20 días") {
		t.Errorf("badge de antigüedad 20 días no aparece")
	}
	if !strings.Contains(html, "C$1,500.00") || !strings.Contains(html, "$100.00") {
		t.Errorf("montos con moneda no aparecen")
	}
}

func TestRenderBillSummaryContent_Empty(t *testing.T) {
	html := renderBillSummaryContent(nil, DefaultCurrencyFormat())
	if !strings.Contains(html, "No hay facturas pendientes") {
		t.Errorf("contenido vacío incorrecto: %s", html)
	}
}

func TestRenderBillSummaryContent_EscapesHTML(t *testing.T) {
	now := time.Now()
	pending := []models.PendingBillDetail{
		{
			BillID: 1, ServiceName: "<script>alert(1)</script>", HomeID: 1, HomeName: "Casa <b>X</b>",
			Institution: "Inst & Co", Month: 8, Year: 2026, Amount: 1500,
			CurrencySymbol: "C$", CreatedAt: now,
		},
	}

	html := renderBillSummaryContent(pending, DefaultCurrencyFormat())

	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Errorf("HTML no escapado en servicio")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("servicio mal escapado")
	}
	if !strings.Contains(html, "Casa &lt;b&gt;X&lt;/b&gt;") {
		t.Errorf("casa mal escapada")
	}
	if !strings.Contains(html, "Inst &amp; Co") {
		t.Errorf("institución mal escapada")
	}
}

func TestAgeBadge(t *testing.T) {
	if !strings.Contains(ageBadge(0), "Hoy") {
		t.Errorf("badge hoy incorrecto")
	}
	if !strings.Contains(ageBadge(1), "Ayer") {
		t.Errorf("badge ayer incorrecto")
	}
	if !strings.Contains(ageBadge(10), "#ff9500") {
		t.Errorf("badge 2-15 días debe ser naranja")
	}
	if !strings.Contains(ageBadge(20), "#ff3b30") {
		t.Errorf("badge >15 días debe ser rojo")
	}
}
