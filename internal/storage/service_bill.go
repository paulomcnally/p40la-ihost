package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// periodExpr calcula el período efectivo (YYYY-MM) de una factura (SPEC-120):
// 1) due_date si existe; 2) year/month si month está entre 1 y 12;
// 3) enero del año para facturas anuales (month = 0) sin due_date, que es
// cuando el scheduler las genera.
const periodExpr = `CASE
	WHEN b.due_date IS NOT NULL AND length(b.due_date) >= 7 THEN substr(b.due_date, 1, 7)
	WHEN b.month BETWEEN 1 AND 12 THEN printf('%04d-%02d', b.year, b.month)
	ELSE printf('%04d-01', b.year)
END`

// ListServiceBillsByPeriod devuelve las facturas cuyo período efectivo cae en
// el año/mes indicado, con contexto de servicio, casa y moneda (SPEC-120).
func (s *BillStorage) ListServiceBillsByPeriod(ctx context.Context, year, month int, homeID *int64) ([]models.ServiceBillDetail, error) {
	query := `
		SELECT b.id, b.service_id, s.name, COALESCE(i.name, s.institution, ''),
		       s.home_id, h.name, COALESCE(s.icon_key, ''), s.currency_id,
		       COALESCE(c.code, ''), COALESCE(c.symbol, ''),
		       s.frequency, s.billing_day, s.active,
		       b.year, b.month, b.amount, COALESCE(b.invoice_number, ''), b.status,
		       COALESCE(b.drive_url, ''), b.issue_date, b.due_date, b.paid_at,
		       COALESCE(b.payment_reference, '')
		FROM bills b
		JOIN services s ON s.id = b.service_id AND s.deleted_at IS NULL
		JOIN homes h ON h.id = s.home_id
		LEFT JOIN institutions i ON i.id = s.institution_id
		LEFT JOIN currencies c ON c.id = s.currency_id
		WHERE b.deleted_at IS NULL
		  AND (
		    (b.due_date IS NOT NULL AND substr(b.due_date, 1, 7) = ?)
		    OR (b.due_date IS NULL AND b.year = ? AND b.month = ?)
		    OR (b.due_date IS NULL AND b.year = ? AND b.month = 0 AND ? = 1)
		  )`
	args := []any{
		fmt.Sprintf("%04d-%02d", year, month),
		year, month,
		year, month,
	}
	if homeID != nil {
		query += " AND s.home_id = ?"
		args = append(args, *homeID)
	}
	query += " ORDER BY s.name, b.id"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listar facturas de servicios del período: %w", err)
	}
	defer rows.Close()

	var out []models.ServiceBillDetail
	for rows.Next() {
		var d models.ServiceBillDetail
		var billingDay sql.NullInt64
		var issueDate, dueDate sql.NullString
		var paidAt sql.NullTime
		if err := rows.Scan(
			&d.ID, &d.ServiceID, &d.ServiceName, &d.Institution,
			&d.HomeID, &d.HomeName, &d.IconKey, &d.CurrencyID,
			&d.CurrencyCode, &d.CurrencySymbol,
			&d.Frequency, &billingDay, &d.Active,
			&d.Year, &d.Month, &d.Amount, &d.InvoiceNumber, &d.Status,
			&d.DriveURL, &issueDate, &dueDate, &paidAt, &d.PaymentReference,
		); err != nil {
			return nil, fmt.Errorf("escanear factura de servicio: %w", err)
		}
		if billingDay.Valid {
			v := int(billingDay.Int64)
			d.BillingDay = &v
		}
		if issueDate.Valid {
			d.IssueDate = &issueDate.String
		}
		if dueDate.Valid {
			d.DueDate = &dueDate.String
		}
		if paidAt.Valid {
			d.PaidAt = &paidAt.Time
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar facturas de servicios del período: %w", err)
	}
	return out, nil
}

