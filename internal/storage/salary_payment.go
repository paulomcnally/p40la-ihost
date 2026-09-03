package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// SalaryPaymentStorage encapsula el acceso a la tabla salary_payments.
type SalaryPaymentStorage struct {
	db *sql.DB
}

// NewSalaryPaymentStorage crea un nuevo SalaryPaymentStorage.
func NewSalaryPaymentStorage(db *sql.DB) *SalaryPaymentStorage {
	return &SalaryPaymentStorage{db: db}
}

const salaryPaymentSelectCols = `
		p.id, p.salary_id, s.employer, p.year, p.month, p.amount, p.currency, p.status,
		p.received_amount, p.received_at, p.notes, p.created_at, p.updated_at`

// ListByFilters devuelve los pagos de salario de un período, opcionalmente por salario.
func (s *SalaryPaymentStorage) ListByFilters(ctx context.Context, year, month int, salaryID int64) ([]models.SalaryPayment, error) {
	query := `
		SELECT` + salaryPaymentSelectCols + `
		FROM salary_payments p
		JOIN salaries s ON s.id = p.salary_id
		WHERE p.year = ? AND p.month = ?`
	args := []any{year, month}

	if salaryID > 0 {
		query += ` AND p.salary_id = ?`
		args = append(args, salaryID)
	}
	query += ` ORDER BY p.created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listar pagos de salario: %w", err)
	}
	defer rows.Close()

	var payments []models.SalaryPayment
	for rows.Next() {
		var p models.SalaryPayment
		var receivedAmount sql.NullFloat64
		var receivedAt sql.NullTime
		var notes sql.NullString
		if err := rows.Scan(&p.ID, &p.SalaryID, &p.Employer, &p.Year, &p.Month, &p.Amount, &p.Currency, &p.Status,
			&receivedAmount, &receivedAt, &notes, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear pago de salario: %w", err)
		}
		if receivedAmount.Valid {
			p.ReceivedAmount = &receivedAmount.Float64
		}
		if receivedAt.Valid {
			p.ReceivedAt = &receivedAt.Time
		}
		if notes.Valid {
			p.Notes = &notes.String
		}
		payments = append(payments, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return payments, nil
}

// GetByID busca un pago de salario por ID.
func (s *SalaryPaymentStorage) GetByID(ctx context.Context, id int64) (*models.SalaryPayment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT`+salaryPaymentSelectCols+`
		FROM salary_payments p
		JOIN salaries s ON s.id = p.salary_id
		WHERE p.id = ?
	`, id)
	return scanSalaryPayment(row)
}

// Create inserta un nuevo pago de salario.
func (s *SalaryPaymentStorage) Create(ctx context.Context, data *models.SalaryPayment) (*models.SalaryPayment, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO salary_payments (salary_id, year, month, amount, currency, status, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, data.SalaryID, data.Year, data.Month, data.Amount, data.Currency, data.Status, nullString(data.Notes))
	if err != nil {
		return nil, fmt.Errorf("insertar pago de salario: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de pago de salario: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Exists verifica si ya existe un pago para el salario en el período.
func (s *SalaryPaymentStorage) Exists(ctx context.Context, salaryID int64, year, month int) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM salary_payments
		WHERE salary_id = ? AND year = ? AND month = ?
	`, salaryID, year, month).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("verificar pago existente: %w", err)
	}
	return count > 0, nil
}

// MarkReceived marca un pago como recibido.
func (s *SalaryPaymentStorage) MarkReceived(ctx context.Context, id int64, receivedAt time.Time, receivedAmount *float64, notes *string) (*models.SalaryPayment, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE salary_payments
		SET status = 'received', received_at = ?, received_amount = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, receivedAt, nullFloat(receivedAmount), nullString(notes), id)
	if err != nil {
		return nil, fmt.Errorf("marcar pago de salario como recibido: %w", err)
	}
	return s.GetByID(ctx, id)
}

// MarkPending devuelve el pago a estado pendiente.
func (s *SalaryPaymentStorage) MarkPending(ctx context.Context, id int64) (*models.SalaryPayment, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE salary_payments
		SET status = 'pending', received_at = NULL, received_amount = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	if err != nil {
		return nil, fmt.Errorf("marcar pago de salario como pendiente: %w", err)
	}
	return s.GetByID(ctx, id)
}

// GetYearMonth devuelve el período (year, month) de un pago.
func (s *SalaryPaymentStorage) GetYearMonth(ctx context.Context, id int64) (year, month int, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT year, month FROM salary_payments WHERE id = ?`, id).Scan(&year, &month)
	if err != nil {
		return 0, 0, fmt.Errorf("obtener período de pago de salario: %w", err)
	}
	return year, month, nil
}

func scanSalaryPayment(row *sql.Row) (*models.SalaryPayment, error) {
	var p models.SalaryPayment
	var receivedAmount sql.NullFloat64
	var receivedAt sql.NullTime
	var notes sql.NullString
	if err := row.Scan(&p.ID, &p.SalaryID, &p.Employer, &p.Year, &p.Month, &p.Amount, &p.Currency, &p.Status,
		&receivedAmount, &receivedAt, &notes, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear pago de salario: %w", err)
	}
	if receivedAmount.Valid {
		p.ReceivedAmount = &receivedAmount.Float64
	}
	if receivedAt.Valid {
		p.ReceivedAt = &receivedAt.Time
	}
	if notes.Valid {
		p.Notes = &notes.String
	}
	return &p, nil
}
