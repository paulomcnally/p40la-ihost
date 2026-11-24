package portfolio

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/modules"
)

type fakeStates []modules.HomeState

func (f fakeStates) HomeStates(ctx context.Context) ([]modules.HomeState, error) {
	return f, nil
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := db.OpenDB(filepath.Join(t.TempDir(), "portfolio.db"),
		filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatalf("abrir db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func mustExec(t *testing.T, database *sql.DB, query string, args ...any) int64 {
	t.Helper()
	res, err := database.Exec(query, args...)
	if err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
	id, _ := res.LastInsertId()
	return id
}

func money(list []Money, currency string) float64 {
	for _, item := range list {
		if item.Currency == currency {
			return item.Amount
		}
	}
	return 0
}

func TestSummaryByHomeAndCurrency(t *testing.T) {
	database := newTestDB(t)
	ctx := context.Background()

	nioID := 0
	if err := database.QueryRow(`SELECT id FROM currencies WHERE code = 'NIO'`).Scan(&nioID); err != nil {
		t.Fatalf("moneda NIO: %v", err)
	}
	usdID := 0
	if err := database.QueryRow(`SELECT id FROM currencies WHERE code = 'USD'`).Scan(&usdID); err != nil {
		t.Fatalf("moneda USD: %v", err)
	}

	homeRented := mustExec(t, database, `INSERT INTO homes (name, address, usage) VALUES (?, ?, ?)`,
		"Casa Renta Demo", "Calle Demo 100", "rental")
	homeReserved := mustExec(t, database, `INSERT INTO homes (name, usage) VALUES (?, ?)`,
		"Casa Reservada Demo", "rental")
	homeInhabited := mustExec(t, database, `INSERT INTO homes (name, usage) VALUES (?, ?)`,
		"Casa Habitada Demo", "inhabited")
	homeUnused := mustExec(t, database, `INSERT INTO homes (name, usage) VALUES (?, ?)`,
		"Terreno Demo", "unused")

	// Gasto: factura de servicio del mes (NIO, pendiente).
	serviceID := mustExec(t, database, `
		INSERT INTO services (home_id, name, currency_id, frequency, suggested_amount, icon_key)
		VALUES (?, ?, ?, 'monthly', 0, 'bolt')`, homeRented, "Luz Demo", nioID)
	mustExec(t, database, `INSERT INTO bills (service_id, year, month, amount, status) VALUES (?, 2026, 11, 1200, 'pending')`, serviceID)

	// Gasto: reparación del terreno (USD).
	maintenanceID := mustExec(t, database, `
		INSERT INTO home_maintenances (home_id, performed_at, kind, area, description)
		VALUES (?, '2026-11-10', 'corrective', 'other', 'Reparación demo')`, homeUnused)
	mustExec(t, database, `
		INSERT INTO home_maintenance_items (maintenance_id, item_type, description, quantity, unit, amount, currency_id)
		VALUES (?, 'labor', 'Mano de obra demo', 1, 'unit', 300, ?)`, maintenanceID, usdID)

	// Ingreso: contrato activo, cargo de 500 y pago parcial de 300.
	personID := mustExec(t, database, `INSERT INTO rental_people (full_name, email) VALUES (?, ?)`,
		"Inquilino Demo", "demo@example.com")
	leaseID := mustExec(t, database, `
		INSERT INTO rental_leases (home_id, status, start_date, end_date, term_months, rent_amount, currency, payment_day)
		VALUES (?, 'active', '2026-01-01', '2026-12-31', 12, 500, 'USD', 5)`, homeRented)
	mustExec(t, database, `
		INSERT INTO rental_charges (lease_id, kind, period, description, amount, currency, due_date, status)
		VALUES (?, 'rent', '2026-11', 'Renta demo', 500, 'USD', '2026-11-05', 'pending')`, leaseID)
	mustExec(t, database, `
		INSERT INTO rental_payments (lease_id, paid_at, amount, currency, method)
		VALUES (?, '2026-11-06', 300, 'USD', 'transfer')`, leaseID)

	// Ingreso: seña de apartado de la casa reservada.
	mustExec(t, database, `
		INSERT INTO rental_reservations (home_id, applicant_person_id, status, amount, currency, starts_at, expires_at)
		VALUES (?, ?, 'active', 100, 'USD', '2026-11-10T10:00:00Z', '2026-12-10T10:00:00Z')`,
		homeReserved, personID)

	states := fakeStates{
		{HomeID: homeRented, Name: "Casa Renta Demo", Usage: "rental", State: "rented"},
		{HomeID: homeReserved, Name: "Casa Reservada Demo", Usage: "rental", State: "reserved"},
		{HomeID: homeInhabited, Name: "Casa Habitada Demo", Usage: "inhabited", State: "inhabited"},
		{HomeID: homeUnused, Name: "Terreno Demo", Usage: "unused", State: "unused"},
	}
	svc := NewService(NewStorage(database), states)
	svc.now = func() time.Time { return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) }

	summary, err := svc.Summary(ctx, 2026, 11)
	if err != nil {
		t.Fatalf("resumen: %v", err)
	}
	if len(summary.Homes) != 4 {
		t.Fatalf("casas = %d, esperaba 4", len(summary.Homes))
	}

	byID := make(map[int64]HomeSummary)
	for _, home := range summary.Homes {
		byID[home.HomeID] = home
	}

	rented := byID[homeRented]
	if got := money(rented.Income, "USD"); got != 300 {
		t.Fatalf("ingreso USD casa rentada = %v, esperaba 300", got)
	}
	if got := money(rented.Expenses, "NIO"); got != 1200 {
		t.Fatalf("gasto NIO casa rentada = %v, esperaba 1200", got)
	}
	if got := money(rented.Net, "NIO"); got != -1200 {
		t.Fatalf("neto NIO casa rentada = %v, esperaba -1200", got)
	}
	if got := money(rented.Net, "USD"); got != 300 {
		t.Fatalf("neto USD casa rentada = %v, esperaba 300", got)
	}
	if got := money(rented.PendingIncome, "USD"); got != 200 {
		t.Fatalf("por cobrar USD casa rentada = %v, esperaba 200", got)
	}
	if got := money(rented.PendingExpenses, "NIO"); got != 1200 {
		t.Fatalf("por pagar NIO casa rentada = %v, esperaba 1200", got)
	}

	reserved := byID[homeReserved]
	if got := money(reserved.Income, "USD"); got != 100 {
		t.Fatalf("ingreso USD casa reservada = %v, esperaba 100", got)
	}

	inhabited := byID[homeInhabited]
	if len(inhabited.Income) != 0 || len(inhabited.Expenses) != 0 {
		t.Fatalf("casa habitada sin movimientos, obtuve %+v", inhabited)
	}

	unused := byID[homeUnused]
	if got := money(unused.Expenses, "USD"); got != 300 {
		t.Fatalf("gasto USD terreno = %v, esperaba 300", got)
	}
	if len(unused.Income) != 0 {
		t.Fatalf("terreno sin ingresos, obtuve %+v", unused.Income)
	}

	// Totales: ingresos USD 400; gastos NIO 1200 + USD 300; neto USD = 0 (se omite).
	if got := money(summary.Totals.Income, "USD"); got != 400 {
		t.Fatalf("total ingreso USD = %v, esperaba 400", got)
	}
	if got := money(summary.Totals.Expenses, "NIO"); got != 1200 {
		t.Fatalf("total gasto NIO = %v, esperaba 1200", got)
	}
	if got := money(summary.Totals.Expenses, "USD"); got != 300 {
		t.Fatalf("total gasto USD = %v, esperaba 300", got)
	}
	if got := money(summary.Totals.Net, "NIO"); got != -1200 {
		t.Fatalf("total neto NIO = %v, esperaba -1200", got)
	}
	if got := money(summary.Totals.Net, "USD"); got != 100 {
		t.Fatalf("total neto USD = %v, esperaba 100", got)
	}
	if len(summary.Totals.Net) != 2 {
		t.Fatalf("neto por moneda = %+v", summary.Totals.Net)
	}

	if summary.StateCounts["rented"] != 1 || summary.StateCounts["reserved"] != 1 ||
		summary.StateCounts["inhabited"] != 1 || summary.StateCounts["unused"] != 1 {
		t.Fatalf("conteo de estados = %+v", summary.StateCounts)
	}
}

func TestSummaryExcludesConvertedReservationAndClosedMonths(t *testing.T) {
	database := newTestDB(t)
	ctx := context.Background()

	homeID := mustExec(t, database, `INSERT INTO homes (name, usage) VALUES (?, ?)`, "Casa Demo", "rental")
	personID := mustExec(t, database, `INSERT INTO rental_people (full_name) VALUES (?)`, "Solicitante Demo")
	mustExec(t, database, `
		INSERT INTO rental_reservations (home_id, applicant_person_id, status, amount, currency, starts_at, expires_at)
		VALUES (?, ?, 'converted', 250, 'USD', '2026-11-05T10:00:00Z', '2026-12-05T10:00:00Z')`,
		homeID, personID)

	svc := NewService(NewStorage(database), fakeStates{
		{HomeID: homeID, Name: "Casa Demo", Usage: "rental", State: "listed"},
	})
	svc.now = func() time.Time { return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) }

	summary, err := svc.Summary(ctx, 2026, 11)
	if err != nil {
		t.Fatalf("resumen: %v", err)
	}
	if len(summary.Homes) != 1 || len(summary.Homes[0].Income) != 0 {
		t.Fatalf("la reserva convertida no debe contar como ingreso: %+v", summary.Homes)
	}
}

