package portfolio

import (
	"context"
	"database/sql"
	"fmt"
)

// Storage es el único punto con SQL del módulo (ARCH-MODULES).
type Storage struct {
	db *sql.DB
}

// NewStorage crea el storage del módulo.
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

// moneyByHome acumula importes por vivienda y moneda.
type moneyByHome map[int64]map[string]float64

func accumulate(dest moneyByHome, homeID int64, currency string, amount float64) {
	if dest[homeID] == nil {
		dest[homeID] = make(map[string]float64)
	}
	dest[homeID][currency] += amount
}

// ExpensesByHome devuelve los gastos del mes y las facturas pendientes de pago,
// por vivienda y moneda: facturas de servicios + reparaciones (SPEC-143).
func (s *Storage) ExpensesByHome(ctx context.Context, p period) (moneyByHome, moneyByHome, error) {
	expenses := make(moneyByHome)
	pending := make(moneyByHome)

	rows, err := s.db.QueryContext(ctx, `
		SELECT s.home_id, c.code,
			COALESCE(SUM(b.amount), 0),
			COALESCE(SUM(CASE WHEN b.status = 'pending' THEN b.amount ELSE 0 END), 0)
		FROM bills b
		JOIN services s ON s.id = b.service_id
		JOIN currencies c ON c.id = s.currency_id
		WHERE b.deleted_at IS NULL
			AND s.deleted_at IS NULL
			AND b.year = ? AND b.month = ?
		GROUP BY s.home_id, c.code
	`, p.Year, p.Month)
	if err != nil {
		return nil, nil, fmt.Errorf("resumir facturas de servicios: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var homeID int64
		var currency string
		var total, pend float64
		if err := rows.Scan(&homeID, &currency, &total, &pend); err != nil {
			return nil, nil, fmt.Errorf("escanear facturas: %w", err)
		}
		accumulate(expenses, homeID, currency, total)
		accumulate(pending, homeID, currency, pend)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	rows, err = s.db.QueryContext(ctx, `
		SELECT m.home_id, c.code, COALESCE(SUM(i.amount), 0)
		FROM home_maintenance_items i
		JOIN home_maintenances m ON m.id = i.maintenance_id
		JOIN currencies c ON c.id = i.currency_id
		WHERE m.performed_at >= ? AND m.performed_at < ?
		GROUP BY m.home_id, c.code
	`, p.Start, p.End)
	if err != nil {
		return nil, nil, fmt.Errorf("resumir reparaciones: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var homeID int64
		var currency string
		var total float64
		if err := rows.Scan(&homeID, &currency, &total); err != nil {
			return nil, nil, fmt.Errorf("escanear reparaciones: %w", err)
		}
		accumulate(expenses, homeID, currency, total)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return expenses, pending, nil
}

// IncomeByHome devuelve cobros y señas del mes, por vivienda y moneda.
func (s *Storage) IncomeByHome(ctx context.Context, p period) (moneyByHome, error) {
	income := make(moneyByHome)

	rows, err := s.db.QueryContext(ctx, `
		SELECT l.home_id, p.currency, COALESCE(SUM(p.amount), 0)
		FROM rental_payments p
		JOIN rental_leases l ON l.id = p.lease_id
		WHERE l.deleted_at IS NULL
			AND p.paid_at >= ? AND p.paid_at < ?
		GROUP BY l.home_id, p.currency
	`, p.Start, p.End)
	if err != nil {
		return nil, fmt.Errorf("resumir cobros: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var homeID int64
		var currency string
		var total float64
		if err := rows.Scan(&homeID, &currency, &total); err != nil {
			return nil, fmt.Errorf("escanear cobros: %w", err)
		}
		accumulate(income, homeID, currency, total)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Las reservas convertidas ya cuentan como depósito del contrato: se
	// excluyen para no duplicar el ingreso.
	rows, err = s.db.QueryContext(ctx, `
		SELECT r.home_id, r.currency, COALESCE(SUM(r.amount), 0)
		FROM rental_reservations r
		WHERE r.status IN ('active', 'forfeited')
			AND r.starts_at >= ? AND r.starts_at < ?
		GROUP BY r.home_id, r.currency
	`, p.Start, p.End)
	if err != nil {
		return nil, fmt.Errorf("resumir señas: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var homeID int64
		var currency string
		var total float64
		if err := rows.Scan(&homeID, &currency, &total); err != nil {
			return nil, fmt.Errorf("escanear señas: %w", err)
		}
		accumulate(income, homeID, currency, total)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return income, nil
}

// PendingIncomeByHome devuelve el saldo por cobrar al cierre del mes: cargos
// vencidos o del pasado no anulados menos pagos registrados hasta el cierre.
func (s *Storage) PendingIncomeByHome(ctx context.Context, p period) (moneyByHome, error) {
	charges := make(moneyByHome)

	rows, err := s.db.QueryContext(ctx, `
		SELECT l.home_id, c.currency, COALESCE(SUM(c.amount), 0)
		FROM rental_charges c
		JOIN rental_leases l ON l.id = c.lease_id
		WHERE l.deleted_at IS NULL
			AND c.status <> 'void'
			AND c.due_date < ?
		GROUP BY l.home_id, c.currency
	`, p.End)
	if err != nil {
		return nil, fmt.Errorf("resumir cargos pendientes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var homeID int64
		var currency string
		var total float64
		if err := rows.Scan(&homeID, &currency, &total); err != nil {
			return nil, fmt.Errorf("escanear cargos pendientes: %w", err)
		}
		accumulate(charges, homeID, currency, total)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.db.QueryContext(ctx, `
		SELECT l.home_id, p.currency, COALESCE(SUM(p.amount), 0)
		FROM rental_payments p
		JOIN rental_leases l ON l.id = p.lease_id
		WHERE l.deleted_at IS NULL
			AND p.paid_at < ?
		GROUP BY l.home_id, p.currency
	`, p.End)
	if err != nil {
		return nil, fmt.Errorf("resumir pagos acumulados: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var homeID int64
		var currency string
		var total float64
		if err := rows.Scan(&homeID, &currency, &total); err != nil {
			return nil, fmt.Errorf("escanear pagos acumulados: %w", err)
		}
		accumulate(charges, homeID, currency, -total)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// El saldo no puede ser negativo en el resumen.
	for homeID, byCurrency := range charges {
		for currency, amount := range byCurrency {
			if amount <= 0 {
				delete(byCurrency, currency)
			}
		}
		if len(byCurrency) == 0 {
			delete(charges, homeID)
		}
	}
	return charges, nil
}

// ---------------------------------------------------------------- finanzas casa

// PaymentRow es un cobro de la ventana de 12 meses.
type PaymentRow struct {
	ID        int64
	LeaseID   int64
	PaidAt    string
	Amount    float64
	Currency  string
	Method    string
	Reference string
}

// ReservationRow es una seña de la ventana.
type ReservationRow struct {
	ID        int64
	StartsAt  string
	Amount    float64
	Currency  string
	Reference string
	Status    string
}

// BillRow es una factura de servicio de la ventana.
type BillRow struct {
	ID          int64
	ServiceID   int64
	Year        int
	Month       int
	Amount      float64
	Status      string
	ServiceName string
	Institution string
	Currency    string
}

// MaintenanceRow es un total de reparación por moneda.
type MaintenanceRow struct {
	ID          int64
	PerformedAt string
	Kind        string
	Area        string
	Description string
	Currency    string
	Amount      float64
}

// CommissionRow es la comisión de un contrato iniciado en la ventana.
type CommissionRow struct {
	LeaseID     int64
	StartDate   string
	Currency    string
	Type        string
	Percent     float64
	FixedAmount float64
	RentAmount  float64
	AgencyName  string
}

// FinanceWindow agrupa las fuentes del tab Finanzas en una sola lectura.
type FinanceWindow struct {
	Payments     []PaymentRow
	Reservations []ReservationRow
	Bills        []BillRow
	Maintenances []MaintenanceRow
	Commissions  []CommissionRow
}

// HomeFinanceWindow devuelve los movimientos de una casa entre start y end
// (fechas ISO, fin exclusivo). Las facturas se traen por rango de años porque
// su fecha es year/month.
func (s *Storage) HomeFinanceWindow(ctx context.Context, homeID int64, start, end string, startYear, endYear int) (*FinanceWindow, error) {
	window := &FinanceWindow{}

	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.lease_id, p.paid_at, p.amount, p.currency, p.method, p.reference
		FROM rental_payments p
		JOIN rental_leases l ON l.id = p.lease_id
		WHERE l.home_id = ? AND l.deleted_at IS NULL
			AND p.paid_at >= ? AND p.paid_at < ?
		ORDER BY p.paid_at DESC, p.id DESC
	`, homeID, start, end)
	if err != nil {
		return nil, fmt.Errorf("listar cobros de la vivienda: %w", err)
	}
	for rows.Next() {
		var row PaymentRow
		if err := rows.Scan(&row.ID, &row.LeaseID, &row.PaidAt, &row.Amount, &row.Currency,
			&row.Method, &row.Reference); err != nil {
			rows.Close()
			return nil, fmt.Errorf("escanear cobro: %w", err)
		}
		window.Payments = append(window.Payments, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `
		SELECT id, starts_at, amount, currency, reference, status
		FROM rental_reservations
		WHERE home_id = ? AND status IN ('active', 'forfeited')
			AND starts_at >= ? AND starts_at < ?
		ORDER BY starts_at DESC, id DESC
	`, homeID, start, end)
	if err != nil {
		return nil, fmt.Errorf("listar señas de la vivienda: %w", err)
	}
	for rows.Next() {
		var row ReservationRow
		if err := rows.Scan(&row.ID, &row.StartsAt, &row.Amount, &row.Currency,
			&row.Reference, &row.Status); err != nil {
			rows.Close()
			return nil, fmt.Errorf("escanear seña: %w", err)
		}
		window.Reservations = append(window.Reservations, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `
		SELECT b.id, b.service_id, b.year, b.month, b.amount, b.status,
			s.name, COALESCE(i.name, s.institution, ''), c.code
		FROM bills b
		JOIN services s ON s.id = b.service_id
		LEFT JOIN institutions i ON i.id = s.institution_id
		JOIN currencies c ON c.id = s.currency_id
		WHERE s.home_id = ? AND b.deleted_at IS NULL AND s.deleted_at IS NULL
			AND b.year >= ? AND b.year <= ?
		ORDER BY b.year DESC, b.month DESC, b.id DESC
	`, homeID, startYear, endYear)
	if err != nil {
		return nil, fmt.Errorf("listar facturas de la vivienda: %w", err)
	}
	for rows.Next() {
		var row BillRow
		if err := rows.Scan(&row.ID, &row.ServiceID, &row.Year, &row.Month, &row.Amount,
			&row.Status, &row.ServiceName, &row.Institution, &row.Currency); err != nil {
			rows.Close()
			return nil, fmt.Errorf("escanear factura: %w", err)
		}
		window.Bills = append(window.Bills, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `
		SELECT m.id, m.performed_at, m.kind, m.area, m.description, c.code, SUM(i.amount)
		FROM home_maintenances m
		JOIN home_maintenance_items i ON i.maintenance_id = m.id
		JOIN currencies c ON c.id = i.currency_id
		WHERE m.home_id = ? AND m.performed_at >= ? AND m.performed_at < ?
		GROUP BY m.id, c.code
		ORDER BY m.performed_at DESC, m.id DESC
	`, homeID, start, end)
	if err != nil {
		return nil, fmt.Errorf("listar reparaciones de la vivienda: %w", err)
	}
	for rows.Next() {
		var row MaintenanceRow
		if err := rows.Scan(&row.ID, &row.PerformedAt, &row.Kind, &row.Area,
			&row.Description, &row.Currency, &row.Amount); err != nil {
			rows.Close()
			return nil, fmt.Errorf("escanear reparación: %w", err)
		}
		window.Maintenances = append(window.Maintenances, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `
		SELECT l.id, l.start_date, l.currency, l.commission_type, l.commission_percent,
			l.commission_amount, l.rent_amount, COALESCE(i.name, '')
		FROM rental_leases l
		LEFT JOIN institutions i ON i.id = l.agency_institution_id
		WHERE l.home_id = ? AND l.deleted_at IS NULL
			AND l.status IN ('active', 'ended', 'terminated')
			AND l.start_date >= ? AND l.start_date < ?
		ORDER BY l.start_date DESC, l.id DESC
	`, homeID, start, end)
	if err != nil {
		return nil, fmt.Errorf("listar comisiones de la vivienda: %w", err)
	}
	for rows.Next() {
		var row CommissionRow
		if err := rows.Scan(&row.LeaseID, &row.StartDate, &row.Currency, &row.Type,
			&row.Percent, &row.FixedAmount, &row.RentAmount, &row.AgencyName); err != nil {
			rows.Close()
			return nil, fmt.Errorf("escanear comisión: %w", err)
		}
		window.Commissions = append(window.Commissions, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	return window, nil
}
