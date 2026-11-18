package statementimport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

const maxPDFSize = 10 * 1024 * 1024

var (
	datedRowRe = regexp.MustCompile(`(?s)(\d{2}/\d{2}/\d{4})(.*?)(-?\d[\d,]*\.\d{2})\s*(-?\d[\d,]*\.\d{2})`)
	bonusRe    = regexp.MustCompile(`(?is)(BONIFICACION[^\d]*?)(-?\d[\d,]*\.\d{2})\s*(-?\d[\d,]*\.\d{2})`)

	sectionStartRe = regexp.MustCompile(`(?i)previous\s*balance`)
	sectionEndRe   = regexp.MustCompile(`(?i)intereses\s*corrientes\s*del`)

	productRe       = regexp.MustCompile(`(?i)producto\s*:\s*([0-9x*\-]+)`)
	periodEndRe     = regexp.MustCompile(`(?i)fecha\s*de\s*corte\s*:\s*(\d{2}/\d{2}/\d{4})`)
	minDueRe        = regexp.MustCompile(`(?i)pago\s*m[ií]nimo\s*/\s*fecha\s*l[ií]mite\s*:\s*(\d{2}/\d{2}/\d{4})`)
	minDueAmountRe  = regexp.MustCompile(`(?i)pago\s*m[ií]nimo\s*/\s*monto\s*local\s*:\s*(-?[\d,.]+)`)
	minDueUSDRe     = regexp.MustCompile(`(?i)pago\s*m[ií]nimo\s*/\s*monto\s*d[oó]lares\s*:\s*(-?[\d,.]+)`)
	fullDueRe       = regexp.MustCompile(`(?i)pago\s*contado\s*/\s*fecha\s*l[ií]mite\s*:\s*(\d{2}/\d{2}/\d{4})`)
	fullDueAmountRe = regexp.MustCompile(`(?i)pago\s*contado\s*/\s*monto\s*local\s*:\s*(-?[\d,.]+)`)
	fullDueUSDRe    = regexp.MustCompile(`(?i)pago\s*contado\s*/\s*monto\s*d[oó]lares\s*:\s*(-?[\d,.]+)`)
	previousRe      = regexp.MustCompile(`(?i)previous\s*balance\s*(-?[\d,]+\.\d{2})(-?[\d,]+\.\d{2})`)
	newBalanceRe    = regexp.MustCompile(`(?i)saldo\s*al\s*corte\s*/\s*local\s*:\s*(-?[\d,.]+)`)
	newBalanceUSDRe = regexp.MustCompile(`(?i)saldo\s*al\s*corte\s*/\s*d[oó]lares\s*:\s*(-?[\d,.]+)`)
	interestRe      = regexp.MustCompile(`(?i)intereses\s*corrientes\s*del\s*mes\s*/\s*local\s*:\s*(-?[\d,.]+)`)
	interestUSDRe   = regexp.MustCompile(`(?i)intereses\s*corrientes\s*del\s*mes\s*/\s*d[oó]lares\s*:\s*(-?[\d,.]+)`)
	last4Re         = regexp.MustCompile(`(\d{4})\D*$`)
)

// Parse extrae el estado de cuenta desde un PDF de BAC Nicaragua.
func (m *Module) Parse(ctx context.Context, r io.Reader) (parsed *models.ParsedStatement, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			parsed = nil
			err = fmt.Errorf("no se pudo interpretar el PDF: %v", recovered)
		}
	}()

	data, err := io.ReadAll(io.LimitReader(r, maxPDFSize+1))
	if err != nil {
		return nil, fmt.Errorf("leer PDF: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("el PDF está vacío")
	}
	if len(data) > maxPDFSize {
		return nil, fmt.Errorf("el PDF supera el límite de 10 MB")
	}

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("abrir PDF: %w", err)
	}

	var text strings.Builder
	for page := 1; page <= reader.NumPage(); page++ {
		content, err := reader.Page(page).GetPlainText(nil)
		if err != nil {
			continue
		}
		text.WriteString(content)
		text.WriteString("\n")
	}
	if strings.TrimSpace(text.String()) == "" {
		return nil, fmt.Errorf("el PDF no contiene texto")
	}
	return parseText(text.String())
}