func TestSummaryPeriodValidation(t *testing.T) {
	svc := &Service{now: func() time.Time { return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) }}
	if _, err := svc.Summary(context.Background(), 1999, 11); err == nil {
		t.Fatal("se esperaba error con año fuera de rango")
	} else if err != ErrInvalidPeriod {
		t.Fatalf("error = %v, esperaba ErrInvalidPeriod", err)
	}
	if _, err := svc.Summary(context.Background(), 2026, 13); err == nil {
		t.Fatal("se esperaba error con mes inválido")
	}
}

func TestNewPeriodDefaultsToCurrentMonth(t *testing.T) {
	p, err := newPeriod(0, 0, time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("período: %v", err)
	}
	if p.Year != 2026 || p.Month != 11 || p.Start != "2026-11-01" || p.End != "2026-12-01" {
		t.Fatalf("período = %+v", p)
	}
}

func findMonth(t *testing.T, months []FinanceMonth, year, month int) FinanceMonth {
	t.Helper()
	for _, item := range months {
		if item.Year == year && item.Month == month {
			return item
		}
	}
	t.Fatalf("mes %04d-%02d no encontrado", year, month)
	return FinanceMonth{}
}

func itemTypes(items []FinanceItem) map[string]int {
	out := make(map[string]int)
	for _, item := range items {
		out[item.Type]++
	}
	return out
}

