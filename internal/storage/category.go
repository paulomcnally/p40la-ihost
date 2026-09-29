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
	c.account_id,
	c.deleted_at, c.created_at, c.updated_at
`

const categoryFrom = `
	FROM categories c
`

// ListByGroup devuelve las categorías no archivadas de un grupo, ordenadas.
func (s *CategoryStorage) ListByGroup(ctx context.Context, groupID int64) ([]models.Category, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+categoryColumns+`
		`+categoryFrom+`
		WHERE c.category_group_id = ? AND c.deleted_at IS NULL
		ORDER BY c.sort_order, c.id
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("listar categorías del grupo: %w", err)
	}
	defer rows.Close()
	cats, err := scanCategories(rows)
	if err != nil {
		return nil, err
	}
	return s.attachServices(ctx, cats)
}

// ListAll devuelve todas las categorías no archivadas con su grupo.
func (s *CategoryStorage) ListAll(ctx context.Context) ([]models.Category, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+categoryColumns+`
		`+categoryFrom+`
		WHERE c.deleted_at IS NULL
		ORDER BY c.category_group_id, c.sort_order, c.id
	`)
	if err != nil {
		return nil, fmt.Errorf("listar categorías: %w", err)
	}
	defer rows.Close()
	cats, err := scanCategories(rows)
	if err != nil {
		return nil, err
	}
	return s.attachServices(ctx, cats)
}

// GetByID busca una categoría por ID (incluye archivadas para historial).
func (s *CategoryStorage) GetByID(ctx context.Context, id int64) (*models.Category, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+categoryColumns+`
		`+categoryFrom+`
		WHERE c.id = ?
	`, id)
	cat, err := scanCategory(row)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, nil
	}
	withServices, err := s.attachServices(ctx, []models.Category{*cat})
	if err != nil {
		return nil, err
	}
	return &withServices[0], nil
}

// GetByServiceID devuelve la categoría activa vinculada a un servicio a través
// de la tabla de enlace (SPEC-094/096). A lo sumo una categoría activa por
// servicio (validado en CategoryService.validate).
func (s *CategoryStorage) GetByServiceID(ctx context.Context, serviceID int64) (*models.Category, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+categoryColumns+`
		`+categoryFrom+`
		JOIN category_service_links l ON l.category_id = c.id
		WHERE l.service_id = ? AND c.deleted_at IS NULL
	`, serviceID)
	cat, err := scanCategory(row)
	if err != nil {
		return nil, err
	}
	if cat == nil {
		return nil, nil
	}
	withServices, err := s.attachServices(ctx, []models.Category{*cat})
	if err != nil {
		return nil, err
	}
	return &withServices[0], nil
}

