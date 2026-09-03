package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// SalaryStorage encapsula el acceso a la tabla salaries.
type SalaryStorage struct {
	db *sql.DB
}

// NewSalaryStorage crea un nuevo SalaryStorage.
func NewSalaryStorage(db *sql.DB) *SalaryStorage {
	return &SalaryStorage{db: db}
}

// List devuelve todos los salarios.
func (s *SalaryStorage) List(ctx context.Context) ([]models.Salary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, employer, amount, currency_id, payment_day, active, note, created_at, updated_at
		FROM salaries
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listar salarios: %w", err)
	}
	defer rows.Close()

	return scanSalaries(rows)
}

// GetByID busca un salario por su ID.
func (s *SalaryStorage) GetByID(ctx context.Context, id int64) (*models.Salary, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, employer, amount, currency_id, payment_day, active, note, created_at, updated_at
		FROM salaries
		WHERE id = ?
	`, id)
	return scanSalary(row)
}

// Create inserta un nuevo salario.
func (s *SalaryStorage) Create(ctx context.Context, employer string, amount float64, currencyID int64, paymentDay int, active bool, note string) (*models.Salary, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO salaries (employer, amount, currency_id, payment_day, active, note) VALUES (?, ?, ?, ?, ?, ?)
	`, employer, amount, currencyID, paymentDay, active, note)
	if err != nil {
		return nil, fmt.Errorf("insertar salario: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de salario: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza un salario existente.
func (s *SalaryStorage) Update(ctx context.Context, id int64, employer string, amount float64, currencyID int64, paymentDay int, active bool, note string) (*models.Salary, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE salaries
		SET employer = ?, amount = ?, currency_id = ?, payment_day = ?, active = ?, note = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, employer, amount, currencyID, paymentDay, active, note, id)
	if err != nil {
		return nil, fmt.Errorf("actualizar salario: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Delete elimina un salario.
func (s *SalaryStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM salaries WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar salario: %w", err)
	}
	return nil
}

func scanSalary(row *sql.Row) (*models.Salary, error) {
	var c models.Salary
	var note sql.NullString
	if err := row.Scan(&c.ID, &c.Employer, &c.Amount, &c.CurrencyID, &c.PaymentDay, &c.Active, &note, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear salario: %w", err)
	}
	c.Note = note.String
	return &c, nil
}

func scanSalaries(rows *sql.Rows) ([]models.Salary, error) {
	var salaries []models.Salary
	for rows.Next() {
		var c models.Salary
		var note sql.NullString
		if err := rows.Scan(&c.ID, &c.Employer, &c.Amount, &c.CurrencyID, &c.PaymentDay, &c.Active, &note, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear salario: %w", err)
		}
		c.Note = note.String
		salaries = append(salaries, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return salaries, nil
}