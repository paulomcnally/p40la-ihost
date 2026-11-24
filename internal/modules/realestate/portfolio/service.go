package portfolio

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/modules"
)

// Service contiene la lógica de negocio del resumen (ARCH-MODULES).
type Service struct {
	storage *Storage
	states  modules.HomeStateProvider
	now     func() time.Time
}

// NewService construye el servicio con el proveedor de estados de viviendas.
func NewService(st *Storage, states modules.HomeStateProvider) *Service {
	return &Service{storage: st, states: states, now: time.Now}
}

// Summary calcula el resumen del mes indicado. Si year y month son 0, usa el
// mes actual.
func (s *Service) Summary(ctx context.Context, year, month int) (*Summary, error) {
	p, err := newPeriod(year, month, s.now())
	if err != nil {
		return nil, err
	}
	if s.states == nil {
		return nil, fmt.Errorf("proveedor de estados de viviendas no configurado")
	}
	homes, err := s.states.HomeStates(ctx)
	if err != nil {
		return nil, err
	}

	income, err := s.storage.IncomeByHome(ctx, p)
	if err != nil {
		return nil, err
	}
	expenses, pendingExpenses, err := s.storage.ExpensesByHome(ctx, p)
	if err != nil {
		return nil, err
	}
	pendingIncome, err := s.storage.PendingIncomeByHome(ctx, p)
	if err != nil {
		return nil, err
	}

	summary := &Summary{
		Year:        p.Year,
		Month:       p.Month,
		Homes:       make([]HomeSummary, 0, len(homes)),
		StateCounts: make(map[string]int),
	}
	totals := struct {
		income, expenses, net, pendingIncome, pendingExpenses map[string]float64
	}{
		income:          make(map[string]float64),
		expenses:        make(map[string]float64),
		net:             make(map[string]float64),
		pendingIncome:   make(map[string]float64),
		pendingExpenses: make(map[string]float64),
	}

	for _, home := range homes {
		homeNet := netFor(income, expenses, home.HomeID)
		summary.Homes = append(summary.Homes, HomeSummary{
			HomeID:          home.HomeID,
			Name:            home.Name,
			Address:         home.Address,
			Usage:           home.Usage,
			State:           home.State,
			Income:          toMoney(income[home.HomeID]),
			Expenses:        toMoney(expenses[home.HomeID]),
			Net:             toMoney(homeNet),
			PendingIncome:   toMoney(pendingIncome[home.HomeID]),
			PendingExpenses: toMoney(pendingExpenses[home.HomeID]),
		})
		summary.StateCounts[home.State]++

		mergeMoney(totals.income, income[home.HomeID])
		mergeMoney(totals.expenses, expenses[home.HomeID])
		mergeMoney(totals.net, homeNet)
		mergeMoney(totals.pendingIncome, pendingIncome[home.HomeID])
		mergeMoney(totals.pendingExpenses, pendingExpenses[home.HomeID])
	}

	summary.Totals = Totals{
		Income:          toMoney(totals.income),
		Expenses:        toMoney(totals.expenses),
		Net:             toMoney(totals.net),
		PendingIncome:   toMoney(totals.pendingIncome),
		PendingExpenses: toMoney(totals.pendingExpenses),
	}
	return summary, nil
}

// newPeriod valida el mes y calcula el rango [inicio, fin) en fechas ISO.
func newPeriod(year, month int, now time.Time) (period, error) {
	if year == 0 && month == 0 {
		year, month = now.Year(), int(now.Month())
	}
	if year < 2000 || year > 2100 || month < 1 || month > 12 {
		return period{}, ErrInvalidPeriod
	}
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	return period{
		Year:  year,
		Month: month,
		Start: start.Format("2006-01-02"),
		End:   end.Format("2006-01-02"),
	}, nil
}

// netFor calcula ingresos − gastos por moneda para una vivienda.
func netFor(income, expenses moneyByHome, homeID int64) map[string]float64 {
	out := make(map[string]float64)
	for currency, amount := range income[homeID] {
		out[currency] += amount
	}
	for currency, amount := range expenses[homeID] {
		out[currency] -= amount
	}
	return out
}

// mergeMoney suma importes por moneda al acumulador de totales.
func mergeMoney(dest map[string]float64, source map[string]float64) {
	for currency, amount := range source {
		dest[currency] += amount
	}
}

// toMoney convierte un mapa moneda→importe en una lista ordenada, omitiendo
// ceros. El frontend nunca recibe una suma entre monedas distintas.
func toMoney(byCurrency map[string]float64) []Money {
	out := make([]Money, 0, len(byCurrency))
	for currency, amount := range byCurrency {
		if amount > -0.005 && amount < 0.005 {
			continue
		}
		out = append(out, Money{Currency: currency, Amount: amount})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Currency < out[j].Currency })
	return out
}

// ---------------------------------------------------------------- finanzas casa

