package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// BudgetMonthStorage encapsula el acceso a la tabla budget_months.
type BudgetMonthStorage struct {
	db *sql.DB
}

// NewBudgetMonthStorage crea un nuevo BudgetMonthStorage.
func NewBudgetMonthStorage(db *sql.DB) *BudgetMonthStorage {
	return &BudgetMonthStorage{db: db}
}

const budgetMonthColumns = "id, year, month, created_at"

// GetByYearMonth busca un mes por año/mes (devuelve nil si no existe).
func (s *BudgetMonthStorage) GetByYearMonth(ctx context.Context, year, month int) (*models.BudgetMonth, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+budgetMonthColumns+`
		FROM budget_months
		WHERE year = ? AND month = ?
	`, year, month)
	var bm models.BudgetMonth
	if err := row.Scan(&bm.ID, &bm.Year, &bm.Month, &bm.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("buscar mes de presupuesto: %w", err)
	}
	return &bm, nil
}

// GetOrCreate devuelve el mes o lo crea si no existe.
func (s *BudgetMonthStorage) GetOrCreate(ctx context.Context, year, month int) (*models.BudgetMonth, error) {
	bm, err := s.GetByYearMonth(ctx, year, month)
	if err != nil {
		return nil, err
	}
	if bm != nil {
		return bm, nil
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO budget_months (year, month) VALUES (?, ?)
	`, year, month)
	if err != nil {
		return nil, fmt.Errorf("insertar mes de presupuesto: %w", err)
	}
	if _, err := result.RowsAffected(); err != nil {
		return nil, err
	}
	return s.GetByYearMonth(ctx, year, month)
}