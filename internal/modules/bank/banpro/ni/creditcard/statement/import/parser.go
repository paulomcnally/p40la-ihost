// Package statementimport parsea el estado de cuenta mensual de BANPRO
// Nicaragua (SPEC-139). El PDF se procesa en memoria y se descarta.
package statementimport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/modules"
	"github.com/paulomcnally/p40la-ihost/internal/modules/bank/statementtext"
)

const maxPDFSize = 10 * 1024 * 1024

var (
	referenceRe = regexp.MustCompile(`(\d{18,26})\s+(\d{2})/(\d{2})`)

	headerRe      = regexp.MustCompile(`(?i)banco\s*de\s*la\s*producci.{0,2}n`)
	last4Re       = regexp.MustCompile(`(?i)n.{0,2}mero\s*de\s*tarjeta\s*:?\D*(\d{4})`)
	dateRe        = regexp.MustCompile(`(\d{2}/\d{2}/\d{4})`)
	previousRe    = regexp.MustCompile(`(?i)saldo\s*anterior\s*:?\s*(-?\d[\d, ]*\.\d{2})\s*(-?\d[\d, ]*\.\d{2})`)
	chargesRe     = regexp.MustCompile(`(?i)cargos\s*y\s*d.{0,2}bitos\s*:?\s*(-?\d[\d, ]*\.\d{2})\s*(-?\d[\d, ]*\.\d{2})`)
	paymentsRe    = regexp.MustCompile(`(?i)pagos\s*y\s*cr.{0,2}ditos\s*:?\s*(-?\d[\d, ]*\.\d{2})\s*(-?\d[\d, ]*\.\d{2})`)
	totalPayRe    = regexp.MustCompile(`(?i)pago\s*de\s*contado\s*:?\s*(-?\d[\d, ]*\.\d{2})\s*(-?\d[\d, ]*\.\d{2})`)
	minPayRe      = regexp.MustCompile(`(?i)pago\s*m.{0,2}nimo\s*:?\s*(-?\d[\d, ]*\.\d{2})\s*(-?\d[\d, ]*\.\d{2})`)
	interestRe    = regexp.MustCompile(`(?i)inter.{0,2}s\s*corrientes\s+(-?\d[\d, ]*\.\d{2})\s+(-?\d[\d, ]*\.\d{2})`)
	bonifiableRe  = regexp.MustCompile(`(?i)inter.{0,2}s\s*bonificable\s+(-?\d[\d, ]*\.\d{2})\s+(-?\d[\d, ]*\.\d{2})`)
	newBalanceRe  = regexp.MustCompile(`(?i)saldos\s*al\s*corte\s+(-?\d[\d, ]*\.\d{2})\s+(-?\d[\d, ]*\.\d{2})`)
	moraRe        = regexp.MustCompile(`(?i)monto\s*en\s*mora\s+(-?\d[\d, ]*\.\d{2})\s+(-?\d[\d, ]*\.\d{2})`)
	arrearsRe     = regexp.MustCompile(`(?i)pagos\s*atrasados\s+(\d+)\s+(\d+)`)
	overLimitRe   = regexp.MustCompile(`(?i)monto\s*en\s*exceso\s+(-?\d[\d, ]*\.\d{2})\s+(-?\d[\d, ]*\.\d{2})`)
	pointsPrevRe  = regexp.MustCompile(`(?i)saldo\s*en\s*puntos\s*anterior\s*:?\s*\+?(-?[\d,]+)`)
	pointsEarnRe  = regexp.MustCompile(`(?i)puntos\s*acreditados\s*:?\s*\+?(-?[\d,]+)`)
	pointsUsedRe  = regexp.MustCompile(`(?i)puntos\s*debitados\s*:?\s*\+?(-?[\d,]+)`)
	pointsAvailRe = regexp.MustCompile(`(?i)puntos\s*disponibles\s*:?\s*\+?(-?[\d,]+)`)

	referenceStartRe = regexp.MustCompile(`(\d{18,26})\s+(\d{2}/\d{2})\s`)
	dateStartRe      = regexp.MustCompile(`(\.\d{2})\s+(\d{2}/\d{2})\s`)
	rowStartRe       = regexp.MustCompile(`^(?:(\d{18,26})\s+)?(\d{2})/(\d{2})\s`)
	amountTokenRe    = regexp.MustCompile(`-?\d[\d, ]*\.\d{2}`)

	regionStartRe = regexp.MustCompile(`(?i)referencia\s*fecha\s*descripci.{0,2}n`)
	regionEndRe   = regexp.MustCompile(`(?i)inter.{0,2}s\s*corrientes`)
)

