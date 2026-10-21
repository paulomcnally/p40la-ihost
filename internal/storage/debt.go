package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// DebtStorage encapsula el acceso a la tabla debts.
type DebtStorage struct {
	db *sql.DB
}

// NewDebtStorage crea un nuevo DebtStorage.
func NewDebtStorage(db *sql.DB) *DebtStorage {
	return &DebtStorage{db: db}
}

const debtColumns = `
	d.id, d.institution_id, COALESCE(i.name, ''), d.identifier, d.description,
	d.total, d.principal, d.currency_id, COALESCE(c.code, ''),
	d.installments_total, d.installment_amount, d.interest_rate, d.payment_day,
	d.start_date, d.status,
	COALESCE(s.pending_count, 0), COALESCE(s.pending_amount, 0),
	COALESCE(s.paid_count, 0), COALESCE(s.paid_amount, 0),
	COALESCE(s.next_due_date, ''), COALESCE(s.last_due_date, ''), COALESCE(s.last_paid_at, ''),
	d.deleted_at, d.created_at, d.updated_at
`

// debtJoins agrega el resumen de cuotas por deuda en una sola consulta
// (SPEC-104): pendientes, pagadas, próxima y última cuota pendiente, último pago.
const debtJoins = `
	FROM debts d
	LEFT JOIN institutions i ON i.id = d.institution_id
	LEFT JOIN currencies c ON c.id = d.currency_id
	LEFT JOIN (
		SELECT debt_id,
		       SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) AS pending_count,
		       SUM(CASE WHEN status = 'pending' THEN amount ELSE 0 END) AS pending_amount,
		       SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END) AS paid_count,
		       SUM(CASE WHEN status = 'paid' THEN amount ELSE 0 END) AS paid_amount,
		       MIN(CASE WHEN status = 'pending' THEN due_date END) AS next_due_date,
		       MAX(CASE WHEN status = 'pending' THEN due_date END) AS last_due_date,
		       MAX(CASE WHEN status = 'paid' THEN paid_at END) AS last_paid_at
		FROM debt_bills
		WHERE deleted_at IS NULL
		GROUP BY debt_id
	) s ON s.debt_id = d.id
`

// List devuelve todas las deudas no eliminadas, con acreedor, moneda y resumen de cuotas.
func (s *DebtStorage) List(ctx context.Context) ([]models.Debt, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+debtColumns+debtJoins+`
		WHERE d.deleted_at IS NULL
		ORDER BY d.description
	`)
	if err != nil {
		return nil, fmt.Errorf("listar deudas: %w", err)
	}
	defer rows.Close()

	return scanDebts(rows)
}

// GetByID busca una deuda por su ID, con su resumen de cuotas.
func (s *DebtStorage) GetByID(ctx context.Context, id int64) (*models.Debt, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+debtColumns+debtJoins+`
		WHERE d.id = ? AND d.deleted_at IS NULL
	`, id)
	return scanDebt(row)
}

// Count devuelve la cantidad de deudas no eliminadas (prerrequisitos).
func (s *DebtStorage) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM debts WHERE deleted_at IS NULL").Scan(&count); err != nil {
		return 0, fmt.Errorf("contar deudas: %w", err)
	}
	return count, nil
}

// Create inserta una nueva deuda.
func (s *DebtStorage) Create(ctx context.Context, debt *models.Debt) (*models.Debt, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO debts (institution_id, identifier, description, total, principal,
		                   currency_id, installments_total, installment_amount,
		                   interest_rate, payment_day, start_date, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, debt.InstitutionID, debt.Identifier, debt.Description, debt.Total, debt.Principal,
		debt.CurrencyID, debt.InstallmentsTotal, debt.InstallmentAmount,
		debt.InterestRate, debt.PaymentDay, debt.StartDate, debt.Status)
	if err != nil {
		return nil, fmt.Errorf("insertar deuda: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de deuda: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza una deuda existente.
func (s *DebtStorage) Update(ctx context.Context, debt *models.Debt) (*models.Debt, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE debts
		SET institution_id = ?, identifier = ?, description = ?, total = ?, principal = ?,
		    currency_id = ?, installments_total = ?, installment_amount = ?,
		    interest_rate = ?, payment_day = ?, start_date = ?, status = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, debt.InstitutionID, debt.Identifier, debt.Description, debt.Total, debt.Principal,
		debt.CurrencyID, debt.InstallmentsTotal, debt.InstallmentAmount,
		debt.InterestRate, debt.PaymentDay, debt.StartDate, debt.Status, debt.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar deuda: %w", err)
	}
	return s.GetByID(ctx, debt.ID)
}

// SoftDelete marca una deuda como eliminada.
func (s *DebtStorage) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE debts SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar deuda: %w", err)
	}
	return nil
}

