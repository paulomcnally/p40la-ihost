package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// AutoStorage encapsula el acceso a la tabla autos.
type AutoStorage struct {
	db *sql.DB
}

// NewAutoStorage crea un nuevo AutoStorage.
func NewAutoStorage(db *sql.DB) *AutoStorage {
	return &AutoStorage{db: db}
}

// List devuelve todos los autos.
func (s *AutoStorage) List(ctx context.Context) ([]models.Auto, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, year, model, brand, color, icon, created_at, updated_at
		FROM autos
		ORDER BY brand, model
	`)
	if err != nil {
		return nil, fmt.Errorf("listar autos: %w", err)
	}
	defer rows.Close()

	return scanAutos(rows)
}

// GetByID busca un auto por su ID.
func (s *AutoStorage) GetByID(ctx context.Context, id int64) (*models.Auto, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, year, model, brand, color, icon, created_at, updated_at
		FROM autos
		WHERE id = ?
	`, id)
	return scanAuto(row)
}

// Create inserta un nuevo auto.
func (s *AutoStorage) Create(ctx context.Context, year int64, model, brand, color, icon string) (*models.Auto, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO autos (year, model, brand, color, icon) VALUES (?, ?, ?, ?, ?)
	`, year, model, brand, color, icon)
	if err != nil {
		return nil, fmt.Errorf("insertar auto: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de auto: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza un auto existente.
func (s *AutoStorage) Update(ctx context.Context, id int64, year int64, model, brand, color, icon string) (*models.Auto, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE autos
		SET year = ?, model = ?, brand = ?, color = ?, icon = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, year, model, brand, color, icon, id)
	if err != nil {
		return nil, fmt.Errorf("actualizar auto: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Delete elimina un auto.
func (s *AutoStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM autos WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar auto: %w", err)
	}
	return nil
}

func scanAuto(row *sql.Row) (*models.Auto, error) {
	var a models.Auto
	if err := row.Scan(&a.ID, &a.Year, &a.Model, &a.Brand, &a.Color, &a.Icon, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear auto: %w", err)
	}
	return &a, nil
}

func scanAutos(rows *sql.Rows) ([]models.Auto, error) {
	var autos []models.Auto
	for rows.Next() {
		var a models.Auto
		if err := rows.Scan(&a.ID, &a.Year, &a.Model, &a.Brand, &a.Color, &a.Icon, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear auto: %w", err)
		}
		autos = append(autos, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return autos, nil
}