// parseText interpreta el texto extraído. Se separa del PDF para poder testearlo
// con fixtures sintéticos (SEC-NO-PII: no se commitean estados de cuenta).
func parseText(text string) (*models.ParsedStatement, error) {
	periodEnd := isoDate(firstMatch(periodEndRe, text))
	minDue := isoDate(firstMatch(minDueRe, text))
	fullDue := isoDate(firstMatch(fullDueRe, text))
	if periodEnd == "" || minDue == "" || fullDue == "" {
		return nil, fmt.Errorf("faltan fechas del encabezado")
	}

	parsed := &models.ParsedStatement{
		ModuleID:           ModuleID,
		ModuleVersion:      ModuleVersion,
		Last4:              extractLast4(firstMatch(productRe, text)),
		PeriodEnd:          periodEnd,
		MinPaymentDue:      minDue,
		FullPaymentDue:     fullDue,
		MinPayment:         amount(firstMatch(minDueAmountRe, text)),
		MinPaymentUSD:      amount(firstMatch(minDueUSDRe, text)),
		TotalPayment:       amount(firstMatch(fullDueAmountRe, text)),
		TotalPaymentUSD:    amount(firstMatch(fullDueUSDRe, text)),
		NewBalance:         amount(firstMatch(newBalanceRe, text)),
		NewBalanceUSD:      amount(firstMatch(newBalanceUSDRe, text)),
		InterestCurrent:    amount(firstMatch(interestRe, text)),
		InterestCurrentUSD: amount(firstMatch(interestUSDRe, text)),
	}

	if previous := previousRe.FindStringSubmatch(text); previous != nil {
		parsed.PreviousBalance = amount(previous[1])
		parsed.PreviousBalanceUSD = amount(previous[2])
	} else {
		parsed.Warnings = append(parsed.Warnings, models.ParseWarning{
			Code:    "missing_previous_balance",
			Message: "No se encontró el saldo anterior en el estado de cuenta",
		})
	}

	region := transactionRegion(text)
	parsed.Transactions = append(parsed.Transactions, parseDatedRows(region)...)
	parsed.Transactions = append(parsed.Transactions, parseBonusRows(region, periodEnd)...)

	if parsed.InterestCurrent != 0 || parsed.InterestCurrentUSD != 0 {
		parsed.Transactions = append(parsed.Transactions, models.ParsedTransaction{
			PostedAt:         periodEnd,
			Concept:          "Intereses corrientes del mes",
			Type:             typeInterest,
			AmountLocal:      parsed.InterestCurrent,
			AmountUSD:        parsed.InterestCurrentUSD,
			CurrencyOriginal: originalCurrency(parsed.InterestCurrent, parsed.InterestCurrentUSD),
			ExternalRef:      externalRef(periodEnd, "interest", parsed.InterestCurrent, parsed.InterestCurrentUSD, 0),
		})
	}

	if len(parsed.Transactions) == 0 {
		return nil, fmt.Errorf("no se encontraron movimientos")
	}

	diffLocal := round2(parsed.PreviousBalance + sumLocal(parsed.Transactions) - parsed.NewBalance)
	diffUSD := round2(parsed.PreviousBalanceUSD + sumUSD(parsed.Transactions) - parsed.NewBalanceUSD)
	if math.Abs(diffLocal) > 0.005 || math.Abs(diffUSD) > 0.005 {
		parsed.Warnings = append(parsed.Warnings, models.ParseWarning{
			Code: "reconciliation_mismatch",
			Message: fmt.Sprintf(
				"El saldo impreso no coincide con los movimientos (diferencia local %.2f, USD %.2f)",
				diffLocal, diffUSD,
			),
		})
	}

	sort.SliceStable(parsed.Transactions, func(i, j int) bool {
		return parsed.Transactions[i].PostedAt < parsed.Transactions[j].PostedAt
	})
	return parsed, nil
}