func TestHomeFinancesSeriesBreakdownAndCommission(t *testing.T) {
	database := newTestDB(t)
	ctx := context.Background()

	nioID := 0
	if err := database.QueryRow(`SELECT id FROM currencies WHERE code = 'NIO'`).Scan(&nioID); err != nil {
		t.Fatalf("moneda NIO: %v", err)
	}
	usdID := 0
	if err := database.QueryRow(`SELECT id FROM currencies WHERE code = 'USD'`).Scan(&usdID); err != nil {
		t.Fatalf("moneda USD: %v", err)
	}

	homeID := mustExec(t, database, `INSERT INTO homes (name, address, usage) VALUES (?, ?, ?)`,
		"Casa Finanzas Demo", "Calle Demo 200", "rental")

	serviceID := mustExec(t, database, `
		INSERT INTO services (home_id, name, currency_id, frequency, suggested_amount, icon_key)
		VALUES (?, ?, ?, 'monthly', 0, 'bolt')`, homeID, "Luz Demo", nioID)
	mustExec(t, database, `INSERT INTO bills (service_id, year, month, amount, status) VALUES (?, 2026, 11, 1200, 'pending')`, serviceID)

	maintenanceID := mustExec(t, database, `
		INSERT INTO home_maintenances (home_id, performed_at, kind, area, description)
		VALUES (?, '2026-11-10', 'corrective', 'plumbing', 'Reparación demo')`, homeID)
	mustExec(t, database, `
		INSERT INTO home_maintenance_items (maintenance_id, item_type, description, quantity, unit, amount, currency_id)
		VALUES (?, 'labor', 'Mano de obra demo', 1, 'unit', 300, ?)`, maintenanceID, usdID)

	personID := mustExec(t, database, `INSERT INTO rental_people (full_name, email) VALUES (?, ?)`,
		"Inquilino Demo", "demo@example.com")
	leaseID := mustExec(t, database, `
		INSERT INTO rental_leases (home_id, status, start_date, end_date, term_months, rent_amount,
			currency, payment_day, commission_type, commission_percent, commission_base)
		VALUES (?, 'active', '2026-01-15', '2027-01-14', 12, 500, 'USD', 5, 'percent', 10, 'collected')`, homeID)
	mustExec(t, database, `
		INSERT INTO rental_payments (lease_id, paid_at, amount, currency, method)
		VALUES (?, '2026-11-06', 300, 'USD', 'transfer')`, leaseID)
	mustExec(t, database, `
		INSERT INTO rental_reservations (home_id, applicant_person_id, status, amount, currency, starts_at, expires_at)
		VALUES (?, ?, 'active', 100, 'USD', '2026-11-02T10:00:00Z', '2026-12-02T10:00:00Z')`,
		homeID, personID)

	svc := NewService(NewStorage(database), fakeStates{
		{HomeID: homeID, Name: "Casa Finanzas Demo", Address: "Calle Demo 200", Usage: "rental", State: "rented"},
	})
	svc.now = func() time.Time { return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) }

	finances, err := svc.HomeFinances(ctx, homeID, 2026, 11)
	if err != nil {
		t.Fatalf("finanzas: %v", err)
	}
	if len(finances.Months) != 12 {
		t.Fatalf("meses = %d, esperaba 12", len(finances.Months))
	}
	if first := finances.Months[0]; first.Year != 2025 || first.Month != 12 {
		t.Fatalf("primer mes = %+v, esperaba 2025-12", first)
	}
	if last := finances.Months[11]; last.Year != 2026 || last.Month != 11 {
		t.Fatalf("último mes = %+v, esperaba 2026-11", last)
	}

	nov := findMonth(t, finances.Months, 2026, 11)
	if got := money(nov.Income, "USD"); got != 400 {
		t.Fatalf("ingreso USD noviembre = %v, esperaba 400", got)
	}
	if got := money(nov.Expenses, "NIO"); got != 1200 {
		t.Fatalf("gasto NIO noviembre = %v, esperaba 1200", got)
	}
	if got := money(nov.Expenses, "USD"); got != 300 {
		t.Fatalf("gasto USD noviembre = %v, esperaba 300", got)
	}
	if got := money(nov.Net, "USD"); got != 100 {
		t.Fatalf("neto USD noviembre = %v, esperaba 100", got)
	}

	jan := findMonth(t, finances.Months, 2026, 1)
	if got := money(jan.Expenses, "USD"); got != 50 {
		t.Fatalf("comisión enero = %v, esperaba 50", got)
	}
	if got := money(nov.Expenses, "USD"); got != 300 {
		t.Fatalf("la comisión no debe repetirse en noviembre, gasto USD = %v", got)
	}

	if len(finances.Breakdown) != 2 {
		t.Fatalf("breakdown noviembre = %+v, esperaba servicios y reparaciones", finances.Breakdown)
	}
	kinds := make(map[string]bool)
	for _, item := range finances.Breakdown {
		kinds[item.Kind] = true
	}
	if !kinds["services"] || !kinds["maintenance"] || kinds["commission"] {
		t.Fatalf("breakdown inesperado: %+v", finances.Breakdown)
	}

	types := itemTypes(finances.Items)
	for _, want := range []string{"payment", "reservation", "bill", "maintenance"} {
		if types[want] != 1 {
			t.Fatalf("items de noviembre = %+v, falta %s", types, want)
		}
	}
	if types["commission"] != 0 {
		t.Fatalf("la comisión no corresponde a noviembre: %+v", types)
	}
	if got := money(finances.PendingExpenses, "NIO"); got != 1200 {
		t.Fatalf("por pagar NIO = %v, esperaba 1200", got)
	}
	if len(finances.Currencies) != 2 {
		t.Fatalf("monedas = %+v, esperaba NIO y USD", finances.Currencies)
	}

	// Enero: la comisión aparece una sola vez, como costo y como movimiento.
	janFinances, err := svc.HomeFinances(ctx, homeID, 2026, 1)
	if err != nil {
		t.Fatalf("finanzas enero: %v", err)
	}
	if itemTypes(janFinances.Items)["commission"] != 1 {
		t.Fatalf("items de enero = %+v, esperaba comisión", itemTypes(janFinances.Items))
	}
	if len(janFinances.Breakdown) != 1 || janFinances.Breakdown[0].Kind != "commission" {
		t.Fatalf("breakdown enero = %+v, esperaba solo comisión", janFinances.Breakdown)
	}
}

func TestHomeFinancesHomeNotFound(t *testing.T) {
	database := newTestDB(t)
	svc := NewService(NewStorage(database), fakeStates{})
	svc.now = func() time.Time { return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) }
	if _, err := svc.HomeFinances(context.Background(), 999, 2026, 11); err != ErrHomeNotFound {
		t.Fatalf("error = %v, esperaba ErrHomeNotFound", err)
	}
}