// ServiceBillsSummary agrupa las facturas por período efectivo y moneda en un
// rango YYYY-MM (inclusive) para la tab Gráfica (SPEC-120).
func (s *BillStorage) ServiceBillsSummary(ctx context.Context, from, to string, homeID *int64) ([]models.ServiceBillSummaryRow, error) {
	query := `
		SELECT period, year, month, currency_code, currency_symbol,
		       SUM(amount) AS total,
		       SUM(CASE WHEN status = 'paid' THEN amount ELSE 0 END) AS paid,
		       SUM(CASE WHEN status = 'pending' THEN amount ELSE 0 END) AS pending,
		       COUNT(*) AS count
		FROM (
			SELECT ` + periodExpr + ` AS period,
			       CASE
			         WHEN b.due_date IS NOT NULL AND length(b.due_date) >= 7 THEN CAST(substr(b.due_date, 1, 4) AS INTEGER)
			         ELSE b.year
			       END AS year,
			       CASE
			         WHEN b.due_date IS NOT NULL AND length(b.due_date) >= 7 THEN CAST(substr(b.due_date, 6, 2) AS INTEGER)
			         WHEN b.month BETWEEN 1 AND 12 THEN b.month
			         ELSE 1
			       END AS month,
			       COALESCE(c.code, '') AS currency_code,
			       COALESCE(c.symbol, '') AS currency_symbol,
			       b.amount, b.status
			FROM bills b
			JOIN services s ON s.id = b.service_id AND s.deleted_at IS NULL
			LEFT JOIN currencies c ON c.id = s.currency_id
			WHERE b.deleted_at IS NULL`
	args := []any{}
	if homeID != nil {
		query += " AND s.home_id = ?"
		args = append(args, *homeID)
	}
	query += `
		) t
		WHERE period >= ? AND period <= ?
		GROUP BY period, year, month, currency_code, currency_symbol
		ORDER BY period, currency_code`
	args = append(args, from, to)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("resumir facturas de servicios: %w", err)
	}
	defer rows.Close()

	var out []models.ServiceBillSummaryRow
	for rows.Next() {
		var r models.ServiceBillSummaryRow
		if err := rows.Scan(&r.Period, &r.Year, &r.Month, &r.CurrencyCode, &r.CurrencySymbol,
			&r.Total, &r.Paid, &r.Pending, &r.Count); err != nil {
			return nil, fmt.Errorf("escanear resumen de facturas: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar resumen de facturas: %w", err)
	}
	return out, nil
}

// ListServiceBillsForProjection devuelve las facturas (pagadas y pendientes)
// cuyo período efectivo cae en el rango, para proyectar sin doble conteo
// (SPEC-120).
func (s *BillStorage) ListServiceBillsForProjection(ctx context.Context, from, to string, homeID *int64) ([]models.ProjectionBillRow, error) {
	query := `
		SELECT bill_id, service_id, service_name, year, month, due_date, amount, status, currency_code, currency_symbol
		FROM (
			SELECT b.id AS bill_id, b.service_id, s.name AS service_name,
			       b.year, b.month, b.due_date, b.amount, b.status,
			       COALESCE(c.code, '') AS currency_code,
			       COALESCE(c.symbol, '') AS currency_symbol,
			       ` + periodExpr + ` AS period
			FROM bills b
			JOIN services s ON s.id = b.service_id AND s.deleted_at IS NULL
			LEFT JOIN currencies c ON c.id = s.currency_id
			WHERE b.deleted_at IS NULL`
	args := []any{}
	if homeID != nil {
		query += " AND s.home_id = ?"
		args = append(args, *homeID)
	}
	query += `
		) t
		WHERE period >= ? AND period <= ?
		ORDER BY period, bill_id`
	args = append(args, from, to)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listar facturas para proyección: %w", err)
	}
	defer rows.Close()

	var out []models.ProjectionBillRow
	for rows.Next() {
		var r models.ProjectionBillRow
		var dueDate sql.NullString
		if err := rows.Scan(&r.BillID, &r.ServiceID, &r.ServiceName, &r.Year, &r.Month, &dueDate,
			&r.Amount, &r.Status, &r.CurrencyCode, &r.CurrencySymbol); err != nil {
			return nil, fmt.Errorf("escanear factura para proyección: %w", err)
		}
		if dueDate.Valid {
			r.DueDate = &dueDate.String
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar facturas para proyección: %w", err)
	}
	return out, nil
}

// ListRecurringServicesForProjection devuelve los servicios activos y
// recurrentes con su moneda para estimar facturas futuras (SPEC-120).
func (s *BillStorage) ListRecurringServicesForProjection(ctx context.Context, homeID *int64) ([]models.ProjectionServiceRow, error) {
	query := `
		SELECT s.id, s.name, s.home_id, s.currency_id,
		       COALESCE(c.code, ''), COALESCE(c.symbol, ''),
		       s.frequency, s.suggested_amount, s.start_date, s.end_date
		FROM services s
		LEFT JOIN currencies c ON c.id = s.currency_id
		WHERE s.deleted_at IS NULL AND s.active = 1 AND s.is_recurring = 1`
	args := []any{}
	if homeID != nil {
		query += " AND s.home_id = ?"
		args = append(args, *homeID)
	}
	query += " ORDER BY s.name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listar servicios para proyección: %w", err)
	}
	defer rows.Close()

	var out []models.ProjectionServiceRow
	for rows.Next() {
		var r models.ProjectionServiceRow
		var startDate, endDate sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &r.HomeID, &r.CurrencyID,
			&r.CurrencyCode, &r.CurrencySymbol,
			&r.Frequency, &r.SuggestedAmount, &startDate, &endDate); err != nil {
			return nil, fmt.Errorf("escanear servicio para proyección: %w", err)
		}
		if startDate.Valid {
			r.StartDate = &startDate.String
		}
		if endDate.Valid {
			r.EndDate = &endDate.String
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar servicios para proyección: %w", err)
	}
	return out, nil
}
