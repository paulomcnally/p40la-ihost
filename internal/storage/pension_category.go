package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// PensionCategoryStorage encapsula el acceso a la tabla pension_categories.
type PensionCategoryStorage struct {
	db *sql.DB
}

// NewPensionCategoryStorage crea un nuevo PensionCategoryStorage.
func NewPensionCategoryStorage(db *sql.DB) *PensionCategoryStorage {
	return &PensionCategoryStorage{db: db}
}

const pensionCategoryColumns = `id, name, description, auto_generate, share_percent, currency, created_at, updated_at`

// List devuelve todas las categorías de pensión.
func (s *PensionCategoryStorage) List(ctx context.Context) ([]models.PensionCategory, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+pensionCategoryColumns+`
		FROM pension_categories
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("listar categorías de pensión: %w", err)
	}
	defer rows.Close()

	return scanPensionCategories(rows)
}

// GetByID busca una categoría de pensión por su ID.
func (s *PensionCategoryStorage) GetByID(ctx context.Context, id int64) (*models.PensionCategory, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+pensionCategoryColumns+`
		FROM pension_categories
		WHERE id = ?
	`, id)
	return scanPensionCategory(row)
}

// Create inserta una nueva categoría de pensión.
func (s *PensionCategoryStorage) Create(ctx context.Context, name, description string, autoGenerate bool, sharePercent int, currency string) (*models.PensionCategory, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO pension_categories (name, description, auto_generate, share_percent, currency)
		VALUES (?, ?, ?, ?, ?)
	`, name, description, autoGenerate, sharePercent, currency)
	if err != nil {
		return nil, fmt.Errorf("insertar categoría de pensión: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de categoría de pensión: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza una categoría de pensión existente.
func (s *PensionCategoryStorage) Update(ctx context.Context, id int64, name, description string, autoGenerate bool, sharePercent int, currency string) (*models.PensionCategory, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE pension_categories
		SET name = ?, description = ?, auto_generate = ?, share_percent = ?, currency = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, description, autoGenerate, sharePercent, currency, id)
	if err != nil {
		return nil, fmt.Errorf("actualizar categoría de pensión: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Delete elimina una categoría de pensión.
func (s *PensionCategoryStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM pension_categories WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar categoría de pensión: %w", err)
	}
	return nil
}

func scanPensionCategory(row *sql.Row) (*models.PensionCategory, error) {
	var c models.PensionCategory
	var description sql.NullString
	var autoGenerate int
	if err := row.Scan(&c.ID, &c.Name, &description, &autoGenerate, &c.SharePercent, &c.Currency, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear categoría de pensión: %w", err)
	}
	c.Description = description.String
	c.AutoGenerate = autoGenerate != 0
	return &c, nil
}

func scanPensionCategories(rows *sql.Rows) ([]models.PensionCategory, error) {
	var categories []models.PensionCategory
	for rows.Next() {
		var c models.PensionCategory
		var description sql.NullString
		var autoGenerate int
		if err := rows.Scan(&c.ID, &c.Name, &description, &autoGenerate, &c.SharePercent, &c.Currency, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear categoría de pensión: %w", err)
		}
		c.Description = description.String
		c.AutoGenerate = autoGenerate != 0
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}