// Parse abre el PDF y delega en parseText.
func (m *Module) Parse(ctx context.Context, r io.Reader) (parsed *models.ParsedStatement, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			parsed = nil
			err = fmt.Errorf("%w: no se pudo interpretar el PDF", modules.ErrUnsupportedStatement)
		}
	}()

	data, err := io.ReadAll(io.LimitReader(r, maxPDFSize+1))
	if err != nil {
		return nil, fmt.Errorf("leer PDF: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: el PDF está vacío", modules.ErrUnsupportedStatement)
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
		return nil, fmt.Errorf("%w: el PDF no contiene texto", modules.ErrUnsupportedStatement)
	}
	return parseText(statementtext.DecodeLegacyEncoding(text.String()))
}

// parseText interpreta el texto extraído. Separado del PDF para testear con
// fixtures sintéticos (SEC-NO-PII: no se commitean estados de cuenta).
func parseText(text string) (*models.ParsedStatement, error) {
	flat := collapse(text)
	if !headerRe.MatchString(flat) {
		return nil, fmt.Errorf("%w: no es un estado de cuenta BANPRO", modules.ErrUnsupportedStatement)
	}

	// El encabezado imprime las etiquetas en una fila y las fechas en la
	// siguiente: se toman las dos primeras fechas antes de los movimientos.
	headerArea := flat
	if idx := regionStartRe.FindStringIndex(flat); idx != nil {
		headerArea = flat[:idx[0]]
	}
	dates := dateRe.FindAllStringSubmatch(headerArea, -1)
	if len(dates) < 2 {
		return nil, fmt.Errorf("%w: faltan fechas del encabezado", modules.ErrUnsupportedStatement)
	}
	periodEnd := isoDate(dates[0][1])
	paymentDue := isoDate(dates[1][1])
	if periodEnd == "" || paymentDue == "" {
		return nil, fmt.Errorf("%w: faltan fechas del encabezado", modules.ErrUnsupportedStatement)
	}

	parsed := &models.ParsedStatement{
		ModuleID:      ModuleID,
		ModuleVersion: ModuleVersion,
		Last4:         firstMatch(last4Re, flat),
		PeriodEnd:     periodEnd,
		// BANPRO imprime una sola fecha de pago (pago mínimo). El pago de contado
		// vence en la fecha de corte.
		MinPaymentDue:         paymentDue,
		FullPaymentDue:        periodEnd,
		InterestCurrent:       amount(firstMatch(interestRe, flat, 1)),
		InterestCurrentUSD:    amount(firstMatch(interestRe, flat, 2)),
		InterestBonifiable:    amount(firstMatch(bonifiableRe, flat, 1)),
		InterestBonifiableUSD: amount(firstMatch(bonifiableRe, flat, 2)),
		MoraAmount:            amount(firstMatch(moraRe, flat, 1)),
		MoraAmountUSD:         amount(firstMatch(moraRe, flat, 2)),
		ArrearsCount:          int(amount(firstMatch(arrearsRe, flat, 1))),
		OverLimitAmount:       amount(firstMatch(overLimitRe, flat, 1)),
		OverLimitAmountUSD:    amount(firstMatch(overLimitRe, flat, 2)),
		MinPayment:            amount(firstMatch(minPayRe, flat, 1)),
		MinPaymentUSD:         amount(firstMatch(minPayRe, flat, 2)),
		TotalPayment:          amount(firstMatch(totalPayRe, flat, 1)),
		TotalPaymentUSD:       amount(firstMatch(totalPayRe, flat, 2)),
		NewBalance:            amount(firstMatch(newBalanceRe, flat, 1)),
		NewBalanceUSD:         amount(firstMatch(newBalanceRe, flat, 2)),
		PointsPrevious:        int(amount(firstMatch(pointsPrevRe, flat, 1))),
		PointsEarned:          int(amount(firstMatch(pointsEarnRe, flat, 1))),
		PointsUsed:            int(amount(firstMatch(pointsUsedRe, flat, 1))),
		PointsAvailable:       int(amount(firstMatch(pointsAvailRe, flat, 1))),
	}
	parsed.Warnings = append(parsed.Warnings, models.ParseWarning{
		Code:    "single_payment_date",
		Message: "BANPRO imprime una sola fecha de pago: se usa también como fecha de pago de contado",
	})

	if previous := previousRe.FindStringSubmatch(flat); previous != nil {
		parsed.PreviousBalance = amount(previous[1])
		parsed.PreviousBalanceUSD = amount(previous[2])
	} else {
		parsed.Warnings = append(parsed.Warnings, models.ParseWarning{
			Code:    "missing_previous_balance",
			Message: "No se encontró el saldo anterior en el estado de cuenta",
		})
	}

	region := transactionRegion(flat)
	parsed.Transactions = parseRows(region, periodEnd)
	if len(parsed.Transactions) == 0 {
		return nil, fmt.Errorf("%w: no se encontraron movimientos", modules.ErrUnsupportedStatement)
	}

	parsed.Warnings = append(parsed.Warnings, sanityWarnings(parsed, flat)...)

	sort.SliceStable(parsed.Transactions, func(i, j int) bool {
		return parsed.Transactions[i].PostedAt < parsed.Transactions[j].PostedAt
	})
	return parsed, nil
}