func parseDatedRows(region string) []models.ParsedTransaction {
	matches := datedRowRe.FindAllStringSubmatch(region, -1)
	out := make([]models.ParsedTransaction, 0, len(matches))
	for index, match := range matches {
		concept := cleanConcept(match[2])
		if concept == "" || isNoiseConcept(concept) {
			continue
		}
		local := amount(match[3])
		usd := amount(match[4])
		out = append(out, models.ParsedTransaction{
			PostedAt:         isoDate(match[1]),
			Concept:          concept,
			Type:             classifyConcept(concept),
			AmountLocal:      local,
			AmountUSD:        usd,
			CurrencyOriginal: originalCurrency(local, usd),
			ExternalRef:      externalRef(match[1], concept, match[3], match[4], index),
		})
	}
	return out
}

func parseBonusRows(region, periodEnd string) []models.ParsedTransaction {
	matches := bonusRe.FindAllStringSubmatch(region, -1)
	out := make([]models.ParsedTransaction, 0, len(matches))
	for index, match := range matches {
		concept := cleanConcept(match[1])
		local := amount(match[2])
		usd := amount(match[3])
		out = append(out, models.ParsedTransaction{
			PostedAt:         periodEnd,
			Concept:          concept,
			Type:             typePaymentBonus,
			AmountLocal:      local,
			AmountUSD:        usd,
			CurrencyOriginal: originalCurrency(local, usd),
			ExternalRef:      externalRef(periodEnd, concept, match[2], match[3], index),
		})
	}
	return out
}

func transactionRegion(text string) string {
	start := sectionStartRe.FindStringIndex(text)
	end := sectionEndRe.FindStringIndex(text)
	if start == nil {
		return text
	}
	from := start[1]
	if end != nil && end[0] > from {
		return text[from:end[0]]
	}
	return text[from:]
}

func firstMatch(re *regexp.Regexp, text string) string {
	match := re.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	return match[1]
}

func extractLast4(product string) string {
	product = strings.TrimSpace(product)
	if product == "" {
		return ""
	}
	match := last4Re.FindStringSubmatch(product)
	if match == nil {
		return ""
	}
	return match[1]
}

func amount(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "C$", "")
	value = strings.ReplaceAll(value, "$", "")
	value = strings.ReplaceAll(value, " ", "")
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func isoDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "/")
	if len(parts) != 3 {
		return ""
	}
	return parts[2] + "-" + parts[1] + "-" + parts[0]
}

func cleanConcept(concept string) string {
	concept = strings.Join(strings.Fields(concept), " ")
	for _, suffix := range []string{" NI", " -R-", " -CY-"} {
		concept = strings.TrimSuffix(concept, suffix)
	}
	concept = strings.TrimSpace(concept)
	if len(concept) > 200 {
		concept = concept[:200]
	}
	return concept
}

func isNoiseConcept(concept string) bool {
	value := normalizeConcept(concept)
	for _, prefix := range []string{
		"PRICECASH", "DISPONIBLE", "SU DISPONIBLE", "ACCIONES GANADAS",
		"SORTEO", "SERIE", "PAGINA", "PRODUCTO", "NOMBRE", "FECHA DE CORTE",
		"PAGO MINIMO", "PAGO CONTADO", "SALDO AL CORTE", "INTERESES",
	} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func originalCurrency(local, usd float64) string {
	if usd != 0 {
		return "USD"
	}
	return "NIO"
}

func sumLocal(transactions []models.ParsedTransaction) float64 {
	var total float64
	for _, item := range transactions {
		total += item.AmountLocal
	}
	return total
}

func sumUSD(transactions []models.ParsedTransaction) float64 {
	var total float64
	for _, item := range transactions {
		total += item.AmountUSD
	}
	return total
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func externalRef(parts ...any) string {
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, fmt.Sprint(part))
	}
	sum := sha256.Sum256([]byte(strings.Join(values, "|")))
	return hex.EncodeToString(sum[:])[:16]
}
