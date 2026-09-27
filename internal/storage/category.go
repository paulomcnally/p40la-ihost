package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// nullableStringValue devuelve nil si la cadena está vacía (para columnas
// NULL-able), o el valor si no.
func nullableStringValue(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// CategoryStorage encapsula el acceso a la tabla categories.
type CategoryStorage struct {
	db *sql.DB
}

// NewCategoryStorage crea un nuevo CategoryStorage.
func NewCategoryStorage(db *sql.DB) *CategoryStorage {
	return &CategoryStorage{db: db}
}

const categoryColumns = `
	c.id, c.category_group_id, c.name, c.icon, c.sort_order,
	c.target_amount, c.target_type, c.target_date,
	c.service_id, c.account_id, COALESCE(s.name, ''),
	c.deleted_at, c.created_at, c.updated_at
`

const categoryFromJoins = `
	FROM categories c
	LEFT JOIN services s ON s.id = c.service_id
`

// ListByGroup devuelve las categorías no archivadas de un grupo, ordenadas.
func (s *CategoryStorage) ListByGroup(ctx context.Context, groupID int64) ([]models.Category, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+categoryColumns+`
		`+categoryFromJoins+`
		WHERE c.category_group_id = ? AND c.deleted_at IS NULL
		ORDER BY c.sort_order, c.id
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("listar categorías del grupo: %w", err)
	}
	defer rows.Close()
	return scanCategories(rows)
}

// ListAll devuelve todas las categorías no archivadas con su grupo.
func (s *CategoryStorage) ListAll(ctx context.Context) ([]models.Category, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+categoryColumns+`
		`+categoryFromJoins+`
		WHERE c.deleted_at IS NULL
		ORDER BY c.category_group_id, c.sort_order, c.id
	`)
	if err != nil {
		return nil, fmt.Errorf("listar categorías: %w", err)
	}
	defer rows.Close()
	return scanCategories(rows)
}

// GetByID busca una categoría por ID (incluye archivadas para historial).
func (s *CategoryStorage) GetByID(ctx context.Context, id int64) (*models.Category, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+categoryColumns+`
		`+categoryFromJoins+`
		WHERE c.id = ?
	`, id)
	return scanCategory(row)
}

// GetByServiceID devuelve la categoría activa vinculada a un servicio
// (SPEC-094). El índice único parcial garantiza a lo sumo una fila activa.
func (s *CategoryStorage) GetByServiceID(ctx context.Context, serviceID int64) (*models.Category, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+categoryColumns+`
		`+categoryFromJoins+`
		WHERE c.service_id = ? AND c.deleted_at IS NULL
	`, serviceID)
	return scanCategory(row)
}

// ServiceLinkInUse indica si el servicio ya está vinculado a otra categoría
// activa distinta de excludeCategoryID (SPEC-094, validación de unicidad).
func (s *CategoryStorage) ServiceLinkInUse(ctx context.Context, serviceID, excludeCategoryID int64) (bool, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM categories
		WHERE service_id = ? AND deleted_at IS NULL AND id != ?
	`, serviceID, excludeCategoryID).Scan(&count); err != nil {
		return false, fmt.Errorf("verificar vínculo de servicio: %w", err)
	}
	return count > 0, nil
}

// Create inserta una nueva categoría.
func (s *CategoryStorage) Create(ctx context.Context, cat *models.Category) (*models.Category, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO categories (category_group_id, name, icon, sort_order,
		                        target_amount, target_type, target_date,
		                        service_id, account_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, cat.CategoryGroupID, cat.Name, cat.Icon, cat.SortOrder,
		cat.TargetAmount, nullableStringValue(cat.TargetType), nullableStringValue(cat.TargetDate),
		cat.ServiceID, cat.AccountID)
	if err != nil {
		return nil, fmt.Errorf("insertar categoría: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de categoría: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza una categoría existente.
func (s *CategoryStorage) Update(ctx context.Context, cat *models.Category) (*models.Category, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE categories
		SET category_group_id = ?, name = ?, icon = ?, sort_order = ?,
		    target_amount = ?, target_type = ?, target_date = ?,
		    service_id = ?, account_id = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, cat.CategoryGroupID, cat.Name, cat.Icon, cat.SortOrder,
		cat.TargetAmount, nullableStringValue(cat.TargetType), nullableStringValue(cat.TargetDate),
		cat.ServiceID, cat.AccountID, cat.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar categoría: %w", err)
	}
	return s.GetByID(ctx, cat.ID)
}

// SoftDelete archiva una categoría (oculta de la vista mensual, conserva historial).
func (s *CategoryStorage) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE categories SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("archivar categoría: %w", err)
	}
	return nil
}

