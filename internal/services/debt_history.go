package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// History devuelve la evolución mensual de deudas para los últimos `months`
// meses, con una serie y totales por moneda (SPEC-105).
//
// La métrica es la cantidad de deudas abiertas al cierre de cada mes. Una deuda
// deja de contar desde el mes en que se cierra (finalizada, sin cuotas
// pendientes con pagos, o eliminada), nunca retroactivamente.
func (s *DebtService) History(ctx context.Context, months int, now time.Time) (*models.DebtHistory, error) {
	if months < 1 {
		months = 1
	}
	if months > 24 {
		months = 24
	}

	rows, err := s.debts.ListForHistory(ctx)
	if err != nil {
		return nil, err
	}

	current := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	first := current.AddDate(0, -(months - 1), 0)
	monthKeys := make([]string, months)
	for i := 0; i < months; i++ {
		monthKeys[i] = first.AddDate(0, i, 0).Format("2006-01")
	}
	currentKey := monthKeys[months-1]

	type currencyAcc struct {
		code   string
		symbol string
		series []models.DebtHistoryPoint
	}
	byCurrency := make(map[string]*currencyAcc)

	for _, row := range rows {
		code := row.CurrencyCode
		if code == "" {
			code = fmt.Sprintf("currency-%d", row.CurrencyID)
		}
		acc, ok := byCurrency[code]
		if !ok {
			series := make([]models.DebtHistoryPoint, months)
			for i, key := range monthKeys {
				series[i] = models.DebtHistoryPoint{Month: key}
			}
			acc = &currencyAcc{code: code, symbol: row.CurrencySymbol, series: series}
			byCurrency[code] = acc
		}

		startMonth := monthPrefix(row.StartDate)
		if startMonth == "" {
			continue
		}
		closing := closingMonth(row)

		for i, key := range monthKeys {
			if startMonth > key {
				continue
			}
			if closing != "" && closing <= key {
				continue
			}
			acc.series[i].OpenCount++
			acc.series[i].OutstandingAmount += outstandingIn(row, key)
		}
		if closing != "" && closing >= monthKeys[0] && closing <= currentKey {
			for i, key := range monthKeys {
				if key == closing {
					acc.series[i].ClosedCount++
					break
				}
			}
		}
	}

	codes := make([]string, 0, len(byCurrency))
	for code, acc := range byCurrency {
		if hasActivity(acc.series) {
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)

	result := &models.DebtHistory{
		Months:      months,
		GeneratedAt: now.UTC(),
		Currencies:  make([]models.DebtHistoryCurrency, 0, len(codes)),
	}

	for _, code := range codes {
		acc := byCurrency[code]
		for i := range acc.series {
			acc.series[i].OutstandingAmount = math.Round(acc.series[i].OutstandingAmount*100) / 100
		}
		summary := models.DebtHistorySummary{
			StartCount:  acc.series[0].OpenCount,
			EndCount:    acc.series[len(acc.series)-1].OpenCount,
			ClosedCount: 0,
		}
		for _, p := range acc.series {
			summary.ClosedCount += p.ClosedCount
		}
		summary.ChangeCount = summary.EndCount - summary.StartCount
		if summary.StartCount > 0 {
			pct := math.Round(float64(summary.ChangeCount)/float64(summary.StartCount)*100*100) / 100
			summary.ChangePct = &pct
		}

		summary.StartAmount = acc.series[0].OutstandingAmount
		summary.EndAmount = acc.series[len(acc.series)-1].OutstandingAmount
		summary.ChangeAmount = math.Round((summary.EndAmount-summary.StartAmount)*100) / 100
		if summary.StartAmount > 0 {
			pctAmount := math.Round(summary.ChangeAmount/summary.StartAmount*100*100) / 100
			summary.ChangePctAmount = &pctAmount
		}

		// El saldo pendiente actual es la suma de pending_amount de las deudas
		// que la serie considera abiertas en el mes en curso.
		for _, row := range rows {
			rowCode := row.CurrencyCode
			if rowCode == "" {
				rowCode = fmt.Sprintf("currency-%d", row.CurrencyID)
			}
			if rowCode != code || !isOpenIn(row, currentKey) {
				continue
			}
			summary.PendingAmountCurrent += row.PendingAmount
		}
		summary.PendingAmountCurrent = math.Round(summary.PendingAmountCurrent*100) / 100

		result.Currencies = append(result.Currencies, models.DebtHistoryCurrency{
			Code:    acc.code,
			Symbol:  acc.symbol,
			Series:  acc.series,
			Summary: summary,
		})
	}

	return result, nil
}

// isOpenIn indica si la deuda cuenta como abierta en el mes indicado.
// Reutiliza las mismas reglas que la construcción de la serie.
func isOpenIn(row storage.DebtHistoryRow, month string) bool {
	start := monthPrefix(row.StartDate)
	if start == "" || start > month {
		return false
	}
	closing := closingMonth(row)
	return closing == "" || closing > month
}

// closingMonth devuelve el mes de cierre (YYYY-MM) de una deuda, o "" si sigue
// abierta. La prioridad es deleted_at > último pago > updated_at > fin derivado.
func closingMonth(row storage.DebtHistoryRow) string {
	if row.DeletedAt.Valid {
		return row.DeletedAt.Time.Format("2006-01")
	}

	closed := row.Status == DebtStatusFinished || (row.PendingCount == 0 && row.PaidCount > 0)
	if !closed {
		return ""
	}

	if m := monthPrefix(row.LastPaidAt); m != "" {
		return m
	}
	if !row.UpdatedAt.IsZero() {
		return row.UpdatedAt.Format("2006-01")
	}
	return derivedEndMonth(row.StartDate, row.InstallmentsTotal)
}

// monthPrefix extrae YYYY-MM de strings con los formatos heterogéneos de
// paid_at/start_date (SPEC-104). No parsea la fecha completa.
func monthPrefix(value string) string {
	if len(value) < 7 {
		return ""
	}
	prefix := value[:7]
	if prefix[4] != '-' {
		return ""
	}
	return prefix
}

// derivedEndMonth calcula start_date + installments_total meses como fallback.
func derivedEndMonth(start string, installments int) string {
	t, err := time.Parse("2006-01-02", start)
	if err != nil {
		return ""
	}
	if installments < 1 {
		installments = 1
	}
	return t.AddDate(0, installments, 0).Format("2006-01")
}

func hasActivity(series []models.DebtHistoryPoint) bool {
	for _, p := range series {
		if p.OpenCount > 0 || p.ClosedCount > 0 || p.OutstandingAmount > 0 {
			return true
		}
	}
	return false
}

// outstandingIn devuelve el monto adeudado de una deuda al cierre del mes
// indicado (SPEC-106): total de cuotas no eliminadas menos lo pagado hasta ese
// mes. Si la deuda no tiene cuotas, se usa su total. Clampea a 0.
func outstandingIn(row storage.DebtHistoryRow, month string) float64 {
	base := row.BillsTotal
	if base <= 0 {
		base = row.Total
	}
	if base <= 0 {
		return 0
	}
	var paid float64
	for paidMonth, amount := range row.PaidByMonth {
		if paidMonth != "" && paidMonth <= month {
			paid += amount
		}
	}
	outstanding := base - paid
	if outstanding < 0 {
		return 0
	}
	return outstanding
}
