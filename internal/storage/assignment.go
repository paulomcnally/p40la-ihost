package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// AssignmentStorage encapsula el acceso a la tabla assignments.
type AssignmentStorage struct {
	db *sql.DB
}

// NewAssignmentStorage crea un nuevo AssignmentStorage.
func NewAssignmentStorage(db *sql.DB) *AssignmentStorage {
	return &AssignmentStorage{db: db}
}

const assignmentColumns = `
	a.id, a.budget_month_id, a.category_id, a.currency_id, COALESCE(c.code, ''),
	a.amount, a.source, a.recurring_rule_id, a.created_at, a.updated_at
`

// ListByMonth devuelve las asignaciones de un mes con su moneda.
func (s *AssignmentStorage) ListByMonth(ctx context.Context, budgetMonthID int64) ([]models.Assignment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+assignmentColumns+`
		FROM assignments a
		LEFT JOIN currencies c ON c.id = a.currency_id
		WHERE a.budget_month_id = ?
	`, budgetMonthID)
	if err != nil {
		return nil, fmt.Errorf("listar asignaciones del mes: %w", err)
	}
	defer rows.Close()

	var assignments []models.Assignment
	for rows.Next() {
		var a models.Assignment
		var code sql.NullString
		var ruleID sql.NullInt64
		if err := rows.Scan(&a.ID, &a.BudgetMonthID, &a.CategoryID, &a.CurrencyID, &code,
			&a.Amount, &a.Source, &ruleID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear asignación: %w", err)
		}
		a.CurrencyCode = code.String
		if ruleID.Valid {
			a.RecurringRuleID = ruleID.Int64
		}
		assignments = append(assignments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return assignments, nil
}

// FindByMonthCategoryCurrency busca una asignación específica.
func (s *AssignmentStorage) FindByMonthCategoryCurrency(ctx context.Context, budgetMonthID, categoryID, currencyID int64) (*models.Assignment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+assignmentColumns+`
		FROM assignments a
		LEFT JOIN currencies c ON c.id = a.currency_id
		WHERE a.budget_month_id = ? AND a.category_id = ? AND a.currency_id = ?
	`, budgetMonthID, categoryID, currencyID)
	return scanAssignment(row)
}

// Upsert inserta o actualiza la asignación (única por mes/categoría/moneda).
func (s *AssignmentStorage) Upsert(ctx context.Context, monthID, categoryID, currencyID int64, amount float64, source string, ruleID *int64) (*models.Assignment, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM assignments
		WHERE budget_month_id = ? AND category_id = ? AND currency_id = ?
	`, monthID, categoryID, currencyID).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO assignments (budget_month_id, category_id, currency_id, amount, source, recurring_rule_id)
			VALUES (?, ?, ?, ?, ?, ?)
		`, monthID, categoryID, currencyID, amount, source, ruleID)
		if err != nil {
			return nil, fmt.Errorf("insertar asignación: %w", err)
		}
	case err != nil:
		return nil, fmt.Errorf("buscar asignación existente: %w", err)
	default:
		_, err := s.db.ExecContext(ctx, `
			UPDATE assignments
			SET amount = ?, source = ?, recurring_rule_id = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, amount, source, ruleID, id)
		if err != nil {
			return nil, fmt.Errorf("actualizar asignación: %w", err)
		}
	}
	return s.FindByMonthCategoryCurrency(ctx, monthID, categoryID, currencyID)
}

// UpsertGenerated inserta una asignación generada por regla recurrente solo si no existe.
func (s *AssignmentStorage) UpsertGenerated(ctx context.Context, monthID, categoryID, currencyID int64, amount float64, ruleID int64) (*models.Assignment, error) {
	existing, err := s.FindByMonthCategoryCurrency(ctx, monthID, categoryID, currencyID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	return s.Upsert(ctx, monthID, categoryID, currencyID, amount, "recurring", &ruleID)
}

// SumByMonthGrouped devuelve la suma de asignaciones por categoría y moneda para un mes.
func (s *AssignmentStorage) SumByMonthGrouped(ctx context.Context, budgetMonthID int64) (map[int64]map[int64]float64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT category_id, currency_id, SUM(amount)
		FROM assignments
		WHERE budget_month_id = ?
		GROUP BY category_id, currency_id
	`, budgetMonthID)
	if err != nil {
		return nil, fmt.Errorf("sumar asignaciones del mes: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]map[int64]float64)
	for rows.Next() {
		var categoryID, currencyID int64
		var sum float64
		if err := rows.Scan(&categoryID, &currencyID, &sum); err != nil {
			return nil, fmt.Errorf("escanear suma de asignaciones: %w", err)
		}
		if result[categoryID] == nil {
			result[categoryID] = make(map[int64]float64)
		}
		result[categoryID][currencyID] = sum
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanAssignment(row *sql.Row) (*models.Assignment, error) {
	var a models.Assignment
	var code sql.NullString
	var ruleID sql.NullInt64
	if err := row.Scan(&a.ID, &a.BudgetMonthID, &a.CategoryID, &a.CurrencyID, &code,
		&a.Amount, &a.Source, &ruleID, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear asignación: %w", err)
	}
	a.CurrencyCode = code.String
	if ruleID.Valid {
		a.RecurringRuleID = ruleID.Int64
	}
	return &a, nil
}