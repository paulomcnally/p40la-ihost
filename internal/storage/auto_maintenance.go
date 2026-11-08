package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// AutoMaintenanceStorage encapsula el acceso a auto_maintenances y sus líneas
// de detalle (SPEC-124).
type AutoMaintenanceStorage struct {
	db *sql.DB
}

// NewAutoMaintenanceStorage crea un nuevo AutoMaintenanceStorage.
func NewAutoMaintenanceStorage(db *sql.DB) *AutoMaintenanceStorage {
	return &AutoMaintenanceStorage{db: db}
}

// ListByAuto devuelve los mantenimientos de un auto con institución y moneda.
func (s *AutoMaintenanceStorage) ListByAuto(ctx context.Context, autoID int64) ([]models.AutoMaintenanceDetail, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			m.id, m.auto_id, m.performed_at, m.mileage, m.institution_id,
			m.description, m.currency_id, COALESCE(i.name, ''), COALESCE(c.code, ''),
			m.created_at, m.updated_at
		FROM auto_maintenances m
		LEFT JOIN institutions i ON i.id = m.institution_id
		LEFT JOIN currencies c ON c.id = m.currency_id
		WHERE m.auto_id = ?
		ORDER BY m.performed_at DESC, m.id DESC
	`, autoID)
	if err != nil {
		return nil, fmt.Errorf("listar mantenimientos: %w", err)
	}
	defer rows.Close()

	items := make([]models.AutoMaintenanceDetail, 0)
	for rows.Next() {
		item, err := scanAutoMaintenance(rows)
		if err != nil {
			return nil, err
		}
		item.Items = []models.AutoMaintenanceItem{}
		item.Attachments = []models.AutoAttachment{}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetByID busca un mantenimiento verificando que pertenezca al auto.
func (s *AutoMaintenanceStorage) GetByID(ctx context.Context, autoID, id int64) (*models.AutoMaintenanceDetail, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			m.id, m.auto_id, m.performed_at, m.mileage, m.institution_id,
			m.description, m.currency_id, COALESCE(i.name, ''), COALESCE(c.code, ''),
			m.created_at, m.updated_at
		FROM auto_maintenances m
		LEFT JOIN institutions i ON i.id = m.institution_id
		LEFT JOIN currencies c ON c.id = m.currency_id
		WHERE m.auto_id = ? AND m.id = ?
	`, autoID, id)

	item, err := scanAutoMaintenanceRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	item.Items = []models.AutoMaintenanceItem{}
	item.Attachments = []models.AutoAttachment{}
	return item, nil
}

// Create inserta un mantenimiento (solo información básica).
func (s *AutoMaintenanceStorage) Create(
	ctx context.Context,
	autoID int64,
	performedAt string,
	mileage int64,
	institutionID int64,
	description string,
	currencyID int64,
) (*models.AutoMaintenance, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO auto_maintenances
			(auto_id, performed_at, mileage, institution_id, description, currency_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, autoID, performedAt, mileage, institutionID, description, currencyID)
	if err != nil {
		return nil, fmt.Errorf("crear mantenimiento: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de mantenimiento: %w", err)
	}
	return s.getBase(ctx, autoID, id)
}

// Update actualiza la información básica de un mantenimiento.
func (s *AutoMaintenanceStorage) Update(
	ctx context.Context,
	autoID, id int64,
	performedAt string,
	mileage int64,
	institutionID int64,
	description string,
	currencyID int64,
) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE auto_maintenances
		SET performed_at = ?, mileage = ?, institution_id = ?, description = ?,
			currency_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE auto_id = ? AND id = ?
	`, performedAt, mileage, institutionID, description, currencyID, autoID, id)
	if err != nil {
		return false, fmt.Errorf("actualizar mantenimiento: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("verificar actualización de mantenimiento: %w", err)
	}
	return affected > 0, nil
}

// Delete elimina un mantenimiento y devuelve si existía. Las líneas se
// eliminan por ON DELETE CASCADE.
func (s *AutoMaintenanceStorage) Delete(ctx context.Context, autoID, id int64) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM auto_maintenances WHERE auto_id = ? AND id = ?
	`, autoID, id)
	if err != nil {
		return false, fmt.Errorf("eliminar mantenimiento: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("verificar eliminación de mantenimiento: %w", err)
	}
	return affected > 0, nil
}

// LatestMileage devuelve el kilometraje más alto registrado para un auto.
func (s *AutoMaintenanceStorage) LatestMileage(ctx context.Context, autoID int64) (int64, error) {
	var mileage int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(mileage), 0) FROM auto_maintenances WHERE auto_id = ?
	`, autoID).Scan(&mileage); err != nil {
		return 0, fmt.Errorf("obtener último kilometraje: %w", err)
	}
	return mileage, nil
}

// TotalsByCurrency devuelve la suma de las líneas por moneda del mantenimiento.
func (s *AutoMaintenanceStorage) TotalsByCurrency(ctx context.Context, autoID int64) ([]models.AutoMaintenanceCurrencyTotal, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.currency_id, COALESCE(c.code, ''), COALESCE(SUM(i.amount), 0)
		FROM auto_maintenances m
		JOIN auto_maintenance_items i ON i.maintenance_id = m.id
		LEFT JOIN currencies c ON c.id = m.currency_id
		WHERE m.auto_id = ?
		GROUP BY m.currency_id, c.code
		ORDER BY c.code
	`, autoID)
	if err != nil {
		return nil, fmt.Errorf("sumar mantenimientos por moneda: %w", err)
	}
	defer rows.Close()

	totals := make([]models.AutoMaintenanceCurrencyTotal, 0)
	for rows.Next() {
		var total models.AutoMaintenanceCurrencyTotal
		if err := rows.Scan(&total.CurrencyID, &total.CurrencyCode, &total.Total); err != nil {
			return nil, fmt.Errorf("escanear total de mantenimientos: %w", err)
		}
		totals = append(totals, total)
	}
	return totals, rows.Err()
}

