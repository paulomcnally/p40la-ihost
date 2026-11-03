package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ParsedLine es una línea de gasto detectada en el documento.
type ParsedLine struct {
	LineNo      int
	RawText     string
	Concept     string
	AmountTotal float64
	Currency    string
	Category    string // solo CSV: nombre de categoría provisto
}

// ParsedDocument es el resultado del parseo de un archivo.
type ParsedDocument struct {
	Lines         []ParsedLine
	DeclaredTotal *float64
	Year          int
	Month         int
	Source        string
	Warnings      []string
}

var amountTokenRe = regexp.MustCompile(`[-+]?\d[\d.,]*`)
var periodYearRe = regexp.MustCompile(`\b(20\d{2})\b`)
var numericPeriodRe = regexp.MustCompile(`\b(20\d{2})\s*[-/ ]\s*(\d{1,2})\b`)

var importMonthNames = map[string]int{
	"ENERO": 1, "FEBRERO": 2, "MARZO": 3, "ABRIL": 4, "MAYO": 5, "JUNIO": 6,
	"JULIO": 7, "AGOSTO": 8, "SEPTIEMBRE": 9, "SETIEMBRE": 9, "OCTUBRE": 10,
	"NOVIEMBRE": 11, "DICIEMBRE": 12,
	"JANUARY": 1, "FEBRUARY": 2, "MARCH": 3, "APRIL": 4, "MAY": 5, "JUNE": 6,
	"JULY": 7, "AUGUST": 8, "SEPTEMBER": 9, "OCTOBER": 10, "NOVEMBER": 11, "DECEMBER": 12,
}

// ExtractPDFText extrae la capa de texto de un PDF.
func ExtractPDFText(content []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("parsear PDF: %w", err)
	}
	var b strings.Builder
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		b.WriteString(text)
		b.WriteString("\n")
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", fmt.Errorf("pdf_without_text")
	}
	return text, nil
}

// ParseText parsea un documento de texto libre con líneas "concepto monto".
func ParseText(text string) (*ParsedDocument, error) {
	doc := &ParsedDocument{Source: "text"}
	lines := strings.Split(text, "\n")
	pendingConcept := ""

	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		// Encabezado de período: si el último número es el año, no es un gasto.
		if y, m := detectPeriod(line); y > 0 {
			if doc.Year == 0 {
				doc.Year, doc.Month = y, m
			}
			tokens := amountTokenRe.FindAllString(line, -1)
			if len(tokens) > 0 {
				if v, err := parseMoney(tokens[len(tokens)-1]); err == nil && int(v) == y {
					continue
				}
			}
		}
		concept, amount, currency, ok := splitConceptAmount(line)
		if !ok {
			pendingConcept = strings.TrimSpace(strings.TrimSpace(pendingConcept) + " " + line)
			continue
		}
		rawText := line
		if concept == "" {
			if pendingConcept == "" {
				continue
			}
			concept = pendingConcept
			rawText = pendingConcept + " " + line
		}
		pendingConcept = ""
		if doc.Year == 0 {
			if y, m := detectPeriod(concept); y > 0 {
				doc.Year, doc.Month = y, m
			}
		}
		if isTotalLine(concept) {
			v := amount
			doc.DeclaredTotal = &v
			continue
		}
		if amount <= 0 {
			doc.Warnings = append(doc.Warnings, fmt.Sprintf("línea %d ignorada: monto inválido", i+1))
			continue
		}
		doc.Lines = append(doc.Lines, ParsedLine{
			LineNo:      i + 1,
			RawText:     rawText,
			Concept:     concept,
			AmountTotal: amount,
			Currency:    currency,
		})
	}

	if doc.Year == 0 {
		// Sin encabezado: el llamador puede aportar year/month.
		return doc, nil
	}
	return doc, nil
}

// ParseCSV parsea el formato canónico definido en la SPEC-119.
func ParseCSV(r io.Reader) (*ParsedDocument, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("leer CSV: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("el CSV está vacío")
	}

	header := make(map[string]int)
	for i, h := range rows[0] {
		header[normalizeText(h)] = i
	}
	required := []string{"PERIODO", "CONCEPTO", "MONTO"}
	for _, col := range required {
		if _, ok := header[col]; !ok {
			return nil, fmt.Errorf("falta la columna obligatoria %q", strings.ToLower(col))
		}
	}

	doc := &ParsedDocument{Source: "csv"}
	seenPeriod := false
	for i, row := range rows[1:] {
		lineNo := i + 2
		if len(strings.TrimSpace(strings.Join(row, ""))) == 0 {
			continue
		}
		get := func(name string) string {
			idx, ok := header[normalizeText(name)]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}

		period := get("periodo")
		y, m, err := parsePeriodString(period)
		if err != nil {
			return nil, fmt.Errorf("fila %d: %w", lineNo, err)
		}
		if !seenPeriod {
			doc.Year, doc.Month = y, m
			seenPeriod = true
		} else if y != doc.Year || m != doc.Month {
			return nil, fmt.Errorf("fila %d: el CSV mezcla períodos distintos", lineNo)
		}

		concept := get("concepto")
		if concept == "" {
			return nil, fmt.Errorf("fila %d: concepto vacío", lineNo)
		}
		amount, err := parseMoney(get("monto"))
		if err != nil {
			return nil, fmt.Errorf("fila %d: monto inválido: %w", lineNo, err)
		}
		doc.Lines = append(doc.Lines, ParsedLine{
			LineNo:      lineNo,
			RawText:     strings.Join(row, ","),
			Concept:     concept,
			AmountTotal: amount,
			Currency:    strings.ToUpper(get("moneda")),
			Category:    get("categoria"),
		})
	}
	return doc, nil
}

