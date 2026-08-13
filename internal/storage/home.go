package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// HomeStorage encapsula el acceso a la tabla homes.
type HomeStorage struct {
	db *sql.DB
}

// NewHomeStorage crea un nuevo HomeStorage.
func NewHomeStorage(db *sql.DB) *HomeStorage {
	return &HomeStorage{db: db}
}

// List devuelve todos los hogares no eliminados.
func (s *HomeStorage) List(ctx context.Context) ([]models.Home, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, address, deleted_at, created_at, updated_at
		FROM homes
		WHERE deleted_at IS NULL
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("listar hogares: %w", err)
	}
	defer rows.Close()

	return scanHomes(rows)
}

// GetByID busca un hogar por su ID.
func (s *HomeStorage) GetByID(ctx context.Context, id int64) (*models.Home, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, address, deleted_at, created_at, updated_at
		FROM homes
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanHome(row)
}

// Count devuelve la cantidad de hogares no eliminados.
func (s *HomeStorage) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM homes WHERE deleted_at IS NULL
	`).Scan(&count); err != nil {
		return 0, fmt.Errorf("contar hogares: %w", err)
	}
	return count, nil
}

// Create inserta un nuevo hogar.
func (s *HomeStorage) Create(ctx context.Context, name, address string) (*models.Home, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO homes (name, address) VALUES (?, ?)
	`, name, address)
	if err != nil {
		return nil, fmt.Errorf("insertar hogar: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de hogar: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza un hogar existente.
func (s *HomeStorage) Update(ctx context.Context, id int64, name, address string) (*models.Home, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE homes
		SET name = ?, address = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, name, address, id)
	if err != nil {
		return nil, fmt.Errorf("actualizar hogar: %w", err)
	}
	return s.GetByID(ctx, id)
}

// SoftDelete marca un hogar como eliminado.
func (s *HomeStorage) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE homes
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar hogar: %w", err)
	}
	return nil
}

func scanHome(row *sql.Row) (*models.Home, error) {
	var h models.Home
	var deletedAt sql.NullTime
	if err := row.Scan(&h.ID, &h.Name, &h.Address, &deletedAt, &h.CreatedAt, &h.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear hogar: %w", err)
	}
	if deletedAt.Valid {
		h.DeletedAt = &deletedAt.Time
	}
	return &h, nil
}

func scanHomes(rows *sql.Rows) ([]models.Home, error) {
	var homes []models.Home
	for rows.Next() {
		var h models.Home
		var deletedAt sql.NullTime
		if err := rows.Scan(&h.ID, &h.Name, &h.Address, &deletedAt, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear hogar: %w", err)
		}
		if deletedAt.Valid {
			h.DeletedAt = &deletedAt.Time
		}
		homes = append(homes, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return homes, nil
}