// collapse unifica espacios y saltos de línea.
func collapse(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func transactionRegion(flat string) string {
	start := regionStartRe.FindStringIndex(flat)
	if start == nil {
		return flat
	}
	from := start[1]
	if end := regionEndRe.FindStringIndex(flat[from:]); end != nil {
		return flat[from : from+end[0]]
	}
	return flat[from:]
}

// parseRows extrae las filas del bloque de movimientos. Las filas pueden venir
// con referencia bancaria y sin ella (BANPRO omite la referencia en algunas).
func parseRows(region, periodEnd string) []models.ParsedTransaction {
	marked := referenceStartRe.ReplaceAllString(region, "|$1 $2 ")
	marked = dateStartRe.ReplaceAllString(marked, "$1|$2 ")
	blocks := strings.Split(marked, "|")

	out := make([]models.ParsedTransaction, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		dateMatch := rowStartRe.FindStringSubmatch(block)
		if dateMatch == nil {
			continue
		}
		month, _ := strconv.Atoi(dateMatch[2])
		day, _ := strconv.Atoi(dateMatch[3])
		postedAt := inferDate(periodEnd, month, day)
		if postedAt == "" {
			continue
		}
		rest := strings.TrimSpace(block[len(dateMatch[0]):])
		ref := dateMatch[1]
		concept, local, usd := splitRow(rest)
		if concept == "" {
			continue
		}
		externalRef := ""
		if ref != "" {
			externalRef = "banpro:" + ref
		}
		out = append(out, models.ParsedTransaction{
			PostedAt:         postedAt,
			Concept:          concept,
			Type:             statementtext.Classify(concept),
			AmountLocal:      local,
			AmountUSD:        usd,
			CurrencyOriginal: originalCurrency(local, usd),
			ExternalRef:      externalRef,
		})
	}
	return out
}

// splitRow separa concepto y montos. Toma el último monto de la fila y, si el
// anterior está pegado, los interpreta como local + USD.
func splitRow(rest string) (string, float64, float64) {
	positions := amountTokenRe.FindAllStringSubmatchIndex(rest, -1)
	if len(positions) == 0 {
		return "", 0, 0
	}
	last := positions[len(positions)-1]
	conceptEnd := last[0]
	local := amount(rest[last[0]:last[1]])
	var usd float64
	if len(positions) >= 2 {
		prev := positions[len(positions)-2]
		if strings.TrimSpace(rest[prev[1]:last[0]]) == "" {
			local = amount(rest[prev[0]:prev[1]])
			usd = amount(rest[last[0]:last[1]])
			conceptEnd = prev[0]
		}
	}
	concept := strings.TrimSpace(rest[:conceptEnd])
	concept = strings.Join(strings.Fields(concept), " ")
	if concept == "" {
		return "", 0, 0
	}
	return concept, local, usd
}

// inferDate completa el año de una fila MM/DD usando la fecha de corte.
func inferDate(periodEnd string, month, day int) string {
	corte, err := time.Parse("2006-01-02", periodEnd)
	if err != nil {
		return ""
	}
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return ""
	}
	year := corte.Year()
	if month > int(corte.Month()) {
		year--
	}
	value := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if value.Month() != time.Month(month) {
		return ""
	}
	return value.Format("2006-01-02")
}

