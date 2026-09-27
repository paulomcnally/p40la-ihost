package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// RecurringRuleStorage encapsula el acceso a la tabla recurring_rules.
type RecurringRuleStorage struct {
	db *sql.DB
}

// NewRecurringRuleStorage crea un nuevo RecurringRuleStorage.
func NewRecurringRuleStorage(db *sql.DB) *RecurringRuleStorage {
	return &RecurringRuleStorage{db: db}
}

const recurringRuleColumns = "id, category_id, amount, frequency, start_month, end_month, active, created_at, updated_at"

// ListActiveByCategory devuelve las reglas activas de una categoría.
func (s *RecurringRuleStorage) ListActiveByCategory(ctx context.Context, categoryID int64) ([]models.RecurringRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+recurringRuleColumns+`
		FROM recurring_rules
		WHERE category_id = ? AND active = 1
		ORDER BY start_month
	`, categoryID)
	if err != nil {
		return nil, fmt.Errorf("listar reglas recurrentes de categoría: %w", err)
	}
	defer rows.Close()
	return scanRecurringRules(rows)
}

// ListActive devuelve todas las reglas activas.
func (s *RecurringRuleStorage) ListActive(ctx context.Context) ([]models.RecurringRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+recurringRuleColumns+`
		FROM recurring_rules
		WHERE active = 1
		ORDER BY category_id, start_month
	`)
	if err != nil {
		return nil, fmt.Errorf("listar reglas recurrentes: %w", err)
	}
	defer rows.Close()
	return scanRecurringRules(rows)
}

// GetByID busca una regla por ID.
func (s *RecurringRuleStorage) GetByID(ctx context.Context, id int64) (*models.RecurringRule, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+recurringRuleColumns+`
		FROM recurring_rules
		WHERE id = ?
	`, id)
	return scanRecurringRule(row)
}

// Create inserta una nueva regla.
func (s *RecurringRuleStorage) Create(ctx context.Context, rule *models.RecurringRule) (*models.RecurringRule, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO recurring_rules (category_id, amount, frequency, start_month, end_month, active)
		VALUES (?, ?, ?, ?, ?, ?)
	`, rule.CategoryID, rule.Amount, rule.Frequency, rule.StartMonth, rule.EndMonth, boolToInt(rule.Active))
	if err != nil {
		return nil, fmt.Errorf("insertar regla recurrente: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de regla recurrente: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza una regla existente.
func (s *RecurringRuleStorage) Update(ctx context.Context, rule *models.RecurringRule) (*models.RecurringRule, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE recurring_rules
		SET amount = ?, frequency = ?, start_month = ?, end_month = ?, active = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, rule.Amount, rule.Frequency, rule.StartMonth, rule.EndMonth, boolToInt(rule.Active), rule.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar regla recurrente: %w", err)
	}
	return s.GetByID(ctx, rule.ID)
}

// Delete elimina una regla.
func (s *RecurringRuleStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM recurring_rules WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar regla recurrente: %w", err)
	}
	return nil
}

func scanRecurringRule(row *sql.Row) (*models.RecurringRule, error) {
	var r models.RecurringRule
	var active int
	var endMonth sql.NullString
	if err := row.Scan(&r.ID, &r.CategoryID, &r.Amount, &r.Frequency, &r.StartMonth,
		&endMonth, &active, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear regla recurrente: %w", err)
	}
	r.Active = active == 1
	if endMonth.Valid {
		r.EndMonth = endMonth.String
	}
	return &r, nil
}

func scanRecurringRules(rows *sql.Rows) ([]models.RecurringRule, error) {
	var rules []models.RecurringRule
	for rows.Next() {
		var r models.RecurringRule
		var active int
		var endMonth sql.NullString
		if err := rows.Scan(&r.ID, &r.CategoryID, &r.Amount, &r.Frequency, &r.StartMonth,
			&endMonth, &active, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear regla recurrente: %w", err)
		}
		r.Active = active == 1
		if endMonth.Valid {
			r.EndMonth = endMonth.String
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}