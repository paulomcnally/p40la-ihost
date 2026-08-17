package all

import (
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/analyzers"
)

// assaRealText es el texto plano tal como lo devuelve pdf.Page.GetPlainText()
// para el recibo oficial de caja de ASSA Compañía de Seguros (reimpresión
// No. H3 116495). Al igual que con otros PDFs de tablas, las celdas quedan
// concatenadas sin separador y el orden de lectura no siempre coincide con
// el orden visual (p. ej. "TAR" aparece antes que su etiqueta "Recibido:").
const assaRealText = `Recibo Oficial de CajaASSA, Compañía de Seguros S.A.RUC J0310000003831Teléfono (505) 2276-9000Fax       (505) 2276-9003Edificio Corporativo ASSA, Pista Jean Paul Genie, costado oeste Edificio BIDWeb: www.assanet.comAutorización DGI No. ASFC 01/0105/12/2021/1.Recibimos de:PAULO ANTONIO MCNALLY ZAMBRANAIdentificación0012801880001HPor Cta de:Intermediario:CORREDURÍA INTERAMERICANA DE SEGUROS, S.ACobro De Prima Giro 9 de 12          67.39U$02B128265PólizaConceptoMonMonto36.6243T. Cambio:TARRecibido: 2,468.11TotalTotal:Impuesto:Prima: 67.39 8.19 58.13Otros Pagos: 0.00D. Emisión: 1.07Dir. Gral. de Bombero 1%: 0.00"Recibo no requiere firma ni sello del cajeropara su validez"Cliente:1079011Fecha:30-JUN-2610:59:37Cajero:CAJATARReling:5233844C$REIMPRESIONNo. H3 116495`

// assaRealTextSerieSinDigito es el texto real de un recibo cuya serie de
// talonario es una sola letra sin dígito ("No. H 254818", en vez de
// "No. H3 116495"). También cubre el caso de un recibo con varios giros de
// prima combinados en un solo pago (3 "Cobro De Prima Giro X de 12" antes
// de la sección de montos) y pago por transferencia ("TRF") en vez de
// tarjeta ("TAR"). Corresponde a un bug real reportado donde el analyzer
// fallaba en recibos con este formato de número de serie.
const assaRealTextSerieSinDigito = `Recibo Oficial de CajaASSA, Compañía de Seguros S.A.RUC J0310000003831Teléfono (505) 2276-9000Fax       (505) 2276-9003Edificio Corporativo ASSA, Pista Jean Paul Genie, costado oeste Edificio BIDWeb: www.assanet.comAutorización DGI No. ASFC 01/0105/12/2021/1.Recibimos de:PAULO ANTONIO MCNALLY ZAMBRANAIdentificación0012801880001HPor Cta de:MCNALLY ZAMBRANA PAULO ANTONIOIntermediario:CORREDURÍA INTERAMERICANA DE SEGUROS, S.ACobro De Prima Giro 1 de 12Cobro De Prima Giro 2 de 12Cobro De Prima Giro 3 de 12          67.39          67.39          67.38U$02B128265PólizaConceptoMonMonto36.6243T. Cambio:TRFRecibido: 202.16TotalTotal:Impuesto:Prima: 202.16 24.57 174.38Otros Pagos: 0.00D. Emisión: 3.21Dir. Gral. de Bombero 1%: 0.00"Recibo no requiere firma ni sello del cajeropara su validez"Cliente:1079011Fecha:20-OCT-2513:35:34Cajero:JRODRIGUEZReling:5044565U$REIMPRESIONNo. H 254818`

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

	if result.InvoiceNumber != "H3116495" {
		t.Errorf("InvoiceNumber = %q, se esperaba %q", result.InvoiceNumber, "H3116495")
	}

	if result.Amount != 2468.11 {
		t.Errorf("Amount = %v, se esperaba %v", result.Amount, 2468.11)
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
	if result.RawData["poliza"] != "02B128265" {
		t.Errorf(`RawData["poliza"] = %v, se esperaba "02B128265"`, result.RawData["poliza"])
	}
	if result.RawData["cliente"] != "1079011" {
		t.Errorf(`RawData["cliente"] = %v, se esperaba "1079011"`, result.RawData["cliente"])
	}
	if result.RawData["reling"] != "5233844" {
		t.Errorf(`RawData["reling"] = %v, se esperaba "5233844"`, result.RawData["reling"])
	}
	if result.RawData["tipo_cambio"] != 36.6243 {
		t.Errorf(`RawData["tipo_cambio"] = %v, se esperaba 36.6243`, result.RawData["tipo_cambio"])
	}
}

func TestParseAssaSeguroAuto_SerieSinDigito(t *testing.T) {
	// Regresión: el analyzer fallaba en recibos cuya serie de talonario es
	// una sola letra sin dígito (ej. "No. H 254818"), porque la regex
	// original exigía exactamente un dígito tras la letra (ej. "H3").
	result, err := parseAssaSeguroAuto(assaRealTextSerieSinDigito)
	if err != nil {
		t.Fatalf("parseAssaSeguroAuto() devolvió error inesperado: %v", err)
	}

	if result.InvoiceNumber != "H254818" {
		t.Errorf("InvoiceNumber = %q, se esperaba %q", result.InvoiceNumber, "H254818")
	}

	if result.Amount != 202.16 {
		t.Errorf("Amount = %v, se esperaba %v", result.Amount, 202.16)
	}

	if result.Year != 2025 {
		t.Errorf("Year = %d, se esperaba %d", result.Year, 2025)
	}

	if result.Month != 10 {
		t.Errorf("Month = %d, se esperaba %d (octubre)", result.Month, 10)
	}

	if result.RawData["poliza"] != "02B128265" {
		t.Errorf(`RawData["poliza"] = %v, se esperaba "02B128265"`, result.RawData["poliza"])
	}
	if result.RawData["reling"] != "5044565" {
		t.Errorf(`RawData["reling"] = %v, se esperaba "5044565"`, result.RawData["reling"])
	}
}

func TestParseAssaSeguroAuto_MissingInvoiceNumber(t *testing.T) {
	text := strings.Replace(assaRealText, "REIMPRESIONNo. H3 116495", "REIMPRESION", 1)
	_, err := parseAssaSeguroAuto(text)
	if err == nil {
		t.Fatal("se esperaba error por falta de número de recibo, pero no hubo error")
	}
}

func TestParseAssaSeguroAuto_MissingAmount(t *testing.T) {
	text := strings.Replace(assaRealText, "Recibido: 2,468.11Total", "Recibido: Total", 1)
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
