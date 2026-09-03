package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// MonthClosingStorage encapsula el acceso a la tabla month_closings.
type MonthClosingStorage struct {
	db *sql.DB
}

// NewMonthClosingStorage crea un nuevo MonthClosingStorage.
func NewMonthClosingStorage(db *sql.DB) *MonthClosingStorage {
	return &MonthClosingStorage{db: db}
}

// IsClosed indica si un mes está cerrado.
func (s *MonthClosingStorage) IsClosed(ctx context.Context, year, month int) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM month_closings WHERE year = ? AND month = ?
	`, year, month).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("verificar cierre de mes: %w", err)
	}
	return count > 0, nil
}

// Find devuelve el cierre de un mes, o nil si no existe.
func (s *MonthClosingStorage) Find(ctx context.Context, year, month int) (*models.MonthClosing, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, year, month, closed_at FROM month_closings WHERE year = ? AND month = ?
	`, year, month)
	return scanMonthClosing(row)
}

// Close inserta el cierre de un mes.
func (s *MonthClosingStorage) Close(ctx context.Context, year, month int) (*models.MonthClosing, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO month_closings (year, month) VALUES (?, ?)
	`, year, month)
	if err != nil {
		return nil, fmt.Errorf("cerrar mes: %w", err)
	}
	return s.Find(ctx, year, month)
}

// Reopen elimina el cierre de un mes.
func (s *MonthClosingStorage) Reopen(ctx context.Context, year, month int) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM month_closings WHERE year = ? AND month = ?
	`, year, month)
	if err != nil {
		return fmt.Errorf("reabrir mes: %w", err)
	}
	return nil
}

func scanMonthClosing(row *sql.Row) (*models.MonthClosing, error) {
	var c models.MonthClosing
	if err := row.Scan(&c.ID, &c.Year, &c.Month, &c.ClosedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear cierre de mes: %w", err)
	}
	return &c, nil
}
