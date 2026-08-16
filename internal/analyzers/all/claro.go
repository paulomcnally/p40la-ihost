package all

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	"github.com/paulomcnally/p40la-ihost/internal/analyzers"
)

func init() {
	analyzers.Register(&ClaroAnalyzer{})
}

type ClaroAnalyzer struct{}

func (a *ClaroAnalyzer) Info() analyzers.AnalyzerInfo {
	return analyzers.AnalyzerInfo{
		ID:   "claro",
		Name: "Claro Nicaragua",
	}
}

func (a *ClaroAnalyzer) Analyze(reader io.Reader, mimeType string) (*analyzers.ExtractedBill, error) {
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

	format := detectFormat(text)
	switch format {
	case "residential":
		return parseResidential(text)
	case "mobile":
		return parseMobile(text)
	default:
		return nil, fmt.Errorf("formato de factura Claro no reconocido")
	}
}

func detectFormat(text string) string {
	re := regexp.MustCompile(`(?is)No\.?\s*de\s*factura:\s*(A\d+|FAC\d+)`)
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	if strings.HasPrefix(m[1], "FAC") {
		return "mobile"
	}
	return "residential"
}

var monthMap = map[string]int{
	"ENE": 1, "JAN": 1, "FEB": 2, "MAR": 3, "ABR": 4, "APR": 4,
	"MAY": 5, "JUN": 6, "JUL": 7, "AGO": 8, "AUG": 8, "SEP": 9,
	"OCT": 10, "NOV": 11, "DIC": 12, "DEC": 12,
}

func parseMonth(m string) int {
	return monthMap[strings.ToUpper(strings.TrimSpace(m))]
}

