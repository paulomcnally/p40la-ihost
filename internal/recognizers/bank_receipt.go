package recognizers

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

// Tipos de documento que reconoce el MVP (SPEC-127).
const (
	DocumentTypeDebitNote = "nota_debito"
	DocumentTypeTransfer  = "confirmacion_transferencia"
)

// BankReceiptRecognizer reconoce comprobantes bancarios de pago: notas de
// débito y confirmaciones de transferencia. Es project-level: no depende de
// una institución, servicio ni deuda.
type BankReceiptRecognizer struct{}

func init() {
	Register(&BankReceiptRecognizer{})
}

// Info describe el reconocedor.
func (r *BankReceiptRecognizer) Info() RecognizerInfo {
	return RecognizerInfo{
		ID:   "bank_receipt_v1",
		Name: "Comprobante bancario",
	}
}

var (
	// Anclas de formato.
	reDebitAnchor    = regexp.MustCompile(`(?i)\bnota\s+de\s+d[eé]bito\b`)
	reTransferAnchor = regexp.MustCompile(`(?i)\bconfirmaci[oó]n\s+de\s+transferencia\b`)

	// Referencia / ID del comprobante.
	reDebitReference    = regexp.MustCompile(`(?i)\bReferencia\s*:?\s*([0-9][0-9\-]{5,23})`)
	reTransferReference = regexp.MustCompile(`(?i)\bN[uú]mero\s+de\s+Comprobante\s*:?\s*([0-9][0-9\-]{5,23})`)

	// Fecha y hora. En la nota de débito se prioriza la línea que empieza con
	// "Fecha" (fecha del pago) sobre la fecha de emisión del encabezado.
	reDebitDate        = regexp.MustCompile(`(?i)\bFecha\s*:?\s*(\d{2}/\d{2}/\d{4})`)
	reDebitPaymentDate = regexp.MustCompile(`(?im)^\s*Fecha\s*:?\s*(\d{2}/\d{2}/\d{4})`)
	reTransferDateTime = regexp.MustCompile(`(?i)\bFecha\s+y\s+Hora\s*:?\s*(\d{2}/\d{2}/\d{4})\s*[-–—]\s*(\d{2}:\d{2})`)
	reTime             = regexp.MustCompile(`(?i)\bHora\s*:?\s*(\d{2}:\d{2}:\d{2})`)
	reGMT              = regexp.MustCompile(`(?i)\bGMT\s*([+-])(\d{2}):?(\d{2})`)

	// Monto y moneda.
	reDebitAmount    = regexp.MustCompile(`(?i)\bMonto\s*:?\s*(?:US\$|C\$|\$)?\s*([\d,]+\.\d{2})`)
	reTransferAmount = regexp.MustCompile(`(?i)\bMonto\s+debitado\s*:?\s*(?:US\$|C\$|\$)?\s*([\d,]+\.\d{2})`)
	reDebitCurrency  = regexp.MustCompile(`(?i)\bMoneda\s*:?\s*(D[oó]lares|C[oó]rdobas|USD|NIO)`)

	// Concepto / detalle / motivo.
	reConceptDetail = regexp.MustCompile(`(?i)\bDetalle\s*:?\s*(.+?)(?:\s+(?:Moneda|Monto|Banco|Cuenta|Referencia|Nombre|Fecha|Hora)\b|$)`)
	reConceptMotivo = regexp.MustCompile(`(?i)\bMotivo\s*:?\s*(.+?)(?:\s+(?:Monto|Tipo|De|Para|Fecha)\b|$)`)

	// Número de cuota dentro del detalle.
	reInstallment = regexp.MustCompile(`(?i)\bCuota\s*#?\s*(\d{1,4})\b`)
)

// Recognize extrae el texto del PDF y detecta el formato del comprobante.
func (r *BankReceiptRecognizer) Recognize(src Source) (*RecognizedDocument, error) {
	if src.MimeType != "" && src.MimeType != "application/pdf" && src.MimeType != "application/octet-stream" {
		return nil, ErrUnsupportedFormat
	}

	text, err := extractPDFText(src)
	if err != nil {
		return nil, err
	}
	return parseBankReceiptText(text)
}

