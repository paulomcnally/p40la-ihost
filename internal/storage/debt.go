package storage

import (
	"context"
	"database/sql"
	"fmt"

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
	d.start_date, d.status, d.deleted_at, d.created_at, d.updated_at
`

// List devuelve todas las deudas no eliminadas, con acreedor y moneda.
func (s *DebtStorage) List(ctx context.Context) ([]models.Debt, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+debtColumns+`
		FROM debts d
		LEFT JOIN institutions i ON i.id = d.institution_id
		LEFT JOIN currencies c ON c.id = d.currency_id
		WHERE d.deleted_at IS NULL
		ORDER BY d.description
	`)
	if err != nil {
		return nil, fmt.Errorf("listar deudas: %w", err)
	}
	defer rows.Close()

	return scanDebts(rows)
}

// GetByID busca una deuda por su ID.
func (s *DebtStorage) GetByID(ctx context.Context, id int64) (*models.Debt, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+debtColumns+`
		FROM debts d
		LEFT JOIN institutions i ON i.id = d.institution_id
		LEFT JOIN currencies c ON c.id = d.currency_id
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

func scanDebt(row *sql.Row) (*models.Debt, error) {
	var d models.Debt
	var deletedAt sql.NullTime
	var institutionName, currencyCode sql.NullString
	if err := row.Scan(&d.ID, &d.InstitutionID, &institutionName, &d.Identifier, &d.Description,
		&d.Total, &d.Principal, &d.CurrencyID, &currencyCode,
		&d.InstallmentsTotal, &d.InstallmentAmount, &d.InterestRate, &d.PaymentDay,
		&d.StartDate, &d.Status, &deletedAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear deuda: %w", err)
	}
	d.InstitutionName = institutionName.String
	d.CurrencyCode = currencyCode.String
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
		var institutionName, currencyCode sql.NullString
		if err := rows.Scan(&d.ID, &d.InstitutionID, &institutionName, &d.Identifier, &d.Description,
			&d.Total, &d.Principal, &d.CurrencyID, &currencyCode,
			&d.InstallmentsTotal, &d.InstallmentAmount, &d.InterestRate, &d.PaymentDay,
			&d.StartDate, &d.Status, &deletedAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear deuda: %w", err)
		}
		d.InstitutionName = institutionName.String
		d.CurrencyCode = currencyCode.String
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