func splitConceptAmount(line string) (string, float64, string, bool) {
	matches := amountTokenRe.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return "", 0, "", false
	}
	last := matches[len(matches)-1]
	token := line[last[0]:last[1]]
	amount, err := parseMoney(token)
	if err != nil {
		return "", 0, "", false
	}
	concept := strings.TrimSpace(line[:last[0]] + " " + line[last[1]:])
	concept = strings.Trim(concept, " \t-–—:.")
	concept = strings.Join(strings.Fields(concept), " ")
	currency := detectCurrency(line)
	return concept, amount, currency, true
}

// parseMoney interpreta montos con separador de miles/decimal es o en.
func parseMoney(raw string) (float64, error) {
	s := strings.TrimSpace(raw)
	s = strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || r == '.' || r == ',' || r == '-' || r == '+' {
			return r
		}
		return -1
	}, s)
	if s == "" {
		return 0, fmt.Errorf("monto vacío")
	}
	lastDot := strings.LastIndex(s, ".")
	lastComma := strings.LastIndex(s, ",")

	switch {
	case lastDot >= 0 && lastComma >= 0:
		if lastComma > lastDot {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.Replace(s, ",", ".", 1)
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	case lastComma >= 0:
		if isThousandsSeparator(s, ',') {
			s = strings.ReplaceAll(s, ",", "")
		} else {
			s = strings.Replace(s, ",", ".", 1)
		}
	case lastDot >= 0:
		if isThousandsSeparator(s, '.') {
			s = strings.ReplaceAll(s, ".", "")
		}
	}

	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("monto inválido %q", raw)
	}
	return v, nil
}

// isThousandsSeparator aplica la regla documentada: si el último grupo tiene
// exactamente 3 dígitos y no hay más separadores, se interpreta como miles.
func isThousandsSeparator(s string, sep byte) bool {
	parts := strings.Split(s, string(sep))
	if len(parts) != 2 {
		return false
	}
	return len(parts[1]) == 3 && parts[0] != "" && parts[0] != "0"
}

func detectCurrency(line string) string {
	u := strings.ToUpper(line)
	switch {
	case strings.Contains(u, "USD") || strings.Contains(u, "US$"):
		return "USD"
	case strings.Contains(u, "NIO") || strings.Contains(u, "C$"):
		return "NIO"
	}
	return ""
}

func isTotalLine(concept string) bool {
	n := normalizeText(concept)
	return strings.HasPrefix(n, "TOTAL") || strings.HasPrefix(n, "SUBTOTAL")
}

func detectPeriod(line string) (int, int) {
	u := normalizeText(line)
	if m := numericPeriodRe.FindStringSubmatch(u); len(m) == 3 {
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		if mo >= 1 && mo <= 12 {
			return y, mo
		}
	}
	year := 0
	if m := periodYearRe.FindStringSubmatch(u); len(m) == 2 {
		year, _ = strconv.Atoi(m[1])
	}
	if year == 0 {
		return 0, 0
	}
	for name, month := range importMonthNames {
		if strings.Contains(" "+u+" ", " "+name+" ") {
			return year, month
		}
	}
	return 0, 0
}

func parsePeriodString(raw string) (int, int, error) {
	u := strings.TrimSpace(raw)
	m := numericPeriodRe.FindStringSubmatch(u)
	if len(m) != 3 {
		return 0, 0, fmt.Errorf("período inválido %q (se espera YYYY-MM)", raw)
	}
	y, _ := strconv.Atoi(m[1])
	mo, _ := strconv.Atoi(m[2])
	if mo < 1 || mo > 12 {
		return 0, 0, fmt.Errorf("mes inválido en %q", raw)
	}
	return y, mo, nil
}

// normalizeText deja mayúsculas sin acentos, con espacios colapsados.
func normalizeText(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	replacer := strings.NewReplacer(
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ü", "U", "Ñ", "N",
	)
	s = replacer.Replace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// roundShare calcula el monto a registrar con redondeo half-up a 2 decimales.
func roundShare(total float64, sharePercent int) (amount, adjustment float64) {
	cents := int64(math.Round(total * 100))
	exact := float64(cents) * float64(sharePercent) / 100
	rounded := math.Round(exact)
	return rounded / 100, (rounded - exact) / 100
}