// extractPDFText lee el PDF por páginas y devuelve el texto plano. No se
// cargan imágenes ni se ejecuta OCR: si no hay capa de texto se devuelve
// ErrNeedsOCR (SPEC-127, REQ-003/CA-013).
func extractPDFText(src Source) (text string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			text = ""
			err = fmt.Errorf("%w: no se pudo leer el PDF", ErrInvalidPDF)
		}
	}()

	reader, err := pdf.NewReader(src.ReaderAt, src.Size)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPDF, err)
	}

	var builder strings.Builder
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		content, pageErr := page.GetPlainText(nil)
		if pageErr != nil {
			continue
		}
		builder.WriteString(content)
		builder.WriteByte('\n')
	}

	result := builder.String()
	if strings.TrimSpace(result) == "" {
		return "", ErrNeedsOCR
	}
	return result, nil
}

// parseBankReceiptText detecta el formato y normaliza los campos. Se separa
// del PDF para poder testear con texto sintético sin generar archivos.
func parseBankReceiptText(raw string) (*RecognizedDocument, error) {
	text := normalizeBankReceiptText(raw)

	switch {
	case reTransferAnchor.MatchString(text):
		return parseTransferReceipt(text), nil
	case reDebitAnchor.MatchString(text):
		return parseDebitReceipt(text), nil
	default:
		return nil, ErrUnsupportedFormat
	}
}

func parseDebitReceipt(text string) *RecognizedDocument {
	doc := &RecognizedDocument{
		DocumentType: DocumentTypeDebitNote,
		Confidence:   0.2,
	}

	if match := reDebitReference.FindStringSubmatch(text); len(match) > 1 {
		doc.Reference = strings.TrimSpace(match[1])
		doc.Confidence += 0.35
	}
	if dateValue := findDebitDate(text); dateValue != "" {
		doc.PaidAt = parseDebitDateTime(text, dateValue)
		if doc.PaidAt != nil {
			doc.Confidence += 0.15
		}
	}
	if match := reDebitAmount.FindStringSubmatch(text); len(match) > 1 {
		doc.Amount = parseAmount(match[1])
		if doc.Amount > 0 {
			doc.Confidence += 0.20
		}
	}
	if match := reDebitCurrency.FindStringSubmatch(text); len(match) > 1 {
		doc.Currency = normalizeCurrency(match[1])
		if doc.Currency != "" {
			doc.Confidence += 0.05
		}
	}

	doc.Concept = extractConcept(reConceptDetail, text)

	doc.InstallmentNumber = extractInstallment(doc.Concept)
	if doc.InstallmentNumber == 0 {
		doc.InstallmentNumber = extractInstallment(text)
	}
	if doc.InstallmentNumber > 0 {
		doc.Confidence += 0.05
	}

	finalizeRecognition(doc)
	return doc
}

func parseTransferReceipt(text string) *RecognizedDocument {
	doc := &RecognizedDocument{
		DocumentType: DocumentTypeTransfer,
		Confidence:   0.2,
	}

	if match := reTransferReference.FindStringSubmatch(text); len(match) > 1 {
		doc.Reference = strings.TrimSpace(match[1])
		doc.Confidence += 0.35
	}
	if match := reTransferDateTime.FindStringSubmatch(text); len(match) > 2 {
		doc.PaidAt = parseTransferDateTime(match[1], match[2])
		if doc.PaidAt != nil {
			doc.Confidence += 0.15
		}
	}
	if match := reTransferAmount.FindStringSubmatch(text); len(match) > 1 {
		doc.Amount = parseAmount(match[1])
		if doc.Amount > 0 {
			doc.Confidence += 0.20
		}
	}

	doc.Currency = detectTransferCurrency(text)
	if doc.Currency != "" {
		doc.Confidence += 0.05
	}

	doc.Concept = extractConcept(reConceptMotivo, text)
	doc.InstallmentNumber = extractInstallment(doc.Concept)
	if doc.InstallmentNumber == 0 {
		doc.InstallmentNumber = extractInstallment(text)
	}
	if doc.InstallmentNumber > 0 {
		doc.Confidence += 0.05
	}

	finalizeRecognition(doc)
	return doc
}

