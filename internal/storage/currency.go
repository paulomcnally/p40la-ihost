package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// CurrencyStorage encapsula el acceso a la tabla currencies.
type CurrencyStorage struct {
	db *sql.DB
}

// NewCurrencyStorage crea un nuevo CurrencyStorage.
func NewCurrencyStorage(db *sql.DB) *CurrencyStorage {
	return &CurrencyStorage{db: db}
}

// List devuelve todas las monedas no eliminadas.
func (s *CurrencyStorage) List(ctx context.Context) ([]models.Currency, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, code, name, symbol, deleted_at, created_at, updated_at
		FROM currencies
		WHERE deleted_at IS NULL
		ORDER BY code
	`)
	if err != nil {
		return nil, fmt.Errorf("listar monedas: %w", err)
	}
	defer rows.Close()

	return scanCurrencies(rows)
}

// GetByID busca una moneda por su ID.
func (s *CurrencyStorage) GetByID(ctx context.Context, id int64) (*models.Currency, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, code, name, symbol, deleted_at, created_at, updated_at
		FROM currencies
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanCurrency(row)
}

// Create inserta una nueva moneda.
func (s *CurrencyStorage) Create(ctx context.Context, code, name, symbol string) (*models.Currency, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO currencies (code, name, symbol) VALUES (?, ?, ?)
	`, code, name, symbol)
	if err != nil {
		return nil, fmt.Errorf("insertar moneda: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de moneda: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza una moneda existente.
func (s *CurrencyStorage) Update(ctx context.Context, id int64, code, name, symbol string) (*models.Currency, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE currencies
		SET code = ?, name = ?, symbol = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, code, name, symbol, id)
	if err != nil {
		return nil, fmt.Errorf("actualizar moneda: %w", err)
	}
	return s.GetByID(ctx, id)
}

// SoftDelete marca una moneda como eliminada.
func (s *CurrencyStorage) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE currencies
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar moneda: %w", err)
	}
	return nil
}

func scanCurrency(row *sql.Row) (*models.Currency, error) {
	var c models.Currency
	var deletedAt sql.NullTime
	if err := row.Scan(&c.ID, &c.Code, &c.Name, &c.Symbol, &deletedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear moneda: %w", err)
	}
	if deletedAt.Valid {
		c.DeletedAt = &deletedAt.Time
	}
	return &c, nil
}

func scanCurrencies(rows *sql.Rows) ([]models.Currency, error) {
	var currencies []models.Currency
	for rows.Next() {
		var c models.Currency
		var deletedAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.Symbol, &deletedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear moneda: %w", err)
		}
		if deletedAt.Valid {
			c.DeletedAt = &deletedAt.Time
		}
		currencies = append(currencies, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return currencies, nil
}
