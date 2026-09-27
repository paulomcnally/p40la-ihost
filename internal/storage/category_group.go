package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// CategoryGroupStorage encapsula el acceso a la tabla category_groups.
type CategoryGroupStorage struct {
	db *sql.DB
}

// NewCategoryGroupStorage crea un nuevo CategoryGroupStorage.
func NewCategoryGroupStorage(db *sql.DB) *CategoryGroupStorage {
	return &CategoryGroupStorage{db: db}
}

const categoryGroupColumns = "id, name, icon, sort_order, created_at, updated_at"

// List devuelve todos los grupos ordenados por sort_order.
func (s *CategoryGroupStorage) List(ctx context.Context) ([]models.CategoryGroup, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+categoryGroupColumns+`
		FROM category_groups
		ORDER BY sort_order, id
	`)
	if err != nil {
		return nil, fmt.Errorf("listar grupos de categorías: %w", err)
	}
	defer rows.Close()

	var groups []models.CategoryGroup
	for rows.Next() {
		var g models.CategoryGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Icon, &g.SortOrder, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear grupo de categoría: %w", err)
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return groups, nil
}

// GetByID busca un grupo por ID.
func (s *CategoryGroupStorage) GetByID(ctx context.Context, id int64) (*models.CategoryGroup, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+categoryGroupColumns+`
		FROM category_groups
		WHERE id = ?
	`, id)
	return scanCategoryGroup(row)
}

// Create inserta un nuevo grupo.
func (s *CategoryGroupStorage) Create(ctx context.Context, name, icon string, sortOrder int) (*models.CategoryGroup, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO category_groups (name, icon, sort_order) VALUES (?, ?, ?)
	`, name, icon, sortOrder)
	if err != nil {
		return nil, fmt.Errorf("insertar grupo de categoría: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de grupo: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza un grupo existente.
func (s *CategoryGroupStorage) Update(ctx context.Context, id int64, name, icon string, sortOrder int) (*models.CategoryGroup, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE category_groups
		SET name = ?, icon = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, icon, sortOrder, id)
	if err != nil {
		return nil, fmt.Errorf("actualizar grupo de categoría: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Delete elimina un grupo (debe estar vacío de categorías activas).
func (s *CategoryGroupStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM category_groups WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar grupo de categoría: %w", err)
	}
	return nil
}

// CountCategories cuenta las categorías no archivadas de un grupo.
func (s *CategoryGroupStorage) CountCategories(ctx context.Context, groupID int64) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM categories
		WHERE category_group_id = ? AND deleted_at IS NULL
	`, groupID).Scan(&count); err != nil {
		return 0, fmt.Errorf("contar categorías del grupo: %w", err)
	}
	return count, nil
}

// Reorder actualiza el sort_order de todos los grupos.
func (s *CategoryGroupStorage) Reorder(ctx context.Context, ids []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar reorden de grupos: %w", err)
	}
	defer tx.Rollback()

	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `
			UPDATE category_groups SET sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
		`, i, id); err != nil {
			return fmt.Errorf("actualizar orden de grupo %d: %w", id, err)
		}
	}
	return tx.Commit()
}

func scanCategoryGroup(row *sql.Row) (*models.CategoryGroup, error) {
	var g models.CategoryGroup
	if err := row.Scan(&g.ID, &g.Name, &g.Icon, &g.SortOrder, &g.CreatedAt, &g.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear grupo de categoría: %w", err)
	}
	return &g, nil
}