func findDebitDate(text string) string {
	if match := reDebitPaymentDate.FindStringSubmatch(text); len(match) > 1 {
		return match[1]
	}
	matches := reDebitDate.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return ""
	}
	// Si el PDF concatenó las líneas, la última fecha suele ser la del pago.
	return matches[len(matches)-1][1]
}

func parseDebitDateTime(text, dateValue string) *time.Time {
	parsedDate, err := time.ParseInLocation("02/01/2006", dateValue, time.UTC)
	if err != nil {
		return nil
	}

	hour, minute, second := 0, 0, 0
	if match := reTime.FindStringSubmatch(text); len(match) > 1 {
		if parsedTime, timeErr := time.Parse("15:04:05", match[1]); timeErr == nil {
			hour, minute, second = parsedTime.Clock()
		}
	}

	location := time.UTC
	if match := reGMT.FindStringSubmatch(text); len(match) > 3 {
		sign := 1
		if match[1] == "-" {
			sign = -1
		}
		hours, _ := strconv.Atoi(match[2])
		minutes, _ := strconv.Atoi(match[3])
		offset := sign * (hours*3600 + minutes*60)
		location = time.FixedZone(fmt.Sprintf("GMT%s%s:%s", match[1], match[2], match[3]), offset)
	}

	result := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), hour, minute, second, 0, location)
	return &result
}

func parseTransferDateTime(dateValue, timeValue string) *time.Time {
	parsed, err := time.ParseInLocation("02/01/2006 15:04", dateValue+" "+timeValue, time.UTC)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseAmount(value string) float64 {
	cleaned := strings.ReplaceAll(strings.TrimSpace(value), ",", "")
	amount, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}
	return amount
}

func normalizeCurrency(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dólares", "dolares", "usd", "us$", "$":
		return "USD"
	case "córdobas", "cordobas", "nio", "c$":
		return "NIO"
	default:
		return ""
	}
}

func detectTransferCurrency(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "c$") || strings.Contains(lower, "córdobas") ||
		strings.Contains(lower, "cordobas") || strings.Contains(lower, "nio") {
		return "NIO"
	}
	if strings.Contains(text, "$") || strings.Contains(lower, "dólares") ||
		strings.Contains(lower, "dolares") || strings.Contains(lower, "usd") {
		return "USD"
	}
	return ""
}

func extractConcept(expression *regexp.Regexp, text string) string {
	match := expression.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return SanitizeConcept(match[1])
}

func extractInstallment(value string) int {
	match := reInstallment.FindStringSubmatch(value)
	if len(match) < 2 {
		return 0
	}
	number, err := strconv.Atoi(match[1])
	if err != nil || number <= 0 || number > 10000 {
		return 0
	}
	return number
}

// finalizeRecognition calcula advertencias y aplica el tope de confianza
// cuando falta la referencia: sin ID del comprobante no se puede asociar con
// seguridad (SPEC-127, REQ-005).
func finalizeRecognition(doc *RecognizedDocument) {
	doc.Warnings = buildRecognitionWarnings(doc)
	if doc.Reference == "" && doc.Confidence > 0.5 {
		doc.Confidence = 0.5
	}
	doc.Confidence = clampConfidence(doc.Confidence)
}

func buildRecognitionWarnings(doc *RecognizedDocument) []string {
	warnings := make([]string, 0, 5)
	if doc.Reference == "" {
		warnings = append(warnings, "sin_referencia")
	}
	if doc.PaidAt == nil {
		warnings = append(warnings, "sin_fecha")
	}
	if doc.Amount <= 0 {
		warnings = append(warnings, "sin_monto")
	}
	if doc.Currency == "" {
		warnings = append(warnings, "sin_moneda")
	}
	if doc.InstallmentNumber == 0 {
		warnings = append(warnings, "sin_numero_cuota")
	}
	return warnings
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return math.Round(value*100) / 100
}

// normalizeBankReceiptText unifica espacios y saltos de línea. Algunos PDFs
// concatenan celdas sin separador; las regex toleran los espacios opcionales.
func normalizeBankReceiptText(raw string) string {
	raw = strings.ReplaceAll(raw, "\u00a0", " ")
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