// HomeFinances calcula la serie de 12 meses y el detalle del mes de una casa.
func (s *Service) HomeFinances(ctx context.Context, homeID int64, year, month int) (*HomeFinances, error) {
	p, err := newPeriod(year, month, s.now())
	if err != nil {
		return nil, err
	}
	if s.states == nil {
		return nil, fmt.Errorf("proveedor de estados de viviendas no configurado")
	}
	states, err := s.states.HomeStates(ctx)
	if err != nil {
		return nil, err
	}
	var home *modules.HomeState
	for i := range states {
		if states[i].HomeID == homeID {
			home = &states[i]
			break
		}
	}
	if home == nil {
		return nil, ErrHomeNotFound
	}

	selectedStart := time.Date(p.Year, time.Month(p.Month), 1, 0, 0, 0, 0, time.UTC)
	windowStart := selectedStart.AddDate(0, -11, 0)
	data, err := s.storage.HomeFinanceWindow(ctx, homeID, windowStart.Format("2006-01-02"), p.End,
		windowStart.Year(), p.Year)
	if err != nil {
		return nil, err
	}

	months := make([]time.Time, 0, 12)
	monthKeys := make(map[string]bool, 12)
	for i := 11; i >= 0; i-- {
		m := selectedStart.AddDate(0, -i, 0)
		months = append(months, m)
		monthKeys[m.Format("2006-01")] = true
	}
	selectedKey := selectedStart.Format("2006-01")

	incomeByMonth := make(map[string]map[string]float64)
	expensesByMonth := make(map[string]map[string]float64)
	servicesByMonth := make(map[string]map[string]float64)
	maintenanceByMonth := make(map[string]map[string]float64)
	commissionByMonth := make(map[string]map[string]float64)

	for _, row := range data.Payments {
		key := monthOf(row.PaidAt)
		if monthKeys[key] {
			addCurrency(incomeByMonth, key, row.Currency, row.Amount)
		}
	}
	for _, row := range data.Reservations {
		key := monthOf(row.StartsAt)
		if monthKeys[key] {
			addCurrency(incomeByMonth, key, row.Currency, row.Amount)
		}
	}
	for _, row := range data.Bills {
		key := fmt.Sprintf("%04d-%02d", row.Year, row.Month)
		if monthKeys[key] {
			addCurrency(expensesByMonth, key, row.Currency, row.Amount)
			addCurrency(servicesByMonth, key, row.Currency, row.Amount)
		}
	}
	for _, row := range data.Maintenances {
		key := monthOf(row.PerformedAt)
		if monthKeys[key] {
			addCurrency(expensesByMonth, key, row.Currency, row.Amount)
			addCurrency(maintenanceByMonth, key, row.Currency, row.Amount)
		}
	}
	for _, row := range data.Commissions {
		key := monthOf(row.StartDate)
		amount := commissionAmount(row)
		if monthKeys[key] && amount > 0 {
			addCurrency(expensesByMonth, key, row.Currency, amount)
			addCurrency(commissionByMonth, key, row.Currency, amount)
		}
	}

	items := make([]FinanceItem, 0)
	pendingExpenseMap := make(map[string]float64)
	for _, row := range data.Payments {
		if monthOf(row.PaidAt) != selectedKey {
			continue
		}
		leaseID := row.LeaseID
		items = append(items, FinanceItem{
			Type:    "payment",
			Date:    row.PaidAt,
			Detail:  row.Reference,
			Status:  "paid",
			Method:  row.Method,
			Amounts: []Money{{Currency: row.Currency, Amount: row.Amount}},
			LeaseID: &leaseID,
		})
	}
	for _, row := range data.Reservations {
		if monthOf(row.StartsAt) != selectedKey {
			continue
		}
		reservationID := row.ID
		items = append(items, FinanceItem{
			Type:          "reservation",
			Date:          row.StartsAt,
			Detail:        row.Reference,
			Status:        row.Status,
			Amounts:       []Money{{Currency: row.Currency, Amount: row.Amount}},
			ReservationID: &reservationID,
		})
	}
	for _, row := range data.Bills {
		if fmt.Sprintf("%04d-%02d", row.Year, row.Month) != selectedKey {
			continue
		}
		billID, serviceID := row.ID, row.ServiceID
		if row.Status == "pending" {
			pendingExpenseMap[row.Currency] += row.Amount
		}
		items = append(items, FinanceItem{
			Type:        "bill",
			Date:        fmt.Sprintf("%04d-%02d-01", row.Year, row.Month),
			Description: row.ServiceName,
			Detail:      row.Institution,
			Status:      row.Status,
			Amounts:     []Money{{Currency: row.Currency, Amount: row.Amount}},
			BillID:      &billID,
			ServiceID:   &serviceID,
		})
	}

	maintenanceIndex := make(map[int64]int)
	for _, row := range data.Maintenances {
		if monthOf(row.PerformedAt) != selectedKey {
			continue
		}
		if idx, ok := maintenanceIndex[row.ID]; ok {
			items[idx].Amounts = mergeAmounts(items[idx].Amounts, row.Currency, row.Amount)
			continue
		}
		maintenanceID := row.ID
		maintenanceIndex[row.ID] = len(items)
		items = append(items, FinanceItem{
			Type:          "maintenance",
			Date:          row.PerformedAt,
			Description:   row.Description,
			Kind:          row.Kind,
			Area:          row.Area,
			Status:        "done",
			Amounts:       []Money{{Currency: row.Currency, Amount: row.Amount}},
			MaintenanceID: &maintenanceID,
		})
	}
	for _, row := range data.Commissions {
		if monthOf(row.StartDate) != selectedKey || commissionAmount(row) <= 0 {
			continue
		}
		leaseID := row.LeaseID
		detail := row.Type
		if row.Type == "percent" {
			detail = fmt.Sprintf("%.2f%%", row.Percent)
		}
		items = append(items, FinanceItem{
			Type:        "commission",
			Date:        row.StartDate,
			Description: row.AgencyName,
			Detail:      detail,
			Status:      "paid",
			Amounts:     []Money{{Currency: row.Currency, Amount: round2(commissionAmount(row))}},
			LeaseID:     &leaseID,
		})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Date > items[j].Date })

	pendingIncome, err := s.storage.PendingIncomeByHome(ctx, p)
	if err != nil {
		return nil, err
	}

	result := &HomeFinances{
		HomeID:          home.HomeID,
		HomeName:        home.Name,
		Address:         home.Address,
		Usage:           home.Usage,
		State:           home.State,
		Year:            p.Year,
		Month:           p.Month,
		Months:          make([]FinanceMonth, 0, len(months)),
		Income:          toMoney(incomeByMonth[selectedKey]),
		Expenses:        toMoney(expensesByMonth[selectedKey]),
		Net:             toMoney(netCurrencies(incomeByMonth[selectedKey], expensesByMonth[selectedKey])),
		PendingIncome:   toMoney(pendingIncome[homeID]),
		PendingExpenses: toMoney(pendingExpenseMap),
		Breakdown:       make([]FinanceBreakdown, 0),
		Items:           items,
	}
	for _, m := range months {
		key := m.Format("2006-01")
		result.Months = append(result.Months, FinanceMonth{
			Year:     m.Year(),
			Month:    int(m.Month()),
			Income:   toMoney(incomeByMonth[key]),
			Expenses: toMoney(expensesByMonth[key]),
			Net:      toMoney(netCurrencies(incomeByMonth[key], expensesByMonth[key])),
		})
	}

	breakdowns := []struct {
		kind string
		data map[string]map[string]float64
	}{
		{"services", servicesByMonth},
		{"maintenance", maintenanceByMonth},
		{"commission", commissionByMonth},
	}
	for _, item := range breakdowns {
		amounts := toMoney(item.data[selectedKey])
		if len(amounts) == 0 {
			continue
		}
		result.Breakdown = append(result.Breakdown, FinanceBreakdown{Kind: item.kind, Amounts: amounts})
	}

	result.Currencies = collectCurrencies(
		[]map[string]map[string]float64{incomeByMonth, expensesByMonth},
		pendingIncome[homeID], pendingExpenseMap,
	)
	return result, nil
}