// sanityWarnings compara los agregados impresos con la suma de movimientos.
func sanityWarnings(parsed *models.ParsedStatement, flat string) []models.ParseWarning {
	var warnings []models.ParseWarning

	if charges := chargesRe.FindStringSubmatch(flat); charges != nil {
		if diff := round2(sumCharges(parsed.Transactions) - math.Abs(amount(charges[1]))); math.Abs(diff) > 0.005 {
			warnings = append(warnings, models.ParseWarning{
				Code:    "charges_mismatch",
				Message: fmt.Sprintf("Los cargos impresos no coinciden con los movimientos (diferencia %.2f)", diff),
			})
		}
	}
	if payments := paymentsRe.FindStringSubmatch(flat); payments != nil {
		if diff := round2(sumPayments(parsed.Transactions) - amount(payments[1])); math.Abs(diff) > 0.005 {
			warnings = append(warnings, models.ParseWarning{
				Code:    "payments_mismatch",
				Message: fmt.Sprintf("Los pagos impresos no coinciden con los movimientos (diferencia %.2f)", diff),
			})
		}
	}
	if parsed.TotalPayment != 0 || parsed.NewBalance != 0 {
		expectedCash := round2(parsed.PreviousBalance + sumAll(parsed.Transactions))
		if diff := round2(expectedCash - parsed.TotalPayment); math.Abs(diff) > 0.005 {
			warnings = append(warnings, models.ParseWarning{
				Code:    "reconciliation_mismatch",
				Message: fmt.Sprintf("El pago de contado impreso no coincide con los movimientos (diferencia %.2f)", diff),
			})
		}
	}
	return warnings
}

func sumCharges(items []models.ParsedTransaction) float64 {
	var total float64
	for _, item := range items {
		if item.AmountLocal > 0 {
			total += item.AmountLocal
		}
	}
	return total
}

func sumPayments(items []models.ParsedTransaction) float64 {
	var total float64
	for _, item := range items {
		if item.AmountLocal < 0 {
			total += item.AmountLocal
		}
	}
	return total
}

func sumAll(items []models.ParsedTransaction) float64 {
	var total float64
	for _, item := range items {
		total += item.AmountLocal
	}
	return total
}

func firstMatch(re *regexp.Regexp, text string, group ...int) string {
	match := re.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	index := 1
	if len(group) > 0 {
		index = group[0]
	}
	if index >= len(match) {
		return ""
	}
	return match[index]
}

func isoDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := time.Parse("02/01/2006", value)
	if err != nil {
		return ""
	}
	return parsed.Format("2006-01-02")
}

func amount(value string) float64 {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "$", "")
	value = strings.ReplaceAll(value, "U", "")
	value = strings.ReplaceAll(value, " ", "")
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func originalCurrency(local, usd float64) string {
	if local == 0 && usd != 0 {
		return "USD"
	}
	return "NIO"
}
