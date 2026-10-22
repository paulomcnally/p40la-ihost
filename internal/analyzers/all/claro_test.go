package all

import (
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/analyzers"
)

const residentialText = `EMPRESA NICARAGÜENSE DE TELECOMUNICACIONES, S.A.
No. de factura: A0000000000
NOMBRE: CLIENTE DEMO EJEMPLO
CÉDULA: 0000000000000X
ID CLIENTE: 0000000
NÚMERO DE CONTRATO: 0000000
PERÍODO FACTURADO FECHA DE EMISIÓN CICLO MES FACTURADO FECHA LÍMITE DE PAGO
06/JUL/2026 - 05/AGO/2026 07/AGO/2026 56 AGO/2026 04/SEP/2026
TOTAL FACTURA COMPRAS A PLAZOS TOTAL MES OTROS CARGOS SALDO PENDIENTE VALOR RECLAMO TOTAL A PAGAR
C$100.00 C$200.00 C$300.00 C$0.00 C$300.00 C$0.00 C$999.99
PRODUCTOS Y SERVICIOS FACTURADOS DE ESTE MES
5831162-CLARO TV C$ 10.00
5831160-TURBONETT FIJO C$ 20.00
IVA FACTURADO C$ 5.00
Total Factura C$ 100.00`

const mobileText = `EMPRESA NICARAGÜENSE DE TELECOMUNICACIONES, S.A.
No. de factura: FAC0000000000000
NOMBRE: CLIENTE DEMO EJEMPLO
CÉDULA: 0000000000000X
ID CLIENTE: 7075990
TELÉFONO TITULAR NO.: 5550-0100
PERÍODO FACTURADO FECHA DE EMISIÓN FECHA LÍMITE DE PAGO FECHA DE ACREDITACIÓN
21/Jun/2026 - 20/Jul/2026 22/07/2026 17/08/2026 21/07/2026
TOTAL FACTURA VENTA EN PLAZOS TOTAL MES SALDO A FAVOR SALDO PENDIENTE AJUSTE TOTAL A PAGAR
C$111.11 C$222.22 C$888.88 C$0.00 C$0.00 C$0.00 C$888.88
PRODUCTOS Y SERVICIOS FACTURADOS DE ESTE MES
Cargo basico mensual C$ 10.00
Cargo mensual C$ 2.00
Renta mensual por Datos GPRS C$ 20.00
IVA FACTURADO C$ 5.00
Total Factura C$ 111.11`

func TestClaroAnalyzer_Info(t *testing.T) {
	a := &ClaroAnalyzer{}
	info := a.Info()
	if info.ID != "claro" {
		t.Errorf("esperado ID 'claro', obtenido '%s'", info.ID)
	}
	if info.Name != "Claro Nicaragua" {
		t.Errorf("esperado Name 'Claro Nicaragua', obtenido '%s'", info.Name)
	}
}

func TestDetectFormat_Residential(t *testing.T) {
	got := detectFormat(residentialText)
	if got != "residential" {
		t.Errorf("esperado 'residential', obtenido '%s'", got)
	}
}

func TestDetectFormat_Mobile(t *testing.T) {
	got := detectFormat(mobileText)
	if got != "mobile" {
		t.Errorf("esperado 'mobile', obtenido '%s'", got)
	}
}

func TestDetectFormat_Unknown(t *testing.T) {
	got := detectFormat("texto sin factura")
	if got != "" {
		t.Errorf("esperado '', obtenido '%s'", got)
	}
}

func TestParseMonth(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"AGO", 8}, {"ago", 8}, {"JUL", 7}, {"SEP", 9},
		{"ENE", 1}, {"DIC", 12}, {"JAN", 1}, {"DEC", 12},
	}
	for _, tt := range tests {
		got := parseMonth(tt.input)
		if got != tt.want {
			t.Errorf("parseMonth(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"C$999.99", 999.99},
		{"C$888.88", 888.88},
		{"C$100.00", 100.00},
		{"C$111.11", 111.11},
		{"100.00", 100.00},
	}
	for _, tt := range tests {
		got, err := parseAmount(tt.input)
		if err != nil {
			t.Errorf("parseAmount(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseAmount(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestParseResidential(t *testing.T) {
	result, err := parseResidential(residentialText)
	if err != nil {
		t.Fatalf("parseResidential error: %v", err)
	}
	if result.InvoiceNumber != "A0000000000" {
		t.Errorf("InvoiceNumber = %q, want %q", result.InvoiceNumber, "A0000000000")
	}
	if result.Amount != 999.99 {
		t.Errorf("Amount = %f, want %f", result.Amount, 999.99)
	}
	if result.Year != 2026 {
		t.Errorf("Year = %d, want %d", result.Year, 2026)
	}
	if result.Month != 8 {
		t.Errorf("Month = %d, want %d", result.Month, 8)
	}
	if result.DueDate == nil {
		t.Error("DueDate is nil")
	} else if result.DueDate.Month() != 9 || result.DueDate.Day() != 4 {
		t.Errorf("DueDate = %v, want 2026-09-04", result.DueDate)
	}
}

func TestParseMobile(t *testing.T) {
	result, err := parseMobile(mobileText)
	if err != nil {
		t.Fatalf("parseMobile error: %v", err)
	}
	if result.InvoiceNumber != "FAC0000000000000" {
		t.Errorf("InvoiceNumber = %q, want %q", result.InvoiceNumber, "FAC0000000000000")
	}
	if result.Amount != 888.88 {
		t.Errorf("Amount = %f, want %f", result.Amount, 888.88)
	}
	if result.Year != 2026 {
		t.Errorf("Year = %d, want %d", result.Year, 2026)
	}
	if result.Month != 7 {
		t.Errorf("Month = %d, want %d", result.Month, 7)
	}
	if result.DueDate == nil {
		t.Error("DueDate is nil")
	} else if result.DueDate.Month() != 8 || result.DueDate.Day() != 17 {
		t.Errorf("DueDate = %v, want 2026-08-17", result.DueDate)
	}
}

func TestClaroAnalyzer_Registered(t *testing.T) {
	a, ok := analyzers.Get("claro")
	if !ok {
		t.Fatal("analyzer 'claro' no está registrado")
	}
	info := a.Info()
	if info.Name != "Claro Nicaragua" {
		t.Errorf("Name = %q, want %q", info.Name, "Claro Nicaragua")
	}
}

func TestParseDateDMY(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"04/SEP/2026", "2026-09-04"},
		{"17/08/2026", "2026-08-17"},
		{"21/Jun/2026", "2026-06-21"},
	}
	for _, tt := range tests {
		got, err := parseDateDMY(tt.input)
		if err != nil {
			t.Errorf("parseDateDMY(%q) error: %v", tt.input, err)
			continue
		}
		if got.Format("2006-01-02") != tt.want {
			t.Errorf("parseDateDMY(%q) = %s, want %s", tt.input, got.Format("2006-01-02"), tt.want)
		}
	}
}

func TestParseResidential_MissingFields(t *testing.T) {
	_, err := parseResidential("texto sin datos de factura")
	if err == nil {
		t.Error("esperado error por campos faltantes")
	}
	if !strings.Contains(err.Error(), "campos obligatorios") {
		t.Errorf("error inesperado: %v", err)
	}
}

func TestParseMobile_MissingFields(t *testing.T) {
	_, err := parseMobile("texto sin datos de factura")
	if err == nil {
		t.Error("esperado error por campos faltantes")
	}
	if !strings.Contains(err.Error(), "campos obligatorios") {
		t.Errorf("error inesperado: %v", err)
	}
}
