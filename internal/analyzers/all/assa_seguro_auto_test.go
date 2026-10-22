package all

import (
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/analyzers"
)

// assaRealText es el texto plano tal como lo devuelve pdf.Page.GetPlainText()
// para el recibo oficial de caja de ASSA Compañía de Seguros (reimpresión
// No. H3 000000). Al igual que con otros PDFs de tablas, las celdas quedan
// concatenadas sin separador y el orden de lectura no siempre coincide con
// el orden visual (p. ej. "TAR" aparece antes que su etiqueta "Recibido:").
const assaRealText = `Recibo Oficial de CajaASSA, Compañía de Seguros S.A.RUC J0310000003831Teléfono (555) 010-0000Fax       (555) 010-0001Edificio Corporativo ASSA, Pista Jean Paul Genie, costado oeste Edificio BIDWeb: www.assanet.comAutorización DGI No. ASFC 01/0105/12/2021/1.Recibimos de:CLIENTE DEMO EJEMPLOIdentificación0000000000000XPor Cta de:Intermediario:CORREDURÍA INTERAMERICANA DE SEGUROS, S.ACobro De Prima Giro 9 de 12          10.00U$00X000000PólizaConceptoMonMonto1.0000T. Cambio:TARRecibido: 999.99TotalTotal:Impuesto:Prima: 10.00 2.00 8.00Otros Pagos: 0.00D. Emisión: 1.00Dir. Gral. de Bombero 1%: 0.00"Recibo no requiere firma ni sello del cajeropara su validez"Cliente:0000000Fecha:30-JUN-2610:59:37Cajero:CAJATARReling:0000000C$REIMPRESIONNo. H3 000000`

// assaRealTextSerieSinDigito es el texto real de un recibo cuya serie de
// talonario es una sola letra sin dígito ("No. H 000000", en vez de
// "No. H3 000000"). También cubre el caso de un recibo con varios giros de
// prima combinados en un solo pago (3 "Cobro De Prima Giro X de 12" antes
// de la sección de montos) y pago por transferencia ("TRF") en vez de
// tarjeta ("TAR"). Corresponde a un bug real reportado donde el analyzer
// fallaba en recibos con este formato de número de serie.
const assaRealTextSerieSinDigito = `Recibo Oficial de CajaASSA, Compañía de Seguros S.A.RUC J0310000003831Teléfono (555) 010-0000Fax       (555) 010-0001Edificio Corporativo ASSA, Pista Jean Paul Genie, costado oeste Edificio BIDWeb: www.assanet.comAutorización DGI No. ASFC 01/0105/12/2021/1.Recibimos de:CLIENTE DEMO EJEMPLOIdentificación0000000000000XPor Cta de:CLIENTE DEMO EJEMPLOIntermediario:CORREDURÍA INTERAMERICANA DE SEGUROS, S.ACobro De Prima Giro 1 de 12Cobro De Prima Giro 2 de 12Cobro De Prima Giro 3 de 12          10.00          10.00          10.00U$00X000000PólizaConceptoMonMonto1.0000T. Cambio:TRFRecibido: 888.88TotalTotal:Impuesto:Prima: 888.88 5.00 20.00Otros Pagos: 0.00D. Emisión: 2.00Dir. Gral. de Bombero 1%: 0.00"Recibo no requiere firma ni sello del cajeropara su validez"Cliente:0000000Fecha:20-OCT-2513:35:34Cajero:CAJEROReling:0000000U$REIMPRESIONNo. H 000000`

func TestAssaSeguroAutoAnalyzer_Info(t *testing.T) {
	a := &AssaSeguroAutoAnalyzer{}
	info := a.Info()

	if info.ID != "assa_seguro_auto" {
		t.Errorf("ID = %q, se esperaba %q", info.ID, "assa_seguro_auto")
	}
	if info.Name != "ASSA - Seguro Auto" {
		t.Errorf("Name = %q, se esperaba %q", info.Name, "ASSA - Seguro Auto")
	}
}

func TestAssaSeguroAutoAnalyzer_Registered(t *testing.T) {
	a, ok := analyzers.Get("assa_seguro_auto")
	if !ok {
		t.Fatal("analyzer 'assa_seguro_auto' no está registrado")
	}
	info := a.Info()
	if info.Name != "ASSA - Seguro Auto" {
		t.Errorf("Name = %q, want %q", info.Name, "ASSA - Seguro Auto")
	}
}

func TestAssaSeguroAutoAnalyzer_Analyze_UnsupportedMimeType(t *testing.T) {
	a := &AssaSeguroAutoAnalyzer{}
	_, err := a.Analyze(strings.NewReader("cualquier contenido"), "image/png")
	if err == nil {
		t.Fatal("se esperaba error por mime type no soportado, pero no hubo error")
	}
}

func TestReAssaAnchor(t *testing.T) {
	if !reAssaAnchor.MatchString(assaRealText) {
		t.Error("el ancla de ASSA debería coincidir con el texto real del recibo")
	}
	if reAssaAnchor.MatchString("Factura de otra empresa cualquiera, S.A.") {
		t.Error("el ancla de ASSA no debería coincidir con texto de otra empresa")
	}
}

