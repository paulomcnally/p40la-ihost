package all

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/paulomcnally/p40la-ihost/internal/analyzers"
)

func init() {
	analyzers.Register(&DisnorteDissurAnalyzer{})
}

// DisnorteDissurAnalyzer extrae datos de facturas de energía eléctrica
// emitidas por Distribuidora de Electricidad del Sur, S.A. (DISNORTE-DISSUR),
// distribuidora eléctrica de Nicaragua.
type DisnorteDissurAnalyzer struct{}

func (a *DisnorteDissurAnalyzer) Info() analyzers.AnalyzerInfo {
	return analyzers.AnalyzerInfo{
		ID:   "disnorte_dissur",
		Name: "DISNORTE-DISSUR",
	}
}

// reDisnorteAnchor identifica que el PDF corresponde a una factura de
// DISNORTE-DISSUR (razón social completa de la distribuidora).
var reDisnorteAnchor = regexp.MustCompile(`(?i)Distribuidora de Electricidad del Sur`)

// reDisnorteInvoice captura el número de factura, ej: "F122026071144973".
// El texto extraído del PDF concatena celdas sin separador
// ("FACTURA NO.:F122026071144973ORDEN DE LECTURA:..."), por lo que \d+
// se detiene naturalmente al llegar a la siguiente letra ("O" de ORDEN).
var reDisnorteInvoice = regexp.MustCompile(`(?i)FACTURA\s*NO\.?:?\s*([A-Z]+\d+)`)

// reDisnorteAmount captura el monto de "Total a Pagar", ej: "1,662.11".
// \D*? consume de forma no ávida cualquier texto/símbolo (como "C$") entre
// la etiqueta y el número.
var reDisnorteAmount = regexp.MustCompile(`(?i)Total\s*a\s*Pagar\D*?([\d,]+\.\d{2})`)

// reDisnorteMesFechas captura, en una sola fila de la tabla de resumen,
// el mes facturado (nombre completo en español), la fecha de emisión y la
// fecha de vencimiento, ej: "JULIOREAL18/07/202607/08/2026" ->
// ("JULIO", "18/07/2026", "07/08/2026"). El grupo [A-Z]{0,15}? absorbe
// palabras intermedias como "REAL" o "ESTIMADO" (tipo de consumo).
var reDisnorteMesFechas = regexp.MustCompile(
	`(?i)(ENERO|FEBRERO|MARZO|ABRIL|MAYO|JUNIO|JULIO|AGOSTO|SEPTIEMBRE|OCTUBRE|NOVIEMBRE|DICIEMBRE)[A-Z]{0,15}?(\d{2}/\d{2}/\d{4})(\d{2}/\d{2}/\d{4})`,
)

func (a *DisnorteDissurAnalyzer) Analyze(reader io.Reader, mimeType string) (*analyzers.ExtractedBill, error) {
	if mimeType != "application/pdf" {
		return nil, fmt.Errorf("formato no soportado por este analyzer: %s", mimeType)
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, fmt.Errorf("leer archivo: %w", err)
	}

	pdfReader, err := pdf.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return nil, fmt.Errorf("parsear PDF: %w", err)
	}

	var textBuilder strings.Builder
	for i := 1; i <= pdfReader.NumPage(); i++ {
		page := pdfReader.Page(i)
		content, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		textBuilder.WriteString(content)
		textBuilder.WriteString("\n")
	}

	text := textBuilder.String()
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("no se pudo extraer texto del PDF")
	}

	if !reDisnorteAnchor.MatchString(text) {
		return nil, fmt.Errorf("no es una factura de DISNORTE-DISSUR")
	}

	return parseDisnorteDissur(text)
}

func parseDisnorteDissur(text string) (*analyzers.ExtractedBill, error) {
	result := &analyzers.ExtractedBill{}

	if m := reDisnorteInvoice.FindStringSubmatch(text); len(m) > 1 {
		result.InvoiceNumber = strings.ToUpper(m[1])
	}

	if m := reDisnorteAmount.FindStringSubmatch(text); len(m) > 1 {
		amt, err := parseAmount(m[1])
		if err == nil {
			result.Amount = amt
		}
	}

	if m := reDisnorteMesFechas.FindStringSubmatch(text); len(m) > 3 {
		fechaEmision := m[2]
		fechaVencimiento := m[3]

		if emision, err := parseDateDMY(fechaEmision); err == nil {
			result.Year = emision.Year()
			result.Month = int(emision.Month())
		}

		if vencimiento, err := parseDateDMY(fechaVencimiento); err == nil {
			result.DueDate = vencimiento
		}
	}

	if result.InvoiceNumber == "" || result.Amount == 0 || result.Year == 0 || result.Month == 0 {
		return nil, fmt.Errorf("campos obligatorios no encontrados (invoice=%s, amount=%.2f, year=%d, month=%d)",
			result.InvoiceNumber, result.Amount, result.Year, result.Month)
	}

	return result, nil
}
