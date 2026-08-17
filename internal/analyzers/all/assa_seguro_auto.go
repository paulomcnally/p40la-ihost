package all

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/paulomcnally/p40la-ihost/internal/analyzers"
)

func init() {
	analyzers.Register(&AssaSeguroAutoAnalyzer{})
}

// AssaSeguroAutoAnalyzer extrae datos de recibos oficiales de caja emitidos
// por ASSA, Compañía de Seguros S.A. (Nicaragua) por cobro de prima. A
// diferencia de una factura, este documento es un comprobante de pago ya
// realizado, por lo que no tiene fecha de vencimiento.
type AssaSeguroAutoAnalyzer struct{}

func (a *AssaSeguroAutoAnalyzer) Info() analyzers.AnalyzerInfo {
	return analyzers.AnalyzerInfo{
		ID:   "assa_seguro_auto",
		Name: "ASSA - Seguro Auto",
	}
}

// reAssaAnchor identifica que el PDF corresponde a un recibo de ASSA
// Compañía de Seguros.
var reAssaAnchor = regexp.MustCompile(`(?i)ASSA,?\s*Compañía\s*de\s*Seguros`)

// reAssaInvoice captura el número de recibo. Existen dos variantes según la
// serie del talonario: "No. H3 116495" -> ("H3", "116495") y también
// "No. H 254818" -> ("H", "254818"), por lo que el dígito de la serie es
// opcional. El formato letra(+dígito opcional) seguido de espacio y un
// número de 4+ dígitos evita confundirse con el otro "No." del documento
// ("Autorización DGI No. ASFC 01/0105/12/2021/1."), que no sigue ese patrón
// (no hay espacio inmediatamente después de una sola letra ahí).
var reAssaInvoice = regexp.MustCompile(`No\.\s*([A-Z]\d?)\s+(\d{4,})`)

// reAssaAmount captura el monto total cobrado en córdobas (C$), ej:
// "Recibido: TAR 2,468.11" -> "2,468.11". El texto extraído del PDF
// concatena las celdas sin separador consistente
// ("...TARRecibido: 2,468.11Total..."), por lo que se ancla directamente
// en la etiqueta "Recibido:".
var reAssaAmount = regexp.MustCompile(`(?i)Recibido:\s*([\d,]+\.\d{2})`)

// reAssaFecha captura la fecha del recibo, ej: "Fecha:30-JUN-26" ->
// ("30", "JUN", "26"). El año se imprime con 2 dígitos.
var reAssaFecha = regexp.MustCompile(`(?i)Fecha:\s*(\d{2})-([A-Za-z]{3})-(\d{2})`)

// Campos opcionales para RawData.
var reAssaPoliza = regexp.MustCompile(`(\d{2}[A-Z]\d+)P`)
var reAssaCliente = regexp.MustCompile(`Cliente:\s*(\d+)`)
var reAssaReling = regexp.MustCompile(`Reling:\s*(\d+)`)
var reAssaTipoCambio = regexp.MustCompile(`(\d+\.\d+)T\.\s*Cambio:`)

func (a *AssaSeguroAutoAnalyzer) Analyze(reader io.Reader, mimeType string) (*analyzers.ExtractedBill, error) {
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

	if !reAssaAnchor.MatchString(text) {
		return nil, fmt.Errorf("no es un recibo de ASSA Compañía de Seguros")
	}

	return parseAssaSeguroAuto(text)
}

func parseAssaSeguroAuto(text string) (*analyzers.ExtractedBill, error) {
	result := &analyzers.ExtractedBill{}
	rawData := make(map[string]interface{})

	if m := reAssaInvoice.FindStringSubmatch(text); len(m) > 2 {
		result.InvoiceNumber = m[1] + m[2]
	}

	if m := reAssaAmount.FindStringSubmatch(text); len(m) > 1 {
		amt, err := parseAmount(m[1])
		if err == nil {
			result.Amount = amt
		}
	}

	if m := reAssaFecha.FindStringSubmatch(text); len(m) > 3 {
		month := parseMonth(m[2])
		yy, errY := strconv.Atoi(m[3])
		if month != 0 && errY == nil {
			result.Month = month
			result.Year = 2000 + yy
		}
	}

	if m := reAssaPoliza.FindStringSubmatch(text); len(m) > 1 {
		rawData["poliza"] = m[1]
	}
	if m := reAssaCliente.FindStringSubmatch(text); len(m) > 1 {
		rawData["cliente"] = m[1]
	}
	if m := reAssaReling.FindStringSubmatch(text); len(m) > 1 {
		rawData["reling"] = m[1]
	}
	if m := reAssaTipoCambio.FindStringSubmatch(text); len(m) > 1 {
		if tc, err := strconv.ParseFloat(m[1], 64); err == nil {
			rawData["tipo_cambio"] = tc
		}
	}
	if len(rawData) > 0 {
		result.RawData = rawData
	}

	if result.InvoiceNumber == "" || result.Amount == 0 || result.Year == 0 || result.Month == 0 {
		return nil, fmt.Errorf("campos obligatorios no encontrados (invoice=%s, amount=%.2f, year=%d, month=%d)",
			result.InvoiceNumber, result.Amount, result.Year, result.Month)
	}

	return result, nil
}