func parseAmount(s string) (float64, error) {
	s = strings.ReplaceAll(s, "C$", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	return strconv.ParseFloat(s, 64)
}

func extractDueDate(section string) *time.Time {
	rePeriod := regexp.MustCompile(`\d{2}/\w+/\d{4}\s*-\s*\d{2}/\w+/\d{4}`)
	periods := rePeriod.FindAllString(section, -1)

	periodDates := make(map[string]bool)
	for _, p := range periods {
		parts := strings.Split(p, "-")
		if len(parts) == 2 {
			periodDates[strings.TrimSpace(parts[0])] = true
			periodDates[strings.TrimSpace(parts[1])] = true
		}
	}

	reDate := regexp.MustCompile(`\d{2}/\w+/\d{4}`)
	dates := reDate.FindAllString(section, -1)
	var maxDate *time.Time
	for _, d := range dates {
		if periodDates[d] {
			continue
		}
		t, err := parseDateDMY(d)
		if err != nil {
			continue
		}
		if maxDate == nil || t.After(*maxDate) {
			maxDate = t
		}
	}
	return maxDate
}

func parseDateDMY(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("formato de fecha inválido: %s", s)
	}
	day, _ := strconv.Atoi(parts[0])
	monthStr := strings.ToUpper(strings.TrimSpace(parts[1]))
	month := parseMonth(monthStr)
	if month == 0 {
		m, err := strconv.Atoi(monthStr)
		if err == nil {
			month = m
		}
	}
	year, _ := strconv.Atoi(parts[2])
	if month == 0 || year == 0 {
		return nil, fmt.Errorf("mes o año no reconocido en: %s", s)
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return &t, nil
}

func parseResidential(text string) (*analyzers.ExtractedBill, error) {
	result := &analyzers.ExtractedBill{}

	reInvoice := regexp.MustCompile(`(?is)No\.?\s*de\s*factura:\s*(A\d+)`)
	if m := reInvoice.FindStringSubmatch(text); len(m) > 1 {
		result.InvoiceNumber = m[1]
	}

	idxTotal := strings.Index(strings.ToUpper(text), "TOTAL A PAGAR")
	if idxTotal >= 0 {
		rest := text[idxTotal:]
		endIdx := len(rest)
		for _, marker := range []string{"PRODUCTOS", "DETALLE", "PUNTOS DE PAGO"} {
			if i := strings.Index(rest, marker); i > 0 && i < endIdx {
				endIdx = i
			}
		}
		section := rest[:endIdx]
		reTotal := regexp.MustCompile(`C\$\s*([\d,]+\.?\d*)`)
		matches := reTotal.FindAllStringSubmatch(section, -1)
		if len(matches) > 0 {
			last := matches[len(matches)-1]
			amt, err := parseAmount(last[1])
			if err == nil {
				result.Amount = amt
			}
		}
	}

	idxFecha := strings.Index(strings.ToUpper(text), "FECHA LÍMITE DE PAGO")
	if idxFecha >= 0 {
		rest := text[idxFecha:]
		endIdx := len(rest)
		for _, marker := range []string{"PRODUCTOS", "TOTAL FACTURA", "DETALLE"} {
			if i := strings.Index(rest, marker); i > 0 && i < endIdx {
				endIdx = i
			}
		}
		section := rest[:endIdx]
		result.DueDate = extractDueDate(section)
	}

	reMes := regexp.MustCompile(`(?is)MES FACTURADO\s+.*?\d{2}/\w+/\d{4}\s*-\s*\d{2}/(\w{3})/(\d{4})`)
	if m := reMes.FindStringSubmatch(text); len(m) > 1 {
		parts := strings.Split(m[1], "/")
		if len(parts) == 2 {
			result.Month = parseMonth(parts[0])
			y, _ := strconv.Atoi(parts[1])
			result.Year = y
		}
	}

	if result.Year == 0 {
		rePer := regexp.MustCompile(`(?is)PERÍODO FACTURADO\s+.*?(\d{2})/(\w+)/(\d{4})\s*-\s*(\d{2})/(\w+)/(\d{4})`)
		if m := rePer.FindStringSubmatch(text); len(m) > 6 {
			result.Month = parseMonth(m[5])
			y, _ := strconv.Atoi(m[6])
			result.Year = y
		}
	}

	if result.InvoiceNumber == "" || result.Amount == 0 || result.Year == 0 || result.Month == 0 {
		return nil, fmt.Errorf("campos obligatorios no encontrados (invoice=%s, amount=%.2f, year=%d, month=%d)",
			result.InvoiceNumber, result.Amount, result.Year, result.Month)
	}

	return result, nil
}

func parseMobile(text string) (*analyzers.ExtractedBill, error) {
	result := &analyzers.ExtractedBill{}

	reInvoice := regexp.MustCompile(`(?is)No\.?\s*de\s*factura:\s*(FAC\d+)`)
	if m := reInvoice.FindStringSubmatch(text); len(m) > 1 {
		result.InvoiceNumber = m[1]
	}

	idxTotal := strings.Index(strings.ToUpper(text), "TOTAL A PAGAR")
	if idxTotal >= 0 {
		rest := text[idxTotal:]
		endIdx := len(rest)
		for _, marker := range []string{"PRODUCTOS", "DETALLE", "PUNTOS DE PAGO"} {
			if i := strings.Index(rest, marker); i > 0 && i < endIdx {
				endIdx = i
			}
		}
		section := rest[:endIdx]
		reTotal := regexp.MustCompile(`C\$\s*([\d,]+\.?\d*)`)
		matches := reTotal.FindAllStringSubmatch(section, -1)
		if len(matches) > 0 {
			last := matches[len(matches)-1]
			amt, err := parseAmount(last[1])
			if err == nil {
				result.Amount = amt
			}
		}
	}

	idxFecha := strings.Index(strings.ToUpper(text), "FECHA LÍMITE DE PAGO")
	if idxFecha >= 0 {
		rest := text[idxFecha:]
		endIdx := len(rest)
		for _, marker := range []string{"PRODUCTOS", "TOTAL FACTURA", "DETALLE"} {
			if i := strings.Index(rest, marker); i > 0 && i < endIdx {
				endIdx = i
			}
		}
		section := rest[:endIdx]
		result.DueDate = extractDueDate(section)
	}

	rePer := regexp.MustCompile(`(?is)PERÍODO FACTURADO\s+.*?(\d{2})/(\w+)/(\d{4})\s*-\s*(\d{2})/(\w+)/(\d{4})`)
	if m := rePer.FindStringSubmatch(text); len(m) > 6 {
		result.Month = parseMonth(m[5])
		y, _ := strconv.Atoi(m[6])
		result.Year = y
	}

	if result.InvoiceNumber == "" || result.Amount == 0 || result.Year == 0 || result.Month == 0 {
		return nil, fmt.Errorf("campos obligatorios no encontrados (invoice=%s, amount=%.2f, year=%d, month=%d)",
			result.InvoiceNumber, result.Amount, result.Year, result.Month)
	}

	return result, nil
}
