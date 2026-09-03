package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// ChildSupportConfigStorage encapsula el acceso a la tabla child_support_configs.
type ChildSupportConfigStorage struct {
	db *sql.DB
}

// NewChildSupportConfigStorage crea un nuevo ChildSupportConfigStorage.
func NewChildSupportConfigStorage(db *sql.DB) *ChildSupportConfigStorage {
	return &ChildSupportConfigStorage{db: db}
}

const childSupportConfigSelectCols = `
		c.id, c.child_id, ch.first_name, ch.last_name,
		c.pension_category_id, pc.name,
		c.amount, c.currency, c.is_active, c.auto_generate, c.created_at, c.updated_at`

// List devuelve todas las configs con nombres resueltos.
func (s *ChildSupportConfigStorage) List(ctx context.Context) ([]models.ChildSupportConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT`+childSupportConfigSelectCols+`
		FROM child_support_configs c
		JOIN children ch ON ch.id = c.child_id
		JOIN pension_categories pc ON pc.id = c.pension_category_id
		ORDER BY c.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listar configs de pensión: %w", err)
	}
	defer rows.Close()
	return scanChildSupportConfigs(rows)
}

// ListByChild devuelve las configs de un hijo.
func (s *ChildSupportConfigStorage) ListByChild(ctx context.Context, childID int64) ([]models.ChildSupportConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT`+childSupportConfigSelectCols+`
		FROM child_support_configs c
		JOIN children ch ON ch.id = c.child_id
		JOIN pension_categories pc ON pc.id = c.pension_category_id
		WHERE c.child_id = ?
		ORDER BY c.created_at DESC
	`, childID)
	if err != nil {
		return nil, fmt.Errorf("listar configs de pensión: %w", err)
	}
	defer rows.Close()
	return scanChildSupportConfigs(rows)
}

// ListActiveAutoGenerate devuelve las configs activas con auto_generate=1.
func (s *ChildSupportConfigStorage) ListActiveAutoGenerate(ctx context.Context) ([]models.ChildSupportConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT`+childSupportConfigSelectCols+`
		FROM child_support_configs c
		JOIN children ch ON ch.id = c.child_id
		JOIN pension_categories pc ON pc.id = c.pension_category_id
		WHERE c.is_active = 1 AND c.auto_generate = 1
		ORDER BY c.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listar configs activas de pensión: %w", err)
	}
	defer rows.Close()
	return scanChildSupportConfigs(rows)
}

// GetByID busca una config por ID.
func (s *ChildSupportConfigStorage) GetByID(ctx context.Context, id int64) (*models.ChildSupportConfig, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT`+childSupportConfigSelectCols+`
		FROM child_support_configs c
		JOIN children ch ON ch.id = c.child_id
		JOIN pension_categories pc ON pc.id = c.pension_category_id
		WHERE c.id = ?
	`, id)
	return scanChildSupportConfig(row)
}

// Exists verifica si ya existe una config para hijo+categoría.
func (s *ChildSupportConfigStorage) Exists(ctx context.Context, childID, categoryID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM child_support_configs
		WHERE child_id = ? AND pension_category_id = ?
	`, childID, categoryID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("verificar config existente: %w", err)
	}
	return count > 0, nil
}

// Create inserta una nueva config.
func (s *ChildSupportConfigStorage) Create(ctx context.Context, data *models.ChildSupportConfig) (*models.ChildSupportConfig, error) {
	active, autoGen := 0, 0
	if data.IsActive {
		active = 1
	}
	if data.AutoGenerate {
		autoGen = 1
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO child_support_configs (child_id, pension_category_id, amount, currency, is_active, auto_generate)
		VALUES (?, ?, ?, ?, ?, ?)
	`, data.ChildID, data.PensionCategoryID, data.Amount, data.Currency, active, autoGen)
	if err != nil {
		return nil, fmt.Errorf("insertar config de pensión: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de config: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza una config existente.
func (s *ChildSupportConfigStorage) Update(ctx context.Context, id int64, data *models.ChildSupportConfig) (*models.ChildSupportConfig, error) {
	active, autoGen := 0, 0
	if data.IsActive {
		active = 1
	}
	if data.AutoGenerate {
		autoGen = 1
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE child_support_configs
		SET pension_category_id = ?, amount = ?, currency = ?, is_active = ?, auto_generate = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, data.PensionCategoryID, data.Amount, data.Currency, active, autoGen, id)
	if err != nil {
		return nil, fmt.Errorf("actualizar config de pensión: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Delete elimina una config.
func (s *ChildSupportConfigStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM child_support_configs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("eliminar config de pensión: %w", err)
	}
	return nil
}

func scanChildSupportConfig(row *sql.Row) (*models.ChildSupportConfig, error) {
	var c models.ChildSupportConfig
	var firstName, lastName string
	if err := row.Scan(&c.ID, &c.ChildID, &firstName, &lastName,
		&c.PensionCategoryID, &c.CategoryName,
		&c.Amount, &c.Currency, &c.IsActive, &c.AutoGenerate, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear config de pensión: %w", err)
	}
	c.ChildName = firstName + " " + lastName
	return &c, nil
}

func scanChildSupportConfigs(rows *sql.Rows) ([]models.ChildSupportConfig, error) {
	var configs []models.ChildSupportConfig
	for rows.Next() {
		var c models.ChildSupportConfig
		var firstName, lastName string
		if err := rows.Scan(&c.ID, &c.ChildID, &firstName, &lastName,
			&c.PensionCategoryID, &c.CategoryName,
			&c.Amount, &c.Currency, &c.IsActive, &c.AutoGenerate, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear config de pensión: %w", err)
		}
		c.ChildName = firstName + " " + lastName
		configs = append(configs, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return configs, nil
}