// HasTransactions indica si la categoría tiene transacciones asociadas.
func (s *CategoryStorage) HasTransactions(ctx context.Context, id int64) (bool, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM transactions WHERE category_id = ? AND deleted_at IS NULL
	`, id).Scan(&count); err != nil {
		return false, fmt.Errorf("verificar transacciones de categoría: %w", err)
	}
	return count > 0, nil
}

// HasAssignments indica si la categoría tiene asignaciones.
func (s *CategoryStorage) HasAssignments(ctx context.Context, id int64) (bool, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM assignments WHERE category_id = ?
	`, id).Scan(&count); err != nil {
		return false, fmt.Errorf("verificar asignaciones de categoría: %w", err)
	}
	return count > 0, nil
}

// Reorder actualiza el sort_order de las categorías de un grupo.
func (s *CategoryStorage) Reorder(ctx context.Context, groupID int64, ids []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar reorden de categorías: %w", err)
	}
	defer tx.Rollback()

	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `
			UPDATE categories SET sort_order = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND category_group_id = ? AND deleted_at IS NULL
		`, i, id, groupID); err != nil {
			return fmt.Errorf("actualizar orden de categoría %d: %w", id, err)
		}
	}
	return tx.Commit()
}

func scanCategory(row *sql.Row) (*models.Category, error) {
	var c models.Category
	var targetAmount sql.NullFloat64
	var targetType, targetDate sql.NullString
	var serviceID, accountID sql.NullInt64
	var serviceName sql.NullString
	var deletedAt sql.NullTime
	if err := row.Scan(&c.ID, &c.CategoryGroupID, &c.Name, &c.Icon, &c.SortOrder,
		&targetAmount, &targetType, &targetDate,
		&serviceID, &accountID, &serviceName,
		&deletedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear categoría: %w", err)
	}
	if targetAmount.Valid {
		c.TargetAmount = &targetAmount.Float64
	}
	if targetType.Valid {
		c.TargetType = targetType.String
	}
	if targetDate.Valid {
		c.TargetDate = targetDate.String
	}
	if serviceID.Valid {
		sid := serviceID.Int64
		c.ServiceID = &sid
	}
	if accountID.Valid {
		aid := accountID.Int64
		c.AccountID = &aid
	}
	if serviceName.Valid {
		c.ServiceName = serviceName.String
	}
	if deletedAt.Valid {
		c.DeletedAt = &deletedAt.Time
	}
	return &c, nil
}

func scanCategories(rows *sql.Rows) ([]models.Category, error) {
	var cats []models.Category
	for rows.Next() {
		var c models.Category
		var targetAmount sql.NullFloat64
		var targetType, targetDate sql.NullString
		var serviceID, accountID sql.NullInt64
		var serviceName sql.NullString
		var deletedAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.CategoryGroupID, &c.Name, &c.Icon, &c.SortOrder,
			&targetAmount, &targetType, &targetDate,
			&serviceID, &accountID, &serviceName,
			&deletedAt, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear categoría: %w", err)
		}
		if targetAmount.Valid {
			c.TargetAmount = &targetAmount.Float64
		}
		if targetType.Valid {
			c.TargetType = targetType.String
		}
		if targetDate.Valid {
			c.TargetDate = targetDate.String
		}
		if serviceID.Valid {
			sid := serviceID.Int64
			c.ServiceID = &sid
		}
		if accountID.Valid {
			aid := accountID.Int64
			c.AccountID = &aid
		}
		if serviceName.Valid {
			c.ServiceName = serviceName.String
		}
		if deletedAt.Valid {
			c.DeletedAt = &deletedAt.Time
		}
		cats = append(cats, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cats, nil
}