// monthOf extrae YYYY-MM de una fecha ISO (date o datetime).
func monthOf(value string) string {
	if len(value) < 7 {
		return ""
	}
	return value[:7]
}

// addCurrency acumula un importe en la matriz mes → moneda → total.
func addCurrency(dest map[string]map[string]float64, monthKey, currency string, amount float64) {
	if dest[monthKey] == nil {
		dest[monthKey] = make(map[string]float64)
	}
	dest[monthKey][currency] += amount
}

// netCurrencies calcula ingresos − gastos por moneda.
func netCurrencies(income, expenses map[string]float64) map[string]float64 {
	out := make(map[string]float64)
	for currency, amount := range income {
		out[currency] += amount
	}
	for currency, amount := range expenses {
		out[currency] -= amount
	}
	return out
}

// commissionAmount calcula la comisión única del contrato (REQ-022).
func commissionAmount(row CommissionRow) float64 {
	if row.Type == "fixed" {
		return round2(row.FixedAmount)
	}
	return round2(row.Percent / 100 * row.RentAmount)
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

// mergeAmounts suma un importe a una lista de Money existente.
func mergeAmounts(list []Money, currency string, amount float64) []Money {
	for i := range list {
		if list[i].Currency == currency {
			list[i].Amount += amount
			return list
		}
	}
	return append(list, Money{Currency: currency, Amount: amount})
}

// collectCurrencies devuelve las monedas presentes, ordenadas.
func collectCurrencies(series []map[string]map[string]float64, extra ...map[string]float64) []string {
	seen := make(map[string]bool)
	for _, byMonth := range series {
		for _, byCurrency := range byMonth {
			for currency := range byCurrency {
				seen[currency] = true
			}
		}
	}
	for _, byCurrency := range extra {
		for currency := range byCurrency {
			seen[currency] = true
		}
	}
	out := make([]string, 0, len(seen))
	for currency := range seen {
		out = append(out, currency)
	}
	sort.Strings(out)
	return out
}
