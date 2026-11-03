package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// billPeriod representa un mes calendario (SPEC-120).
type billPeriod struct {
	Year  int
	Month int
}

func (p billPeriod) key() string {
	return fmt.Sprintf("%04d-%02d", p.Year, p.Month)
}

func (p billPeriod) start() time.Time {
	return time.Date(p.Year, time.Month(p.Month), 1, 0, 0, 0, 0, time.UTC)
}

func (p billPeriod) end() time.Time {
	return time.Date(p.Year, time.Month(p.Month)+1, 0, 0, 0, 0, 0, time.UTC)
}

// add devuelve el período desplazado n meses (n puede ser negativo).
func (p billPeriod) add(n int) billPeriod {
	m := p.Month - 1 + n
	year := p.Year + m/12
	m %= 12
	if m < 0 {
		m += 12
		year--
	}
	return billPeriod{Year: year, Month: m + 1}
}

// ListServiceBillsByPeriod devuelve las facturas del período efectivo con
// período y fecha efectiva calculados (SPEC-120).
func (s *BillService) ListServiceBillsByPeriod(ctx context.Context, year, month int, homeID *int64) ([]models.ServiceBillDetail, error) {
	if year < 1900 || year > 2100 {
		return nil, fmt.Errorf("año inválido")
	}
	if month < 1 || month > 12 {
		return nil, fmt.Errorf("mes inválido")
	}
	bills, err := s.storage.ListServiceBillsByPeriod(ctx, year, month, homeID)
	if err != nil {
		return nil, err
	}
	if bills == nil {
		bills = []models.ServiceBillDetail{}
	}
	for i := range bills {
		applyServiceBillPeriod(&bills[i])
	}
	return bills, nil
}

// ServiceBillsSummary devuelve el agregado por período y moneda del rango
// solicitado (SPEC-120).
func (s *BillService) ServiceBillsSummary(ctx context.Context, from, to string, homeID *int64) (*models.ServiceBillSummaryResponse, error) {
	fromPeriod, err := parseBillPeriod(from)
	if err != nil {
		return nil, fmt.Errorf("período inicial inválido")
	}
	toPeriod, err := parseBillPeriod(to)
	if err != nil {
		return nil, fmt.Errorf("período final inválido")
	}
	if fromPeriod.key() > toPeriod.key() {
		return nil, fmt.Errorf("el período inicial no puede ser posterior al final")
	}
	rows, err := s.storage.ServiceBillsSummary(ctx, fromPeriod.key(), toPeriod.key(), homeID)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []models.ServiceBillSummaryRow{}
	}
	return &models.ServiceBillSummaryResponse{
		From:    fromPeriod.key(),
		To:      toPeriod.key(),
		Periods: rows,
	}, nil
}