func TestParseAssaSeguroAuto_RealReceipt(t *testing.T) {
	result, err := parseAssaSeguroAuto(assaRealText)
	if err != nil {
		t.Fatalf("parseAssaSeguroAuto() devolvió error inesperado: %v", err)
	}

	if result.InvoiceNumber != "H3000000" {
		t.Errorf("InvoiceNumber = %q, se esperaba %q", result.InvoiceNumber, "H3000000")
	}

	if result.Amount != 999.99 {
		t.Errorf("Amount = %v, se esperaba %v", result.Amount, 999.99)
	}

	if result.Year != 2026 {
		t.Errorf("Year = %d, se esperaba %d", result.Year, 2026)
	}

	if result.Month != 6 {
		t.Errorf("Month = %d, se esperaba %d (junio)", result.Month, 6)
	}

	if result.DueDate != nil {
		t.Errorf("DueDate = %v, se esperaba nil (es un recibo, no una factura)", result.DueDate)
	}

	if result.RawData == nil {
		t.Fatal("RawData no debería ser nil")
	}
	if result.RawData["poliza"] != "00X000000" {
		t.Errorf(`RawData["poliza"] = %v, se esperaba "00X000000"`, result.RawData["poliza"])
	}
	if result.RawData["cliente"] != "0000000" {
		t.Errorf(`RawData["cliente"] = %v, se esperaba "0000000"`, result.RawData["cliente"])
	}
	if result.RawData["reling"] != "0000000" {
		t.Errorf(`RawData["reling"] = %v, se esperaba "0000000"`, result.RawData["reling"])
	}
	if result.RawData["tipo_cambio"] != 1.0000 {
		t.Errorf(`RawData["tipo_cambio"] = %v, se esperaba 1.0000`, result.RawData["tipo_cambio"])
	}
}

func TestParseAssaSeguroAuto_SerieSinDigito(t *testing.T) {
	// Regresión: el analyzer fallaba en recibos cuya serie de talonario es
	// una sola letra sin dígito (ej. "No. H 000000"), porque la regex
	// original exigía exactamente un dígito tras la letra (ej. "H3").
	result, err := parseAssaSeguroAuto(assaRealTextSerieSinDigito)
	if err != nil {
		t.Fatalf("parseAssaSeguroAuto() devolvió error inesperado: %v", err)
	}

	if result.InvoiceNumber != "H000000" {
		t.Errorf("InvoiceNumber = %q, se esperaba %q", result.InvoiceNumber, "H000000")
	}

	if result.Amount != 888.88 {
		t.Errorf("Amount = %v, se esperaba %v", result.Amount, 888.88)
	}

	if result.Year != 2025 {
		t.Errorf("Year = %d, se esperaba %d", result.Year, 2025)
	}

	if result.Month != 10 {
		t.Errorf("Month = %d, se esperaba %d (octubre)", result.Month, 10)
	}

	if result.RawData["poliza"] != "00X000000" {
		t.Errorf(`RawData["poliza"] = %v, se esperaba "00X000000"`, result.RawData["poliza"])
	}
	if result.RawData["reling"] != "0000000" {
		t.Errorf(`RawData["reling"] = %v, se esperaba "0000000"`, result.RawData["reling"])
	}
}

func TestParseAssaSeguroAuto_MissingInvoiceNumber(t *testing.T) {
	text := strings.Replace(assaRealText, "REIMPRESIONNo. H3 000000", "REIMPRESION", 1)
	_, err := parseAssaSeguroAuto(text)
	if err == nil {
		t.Fatal("se esperaba error por falta de número de recibo, pero no hubo error")
	}
}

func TestParseAssaSeguroAuto_MissingAmount(t *testing.T) {
	text := strings.Replace(assaRealText, "Recibido: 999.99Total", "Recibido: Total", 1)
	_, err := parseAssaSeguroAuto(text)
	if err == nil {
		t.Fatal("se esperaba error por falta de monto, pero no hubo error")
	}
}

func TestParseAssaSeguroAuto_MissingFecha(t *testing.T) {
	text := strings.Replace(assaRealText, "Fecha:30-JUN-26", "Fecha:", 1)
	_, err := parseAssaSeguroAuto(text)
	if err == nil {
		t.Fatal("se esperaba error por falta de fecha, pero no hubo error")
	}
}

func TestParseAssaSeguroAuto_EmptyText(t *testing.T) {
	_, err := parseAssaSeguroAuto("")
	if err == nil {
		t.Fatal("se esperaba error con texto vacío, pero no hubo error")
	}
}

func TestParseAssaSeguroAuto_DoesNotMatchDgiAuthorizationNumber(t *testing.T) {
	// El texto contiene otro "No." (Autorización DGI No. ASFC ...) que no
	// debe confundirse con el número de recibo real.
	if strings.Contains(assaRealText, "No. ASFC") {
		if reAssaInvoice.MatchString("Autorización DGI No. ASFC 01/0105/12/2021/1.") {
			t.Error("reAssaInvoice no debería matchear el número de autorización DGI")
		}
	}
}
