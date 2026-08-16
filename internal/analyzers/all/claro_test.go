package all

import (
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/analyzers"
)

const residentialText = `EMPRESA NICARAGÜENSE DE TELECOMUNICACIONES, S.A.
No. de factura: A0061376765
NOMBRE: PAULO ANTONIO MCNALLY ZAMBRANA
CÉDULA: 0012801880001H
ID CLIENTE: 2457168
NÚMERO DE CONTRATO: 2371975
PERÍODO FACTURADO FECHA DE EMISIÓN CICLO MES FACTURADO FECHA LÍMITE DE PAGO
06/JUL/2026 - 05/AGO/2026 07/AGO/2026 56 AGO/2026 04/SEP/2026
TOTAL FACTURA COMPRAS A PLAZOS TOTAL MES OTROS CARGOS SALDO PENDIENTE VALOR RECLAMO TOTAL A PAGAR
C$1,821.60 C$1,723.00 C$3,544.60 C$0.00 C$3,544.60 C$0.00 C$7,089.20
PRODUCTOS Y SERVICIOS FACTURADOS DE ESTE MES
5831162-CLARO TV C$ 329.62
5831160-TURBONETT FIJO C$ 1,254.38
IVA FACTURADO C$ 237.60
Total Factura C$ 1,821.60`

const mobileText = `EMPRESA NICARAGÜENSE DE TELECOMUNICACIONES, S.A.
No. de factura: FAC0257566732026
NOMBRE: PAULO ANTONIO MCNALLY ZAMBRANA
CÉDULA: 0012801880001H
ID CLIENTE: 7075990
TELÉFONO TITULAR NO.: 8412-6107
PERÍODO FACTURADO FECHA DE EMISIÓN FECHA LÍMITE DE PAGO FECHA DE ACREDITACIÓN
21/Jun/2026 - 20/Jul/2026 22/07/2026 17/08/2026 21/07/2026
TOTAL FACTURA VENTA EN PLAZOS TOTAL MES SALDO A FAVOR SALDO PENDIENTE AJUSTE TOTAL A PAGAR
C$841.93 C$134.04 C$975.97 C$0.00 C$0.00 C$0.00 C$975.97
PRODUCTOS Y SERVICIOS FACTURADOS DE ESTE MES
Cargo basico mensual C$ 338.77
Cargo mensual C$ 36.62
Renta mensual por Datos GPRS C$ 356.72
IVA FACTURADO C$ 109.82
Total Factura C$ 841.93`

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
		{"C$7,089.20", 7089.20},
		{"C$975.97", 975.97},
		{"C$1,821.60", 1821.60},
		{"C$841.93", 841.93},
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
	if result.InvoiceNumber != "A0061376765" {
		t.Errorf("InvoiceNumber = %q, want %q", result.InvoiceNumber, "A0061376765")
	}
	if result.Amount != 7089.20 {
		t.Errorf("Amount = %f, want %f", result.Amount, 7089.20)
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
	if result.InvoiceNumber != "FAC0257566732026" {
		t.Errorf("InvoiceNumber = %q, want %q", result.InvoiceNumber, "FAC0257566732026")
	}
	if result.Amount != 975.97 {
		t.Errorf("Amount = %f, want %f", result.Amount, 975.97)
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
