package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

type InstitutionCategoryStorage struct {
	db *sql.DB
}

func NewInstitutionCategoryStorage(db *sql.DB) *InstitutionCategoryStorage {
	return &InstitutionCategoryStorage{db: db}
}

func (s *InstitutionCategoryStorage) List(ctx context.Context) ([]models.InstitutionCategory, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, key, name, description, icon_key, created_at, updated_at
		FROM institution_categories ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("listar categorías: %w", err)
	}
	defer rows.Close()

	var cats []models.InstitutionCategory
	for rows.Next() {
		var c models.InstitutionCategory
		if err := rows.Scan(&c.ID, &c.Key, &c.Name, &c.Description, &c.IconKey, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear categoría: %w", err)
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (s *InstitutionCategoryStorage) GetByID(ctx context.Context, id int64) (*models.InstitutionCategory, error) {
	var c models.InstitutionCategory
	if err := s.db.QueryRowContext(ctx,
		"SELECT id, key, name, description, icon_key, created_at, updated_at FROM institution_categories WHERE id = ?", id,
	).Scan(&c.ID, &c.Key, &c.Name, &c.Description, &c.IconKey, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("obtener categoría: %w", err)
	}
	return &c, nil
}

func (s *InstitutionCategoryStorage) GetByKey(ctx context.Context, key string) (*models.InstitutionCategory, error) {
	var c models.InstitutionCategory
	if err := s.db.QueryRowContext(ctx,
		"SELECT id, key, name, description, icon_key, created_at, updated_at FROM institution_categories WHERE key = ?", key,
	).Scan(&c.ID, &c.Key, &c.Name, &c.Description, &c.IconKey, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("obtener categoría por key: %w", err)
	}
	return &c, nil
}

func (s *InstitutionCategoryStorage) Create(ctx context.Context, cat *models.InstitutionCategory) (*models.InstitutionCategory, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO institution_categories (key, name, description, icon_key) VALUES (?, ?, ?, ?)",
		cat.Key, cat.Name, cat.Description, cat.IconKey,
	)
	if err != nil {
		return nil, fmt.Errorf("crear categoría: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener ID: %w", err)
	}
	return s.GetByID(ctx, id)
}

func (s *InstitutionCategoryStorage) Update(ctx context.Context, cat *models.InstitutionCategory) (*models.InstitutionCategory, error) {
	_, err := s.db.ExecContext(ctx,
		"UPDATE institution_categories SET name = ?, description = ?, icon_key = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		cat.Name, cat.Description, cat.IconKey, cat.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("actualizar categoría: %w", err)
	}
	return s.GetByID(ctx, cat.ID)
}

func (s *InstitutionCategoryStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM institution_categories WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("eliminar categoría: %w", err)
	}
	return nil
}

func (s *InstitutionCategoryStorage) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM institution_categories").Scan(&count); err != nil {
		return 0, fmt.Errorf("contar categorías: %w", err)
	}
	return count, nil
}
