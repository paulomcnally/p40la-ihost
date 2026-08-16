package all

import (
	"strings"
	"testing"
)

// disnorteRealText es el texto plano tal como lo devuelve
// pdf.Page.GetPlainText() para la factura F122026071144973 (DISNORTE-DISSUR,
// julio 2026). Las celdas de las tablas quedan concatenadas sin separador,
// que es el comportamiento real de la librería ledongthuc/pdf con este tipo
// de PDF, por lo que las regex del analyzer están diseñadas para ese formato.
const disnorteRealText = `Distribuidora de Electricidad del Sur, S.A. J03100000037503458586CIRCUITO:TCPI3010MEDIDOR:24600326HEFACTURA NO.:F122026071144973ORDEN DE LECTURA:1220.33.0120.0578Total FacturadoC$1,662.11Cuota      0/0C$0.00Total a PagarC$1,662.11Consumo medioúltimos   12mesesKw máx/mes0,00Kwh/mes135C$/día29.31Oficina ComercialReferencia de cobroDías FacturadosMes de la FacturaConsumoFecha de EmisiónFecha de VencimientoTICUANTEPE345858612630JULIOREAL18/07/202607/08/2026Tipo de consumoNo. deMedidorLectura AnteriorLectura ActualMultip.Activa kWh BT24600326HE352237011179Período de ConsumoTarifa18/06/2026-18/07/2026 T0 BT. DOMESTICOFactor de Potencia:   0.00kW Contratados:      2Información ComplementariaEnergia (kWh)C$/kWhImporte252.4807062.02255.93370148.34286.22120174.19226.22120136.87508.26930413.47298.38950243.301791,178.19Detalle de FacturaciónImporte en C$Energia (kWh)1,178.19Alumbrado Publico147.78Comercializacion105.03Regulacion INE14.31IVA216.80Detalle de Deuda90 o más díasC$0.0060 díasC$0.0030 díasC$0.00Arreglo de PagoC$0.00MCNALLY ZAMBRANA PAULO ANTONIO1,662.1122/07/2026AGSPOCNODIENFEMZABMYJNJL0306090120150180210Impreso en PBS de Nicaragua S.A. RUC:J0310000006202 AIMP/1/0002/06-2025 Fecha Imp: 2026-07-18 Aut. DGI ASFC 01/0052/07/2024/8NINDIRI, VEREDAS DE VERACRUZ, VEREDAS DE VERACRUZ 1 1, 0 , ,KM 14 C MYA ENTRADA NUEVO MILENIO 486MDISTR. En manoVEREDAS DE VERACRUZ, VEREDAS DE VERACRUZVIVIENDA 1 CSA-A3CASA-A3 V337917512070DISSUR, S.A.3458586126000000000016621161MCNALLY ZAMBRANA PAULO ANTONIONINDIRI, VEREDAS DE VERACRUZ, VEREDAS DE VERACRUZ 1 1, 0 , ,KM 14 C MYA ENTRADA NUEVO MILENIO 486M3458586126JULIO18/07/20261,662.11DUPLICADO`

func TestDisnorteDissurAnalyzer_Info(t *testing.T) {
	a := &DisnorteDissurAnalyzer{}
	info := a.Info()

	if info.ID != "disnorte_dissur" {
		t.Errorf("ID = %q, se esperaba %q", info.ID, "disnorte_dissur")
	}
	if info.Name != "DISNORTE-DISSUR" {
		t.Errorf("Name = %q, se esperaba %q", info.Name, "DISNORTE-DISSUR")
	}
}

func TestDisnorteDissurAnalyzer_Analyze_UnsupportedMimeType(t *testing.T) {
	a := &DisnorteDissurAnalyzer{}
	_, err := a.Analyze(strings.NewReader("cualquier contenido"), "image/png")
	if err == nil {
		t.Fatal("se esperaba error por mime type no soportado, pero no hubo error")
	}
}

func TestReDisnorteAnchor(t *testing.T) {
	if !reDisnorteAnchor.MatchString(disnorteRealText) {
		t.Error("el ancla de DISNORTE-DISSUR debería coincidir con el texto real de la factura")
	}
	if reDisnorteAnchor.MatchString("Factura de otra empresa cualquiera, S.A.") {
		t.Error("el ancla de DISNORTE-DISSUR no debería coincidir con texto de otra empresa")
	}
}

func TestParseDisnorteDissur_RealInvoice(t *testing.T) {
	result, err := parseDisnorteDissur(disnorteRealText)
	if err != nil {
		t.Fatalf("parseDisnorteDissur() devolvió error inesperado: %v", err)
	}

	if result.InvoiceNumber != "F122026071144973" {
		t.Errorf("InvoiceNumber = %q, se esperaba %q", result.InvoiceNumber, "F122026071144973")
	}

	if result.Amount != 1662.11 {
		t.Errorf("Amount = %v, se esperaba %v", result.Amount, 1662.11)
	}

	if result.Year != 2026 {
		t.Errorf("Year = %d, se esperaba %d", result.Year, 2026)
	}

	if result.Month != 7 {
		t.Errorf("Month = %d, se esperaba %d (julio)", result.Month, 7)
	}

	if result.DueDate == nil {
		t.Fatal("DueDate no debería ser nil")
	}
	wantDueDate := "2026-08-07"
	gotDueDate := result.DueDate.Format("2006-01-02")
	if gotDueDate != wantDueDate {
		t.Errorf("DueDate = %s, se esperaba %s", gotDueDate, wantDueDate)
	}
}

func TestParseDisnorteDissur_MissingInvoiceNumber(t *testing.T) {
	text := strings.Replace(disnorteRealText, "FACTURA NO.:F122026071144973", "", 1)
	_, err := parseDisnorteDissur(text)
	if err == nil {
		t.Fatal("se esperaba error por falta de número de factura, pero no hubo error")
	}
}

func TestParseDisnorteDissur_MissingAmount(t *testing.T) {
	text := strings.Replace(disnorteRealText, "Total a PagarC$1,662.11", "Total a PagarC$0.00", 1)
	_, err := parseDisnorteDissur(text)
	if err == nil {
		t.Fatal("se esperaba error por monto en cero, pero no hubo error")
	}
}

func TestParseDisnorteDissur_MissingMesFechas(t *testing.T) {
	text := strings.Replace(disnorteRealText, "JULIOREAL18/07/202607/08/2026", "", 1)
	_, err := parseDisnorteDissur(text)
	if err == nil {
		t.Fatal("se esperaba error por falta de mes/año, pero no hubo error")
	}
}

func TestParseDisnorteDissur_EmptyText(t *testing.T) {
	_, err := parseDisnorteDissur("")
	if err == nil {
		t.Fatal("se esperaba error con texto vacío, pero no hubo error")
	}
}