// ServiceBillsProjection calcula los próximos meses separando facturas
// pendientes confirmadas de estimaciones por servicios recurrentes
// (SPEC-120).
func (s *BillService) ServiceBillsProjection(ctx context.Context, fromYear, fromMonth, months int, homeID *int64) (*models.ServiceProjectionResponse, error) {
	if fromYear < 1900 || fromYear > 2100 || fromMonth < 1 || fromMonth > 12 {
		return nil, fmt.Errorf("período inicial inválido")
	}
	if months < 1 || months > 24 {
		return nil, fmt.Errorf("cantidad de meses inválida")
	}

	start := billPeriod{Year: fromYear, Month: fromMonth}
	periods := make([]billPeriod, months)
	periodIndex := make(map[string]int, months)
	for i := 0; i < months; i++ {
		periods[i] = start.add(i)
		periodIndex[periods[i].key()] = i
	}
	last := periods[len(periods)-1]

	bills, err := s.storage.ListServiceBillsForProjection(ctx, start.key(), last.key(), homeID)
	if err != nil {
		return nil, err
	}
	services, err := s.storage.ListRecurringServicesForProjection(ctx, homeID)
	if err != nil {
		return nil, err
	}

	type projectionAcc struct {
		symbol    string
		confirmed float64
		estimated float64
		items     []models.ServiceProjectionItem
	}
	accs := make(map[string]map[string]*projectionAcc)
	billedServices := make(map[string]map[int64]bool)

	getAcc := func(periodKey, code, symbol string) *projectionAcc {
		if accs[periodKey] == nil {
			accs[periodKey] = make(map[string]*projectionAcc)
		}
		acc := accs[periodKey][code]
		if acc == nil {
			acc = &projectionAcc{}
			accs[periodKey][code] = acc
		}
		if acc.symbol == "" {
			acc.symbol = symbol
		}
		return acc
	}

	for _, b := range bills {
		period := effectiveBillPeriod(b.Year, b.Month, b.DueDate)
		if _, ok := periodIndex[period.key()]; !ok {
			continue
		}
		if billedServices[period.key()] == nil {
			billedServices[period.key()] = make(map[int64]bool)
		}
		billedServices[period.key()][b.ServiceID] = true

		if b.Status != "pending" {
			continue
		}
		code := b.CurrencyCode
		if code == "" {
			code = "NIO"
		}
		acc := getAcc(period.key(), code, b.CurrencySymbol)
		acc.confirmed += b.Amount
		acc.items = append(acc.items, models.ServiceProjectionItem{
			ServiceID:    b.ServiceID,
			Name:         b.ServiceName,
			Amount:       b.Amount,
			Source:       "bill",
			CurrencyCode: code,
		})
	}

	for _, svc := range services {
		code := svc.CurrencyCode
		if code == "" {
			code = "NIO"
		}
		for _, period := range periods {
			if svc.Frequency == FrequencyYearly && period.Month != 1 {
				continue
			}
			if !serviceActiveInPeriod(svc, period) {
				continue
			}
			if billedServices[period.key()][svc.ID] {
				continue
			}
			if svc.SuggestedAmount <= 0 {
				continue
			}
			acc := getAcc(period.key(), code, svc.CurrencySymbol)
			acc.estimated += svc.SuggestedAmount
			acc.items = append(acc.items, models.ServiceProjectionItem{
				ServiceID:    svc.ID,
				Name:         svc.Name,
				Amount:       svc.SuggestedAmount,
				Source:       "estimate",
				CurrencyCode: code,
			})
		}
	}

	out := make([]models.ServiceProjectionPeriod, 0)
	for _, period := range periods {
		byCurrency := accs[period.key()]
		codes := make([]string, 0, len(byCurrency))
		for code := range byCurrency {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		for _, code := range codes {
			acc := byCurrency[code]
			if acc.confirmed == 0 && acc.estimated == 0 && len(acc.items) == 0 {
				continue
			}
			// Un servicio puede tener más de una factura en el mismo período
			// efectivo (p. ej. una mensual y otra con vencimiento distinto).
			// Se consolidan por servicio y origen para no mostrar duplicados.
			merged := make([]models.ServiceProjectionItem, 0, len(acc.items))
			itemIndex := make(map[string]int)
			for _, item := range acc.items {
				key := fmt.Sprintf("%d|%s", item.ServiceID, item.Source)
				if index, ok := itemIndex[key]; ok {
					merged[index].Amount += item.Amount
					continue
				}
				itemIndex[key] = len(merged)
				merged = append(merged, item)
			}
			acc.items = merged
			sort.Slice(acc.items, func(i, j int) bool {
				if acc.items[i].Source != acc.items[j].Source {
					return acc.items[i].Source == "bill"
				}
				return acc.items[i].Name < acc.items[j].Name
			})
			out = append(out, models.ServiceProjectionPeriod{
				Period:         period.key(),
				Year:           period.Year,
				Month:          period.Month,
				CurrencyCode:   code,
				CurrencySymbol: acc.symbol,
				Confirmed:      roundMoney(acc.confirmed),
				Estimated:      roundMoney(acc.estimated),
				Total:          roundMoney(acc.confirmed + acc.estimated),
				Services:       acc.items,
			})
		}
	}

	return &models.ServiceProjectionResponse{
		From:         start.key(),
		Months:       months,
		Periods:      out,
		EstimateNote: true,
	}, nil
}

// applyServiceBillPeriod completa período y fecha efectiva de una factura
// (SPEC-120 REQ-007).
func applyServiceBillPeriod(b *models.ServiceBillDetail) {
	if b.DueDate != nil {
		if period, ok := billPeriodFromDate(*b.DueDate); ok {
			b.PeriodYear = period.Year
			b.PeriodMonth = period.Month
			b.PeriodDate = datePrefix(*b.DueDate)
			b.DateEstimated = false
			return
		}
	}

	year, month := b.Year, b.Month
	if month < 1 || month > 12 {
		// Factura anual sin due_date: se ubica en enero, igual que el scheduler.
		month = 1
	}
	day := 1
	if b.BillingDay != nil {
		day = *b.BillingDay
		last := daysInMonth(year, time.Month(month))
		if day < 1 {
			day = 1
		}
		if day > last {
			day = last
		}
	}
	b.PeriodYear = year
	b.PeriodMonth = month
	b.PeriodDate = fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	b.DateEstimated = true
}

// effectiveBillPeriod devuelve el período efectivo sin construir la fecha.
func effectiveBillPeriod(year, month int, dueDate *string) billPeriod {
	if dueDate != nil {
		if period, ok := billPeriodFromDate(*dueDate); ok {
			return period
		}
	}
	if month >= 1 && month <= 12 {
		return billPeriod{Year: year, Month: month}
	}
	return billPeriod{Year: year, Month: 1}
}

// serviceActiveInPeriod indica si el servicio recurrente está vigente en el
// período (start_date/end_date inclusivos).
func serviceActiveInPeriod(svc models.ProjectionServiceRow, period billPeriod) bool {
	periodStart := period.start()
	periodEnd := period.end()
	if svc.StartDate != nil {
		if d, ok := parseDatePrefix(*svc.StartDate); ok && d.After(periodEnd) {
			return false
		}
	}
	if svc.EndDate != nil {
		if d, ok := parseDatePrefix(*svc.EndDate); ok && d.Before(periodStart) {
			return false
		}
	}
	return true
}

func parseBillPeriod(value string) (billPeriod, error) {
	t, err := time.Parse("2006-01", strings.TrimSpace(value))
	if err != nil {
		return billPeriod{}, err
	}
	return billPeriod{Year: t.Year(), Month: int(t.Month())}, nil
}

func billPeriodFromDate(value string) (billPeriod, bool) {
	value = strings.TrimSpace(value)
	if len(value) < 7 {
		return billPeriod{}, false
	}
	t, err := time.Parse("2006-01", value[:7])
	if err != nil {
		return billPeriod{}, false
	}
	return billPeriod{Year: t.Year(), Month: int(t.Month())}, true
}

// datePrefix normaliza fechas ISO a YYYY-MM-DD o YYYY-MM.
func datePrefix(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}

func parseDatePrefix(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if len(value) >= 10 {
		if t, err := time.Parse("2006-01-02", value[:10]); err == nil {
			return t, true
		}
	}
	if len(value) >= 7 {
		if t, err := time.Parse("2006-01", value[:7]); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func roundMoney(value float64) float64 {
	return float64(int64(value*100+0.5)) / 100
}