// ServiceLinkInUse indica si el servicio ya está vinculado a otra categoría
// activa distinta de excludeCategoryID (SPEC-094, validación de unicidad).
func (s *CategoryStorage) ServiceLinkInUse(ctx context.Context, serviceID, excludeCategoryID int64) (bool, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM category_service_links l
		JOIN categories c ON c.id = l.category_id
		WHERE l.service_id = ? AND c.deleted_at IS NULL AND l.category_id != ?
	`, serviceID, excludeCategoryID).Scan(&count); err != nil {
		return false, fmt.Errorf("verificar vínculo de servicio: %w", err)
	}
	return count > 0, nil
}

// Create inserta una nueva categoría con sus servicios vinculados.
func (s *CategoryStorage) Create(ctx context.Context, cat *models.Category) (*models.Category, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar creación de categoría: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO categories (category_group_id, name, icon, sort_order,
		                        target_amount, target_type, target_date,
		                        account_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, cat.CategoryGroupID, cat.Name, cat.Icon, cat.SortOrder,
		cat.TargetAmount, nullableStringValue(cat.TargetType), nullableStringValue(cat.TargetDate),
		cat.AccountID)
	if err != nil {
		return nil, fmt.Errorf("insertar categoría: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de categoría: %w", err)
	}
	if err := replaceServiceLinks(ctx, tx, id, cat.ServiceIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit categoría: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza una categoría existente y reemplaza sus servicios.
func (s *CategoryStorage) Update(ctx context.Context, cat *models.Category) (*models.Category, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar actualización de categoría: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE categories
		SET category_group_id = ?, name = ?, icon = ?, sort_order = ?,
		    target_amount = ?, target_type = ?, target_date = ?,
		    account_id = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, cat.CategoryGroupID, cat.Name, cat.Icon, cat.SortOrder,
		cat.TargetAmount, nullableStringValue(cat.TargetType), nullableStringValue(cat.TargetDate),
		cat.AccountID, cat.ID); err != nil {
		return nil, fmt.Errorf("actualizar categoría: %w", err)
	}
	if err := replaceServiceLinks(ctx, tx, cat.ID, cat.ServiceIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit actualización de categoría: %w", err)
	}
	return s.GetByID(ctx, cat.ID)
}

// SoftDelete archiva una categoría (oculta de la vista mensual, conserva historial).
func (s *CategoryStorage) SoftDelete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar archivado de categoría: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE categories SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL
	`, id); err != nil {
		return fmt.Errorf("archivar categoría: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM category_service_links WHERE category_id = ?
	`, id); err != nil {
		return fmt.Errorf("limpiar vínculos de categoría: %w", err)
	}
	return tx.Commit()
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

// replaceServiceLinks reemplaza el set completo de servicios vinculados de una
// categoría (delete + insert) dentro de una transacción.
func replaceServiceLinks(ctx context.Context, tx *sql.Tx, categoryID int64, serviceIDs []int64) error {
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM category_service_links WHERE category_id = ?
	`, categoryID); err != nil {
		return fmt.Errorf("limpiar vínculos de categoría: %w", err)
	}
	for _, sid := range serviceIDs {
		if sid == 0 {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO category_service_links (category_id, service_id)
			VALUES (?, ?)
		`, categoryID, sid); err != nil {
			return fmt.Errorf("vincular servicio %d a categoría: %w", sid, err)
		}
	}
	return nil
}

// attachServices carga los servicios vinculados (ids y nombres) para un set de
// categorías en una sola query agrupada.
func (s *CategoryStorage) attachServices(ctx context.Context, cats []models.Category) ([]models.Category, error) {
	if len(cats) == 0 {
		return cats, nil
	}
	ids := make([]int64, 0, len(cats))
	for _, c := range cats {
		ids = append(ids, c.ID)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT l.category_id, s.id, s.name
		FROM category_service_links l
		JOIN services s ON s.id = l.service_id
		WHERE l.category_id IN (`+inClause(len(ids))+`)
		ORDER BY l.category_id, s.name
	`, toAnySlice(ids)...)
	if err != nil {
		return nil, fmt.Errorf("cargar servicios de categorías: %w", err)
	}
	defer rows.Close()

	links := make(map[int64][]int64)
	names := make(map[int64][]string)
	for rows.Next() {
		var categoryID, serviceID int64
		var name string
		if err := rows.Scan(&categoryID, &serviceID, &name); err != nil {
			return nil, fmt.Errorf("escanear servicio de categoría: %w", err)
		}
		links[categoryID] = append(links[categoryID], serviceID)
		names[categoryID] = append(names[categoryID], name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range cats {
		cats[i].ServiceIDs = links[cats[i].ID]
		cats[i].ServiceNames = names[cats[i].ID]
	}
	return cats, nil
}

func inClause(n int) string {
	placeholders := make([]byte, 0, n*2)
	for i := 0; i < n; i++ {
		if i > 0 {
			placeholders = append(placeholders, ',')
		}
		placeholders = append(placeholders, '?')
	}
	return string(placeholders)
}

func toAnySlice(ids []int64) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

func scanCategory(row *sql.Row) (*models.Category, error) {
	var c models.Category
	var targetAmount sql.NullFloat64
	var targetType, targetDate sql.NullString
	var accountID sql.NullInt64
	var deletedAt sql.NullTime
	if err := row.Scan(&c.ID, &c.CategoryGroupID, &c.Name, &c.Icon, &c.SortOrder,
		&targetAmount, &targetType, &targetDate,
		&accountID,
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
	if accountID.Valid {
		aid := accountID.Int64
		c.AccountID = &aid
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
		var accountID sql.NullInt64
		var deletedAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.CategoryGroupID, &c.Name, &c.Icon, &c.SortOrder,
			&targetAmount, &targetType, &targetDate,
			&accountID,
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
		if accountID.Valid {
			aid := accountID.Int64
			c.AccountID = &aid
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