// ListItemsByAuto devuelve las líneas de todos los mantenimientos de un auto,
// agrupadas por mantenimiento.
func (s *AutoMaintenanceStorage) ListItemsByAuto(ctx context.Context, autoID int64) (map[int64][]models.AutoMaintenanceItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT i.id, i.maintenance_id, i.item_type, i.description, i.amount, i.created_at, i.updated_at
		FROM auto_maintenance_items i
		JOIN auto_maintenances m ON m.id = i.maintenance_id
		WHERE m.auto_id = ?
		ORDER BY i.maintenance_id, i.id
	`, autoID)
	if err != nil {
		return nil, fmt.Errorf("listar conceptos de mantenimientos: %w", err)
	}
	defer rows.Close()

	grouped := make(map[int64][]models.AutoMaintenanceItem)
	for rows.Next() {
		item, err := scanAutoMaintenanceItem(rows)
		if err != nil {
			return nil, err
		}
		grouped[item.MaintenanceID] = append(grouped[item.MaintenanceID], item)
	}
	return grouped, rows.Err()
}

// ListItemsByMaintenance devuelve las líneas de un mantenimiento.
func (s *AutoMaintenanceStorage) ListItemsByMaintenance(ctx context.Context, maintenanceID int64) ([]models.AutoMaintenanceItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, maintenance_id, item_type, description, amount, created_at, updated_at
		FROM auto_maintenance_items
		WHERE maintenance_id = ?
		ORDER BY id
	`, maintenanceID)
	if err != nil {
		return nil, fmt.Errorf("listar conceptos del mantenimiento: %w", err)
	}
	defer rows.Close()

	items := make([]models.AutoMaintenanceItem, 0)
	for rows.Next() {
		item, err := scanAutoMaintenanceItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *AutoMaintenanceStorage) getBase(ctx context.Context, autoID, id int64) (*models.AutoMaintenance, error) {
	item, err := s.GetByID(ctx, autoID, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("mantenimiento no encontrado después de crear")
	}
	base := item.AutoMaintenance
	return &base, nil
}

// CreateItem inserta una línea de detalle en un mantenimiento.
func (s *AutoMaintenanceStorage) CreateItem(ctx context.Context, maintenanceID int64, input models.AutoMaintenanceItemInput) (*models.AutoMaintenanceItem, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO auto_maintenance_items (maintenance_id, item_type, description, amount)
		VALUES (?, ?, ?, ?)
	`, maintenanceID, input.ItemType, input.Description, input.Amount)
	if err != nil {
		return nil, fmt.Errorf("crear concepto del mantenimiento: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de concepto: %w", err)
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT id, maintenance_id, item_type, description, amount, created_at, updated_at
		FROM auto_maintenance_items
		WHERE id = ? AND maintenance_id = ?
	`, id, maintenanceID)
	return scanAutoMaintenanceItemRow(row)
}

// UpdateItem actualiza una línea de detalle y devuelve si existía.
func (s *AutoMaintenanceStorage) UpdateItem(ctx context.Context, maintenanceID, itemID int64, input models.AutoMaintenanceItemInput) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE auto_maintenance_items
		SET item_type = ?, description = ?, amount = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND maintenance_id = ?
	`, input.ItemType, input.Description, input.Amount, itemID, maintenanceID)
	if err != nil {
		return false, fmt.Errorf("actualizar concepto del mantenimiento: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("verificar actualización del concepto: %w", err)
	}
	return affected > 0, nil
}

// DeleteItem elimina una línea de detalle y devuelve si existía.
func (s *AutoMaintenanceStorage) DeleteItem(ctx context.Context, maintenanceID, itemID int64) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM auto_maintenance_items
		WHERE id = ? AND maintenance_id = ?
	`, itemID, maintenanceID)
	if err != nil {
		return false, fmt.Errorf("eliminar concepto del mantenimiento: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("verificar eliminación del concepto: %w", err)
	}
	return affected > 0, nil
}

type autoMaintenanceScanner interface {
	Scan(dest ...any) error
}

func scanAutoMaintenance(scanner autoMaintenanceScanner) (models.AutoMaintenanceDetail, error) {
	var item models.AutoMaintenanceDetail
	if err := scanner.Scan(
		&item.ID, &item.AutoID, &item.PerformedAt, &item.Mileage, &item.InstitutionID,
		&item.Description, &item.CurrencyID, &item.InstitutionName, &item.CurrencyCode,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return item, fmt.Errorf("escanear mantenimiento: %w", err)
	}
	return item, nil
}

func scanAutoMaintenanceRow(row *sql.Row) (*models.AutoMaintenanceDetail, error) {
	item, err := scanAutoMaintenance(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &item, nil
}

func scanAutoMaintenanceItem(scanner autoMaintenanceScanner) (models.AutoMaintenanceItem, error) {
	var item models.AutoMaintenanceItem
	if err := scanner.Scan(
		&item.ID, &item.MaintenanceID, &item.ItemType, &item.Description,
		&item.Amount, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return item, fmt.Errorf("escanear concepto de mantenimiento: %w", err)
	}
	return item, nil
}

func scanAutoMaintenanceItemRow(row *sql.Row) (*models.AutoMaintenanceItem, error) {
	item, err := scanAutoMaintenanceItem(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &item, nil
}