// DebtHistoryRow es una deuda con los datos necesarios para reconstruir la
// serie histórica mensual (SPEC-105). Incluye deudas eliminadas lógicamente.
type DebtHistoryRow struct {
	ID                int64
	CurrencyID        int64
	CurrencyCode      string
	CurrencySymbol    string
	Total             float64
	StartDate         string
	InstallmentsTotal int
	Status            string
	UpdatedAt         time.Time
	DeletedAt         sql.NullTime
	LastPaidAt        string
	PendingCount      int
	PaidCount         int
	PendingAmount     float64
}

// ListForHistory devuelve todas las deudas, incluidas las eliminadas
// lógicamente, con el último pago y el resumen de cuotas. La historia no debe
// perder las deudas borradas: se usa deleted_at como fecha de cierre.
func (s *DebtStorage) ListForHistory(ctx context.Context) ([]DebtHistoryRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.currency_id, COALESCE(c.code, ''), COALESCE(c.symbol, ''),
		       d.total, d.start_date, d.installments_total, d.status, d.updated_at, d.deleted_at,
		       COALESCE(s.last_paid_at, ''), COALESCE(s.pending_count, 0),
		       COALESCE(s.paid_count, 0), COALESCE(s.pending_amount, 0)
		FROM debts d
		LEFT JOIN currencies c ON c.id = d.currency_id
		LEFT JOIN (
			SELECT debt_id,
			       MAX(CASE WHEN status = 'paid' THEN paid_at END) AS last_paid_at,
			       SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) AS pending_count,
			       SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END) AS paid_count,
			       SUM(CASE WHEN status = 'pending' THEN amount ELSE 0 END) AS pending_amount
			FROM debt_bills
			WHERE deleted_at IS NULL
			GROUP BY debt_id
		) s ON s.debt_id = d.id
		ORDER BY d.id
	`)
	if err != nil {
		return nil, fmt.Errorf("listar deudas para historia: %w", err)
	}
	defer rows.Close()

	var result []DebtHistoryRow
	for rows.Next() {
		var row DebtHistoryRow
		if err := rows.Scan(&row.ID, &row.CurrencyID, &row.CurrencyCode, &row.CurrencySymbol,
			&row.Total, &row.StartDate, &row.InstallmentsTotal, &row.Status,
			&row.UpdatedAt, &row.DeletedAt, &row.LastPaidAt, &row.PendingCount,
			&row.PaidCount, &row.PendingAmount); err != nil {
			return nil, fmt.Errorf("escanear deuda para historia: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanDebt(row *sql.Row) (*models.Debt, error) {
	var d models.Debt
	var deletedAt sql.NullTime
	var institutionName, currencyCode, nextDue, lastDue, lastPaid sql.NullString
	if err := row.Scan(&d.ID, &d.InstitutionID, &institutionName, &d.Identifier, &d.Description,
		&d.Total, &d.Principal, &d.CurrencyID, &currencyCode,
		&d.InstallmentsTotal, &d.InstallmentAmount, &d.InterestRate, &d.PaymentDay,
		&d.StartDate, &d.Status,
		&d.PendingCount, &d.PendingAmount, &d.PaidCount, &d.PaidAmount,
		&nextDue, &lastDue, &lastPaid,
		&deletedAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear deuda: %w", err)
	}
	d.InstitutionName = institutionName.String
	d.CurrencyCode = currencyCode.String
	d.NextDueDate = nextDue.String
	d.LastDueDate = lastDue.String
	d.LastPaidAt = lastPaid.String
	if deletedAt.Valid {
		d.DeletedAt = &deletedAt.Time
	}
	return &d, nil
}

func scanDebts(rows *sql.Rows) ([]models.Debt, error) {
	var debts []models.Debt
	for rows.Next() {
		var d models.Debt
		var deletedAt sql.NullTime
		var institutionName, currencyCode, nextDue, lastDue, lastPaid sql.NullString
		if err := rows.Scan(&d.ID, &d.InstitutionID, &institutionName, &d.Identifier, &d.Description,
			&d.Total, &d.Principal, &d.CurrencyID, &currencyCode,
			&d.InstallmentsTotal, &d.InstallmentAmount, &d.InterestRate, &d.PaymentDay,
			&d.StartDate, &d.Status,
			&d.PendingCount, &d.PendingAmount, &d.PaidCount, &d.PaidAmount,
			&nextDue, &lastDue, &lastPaid,
			&deletedAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear deuda: %w", err)
		}
		d.InstitutionName = institutionName.String
		d.CurrencyCode = currencyCode.String
		d.NextDueDate = nextDue.String
		d.LastDueDate = lastDue.String
		d.LastPaidAt = lastPaid.String
		if deletedAt.Valid {
			d.DeletedAt = &deletedAt.Time
		}
		debts = append(debts, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return debts, nil
}
