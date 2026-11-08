package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// AutoMaintenanceStorage encapsula el acceso a auto_maintenances (SPEC-124).
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
			m.workshop_name, m.description, m.labor_cost, m.supplies_cost,
			m.currency_id, COALESCE(i.name, ''), COALESCE(c.code, ''),
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
			m.workshop_name, m.description, m.labor_cost, m.supplies_cost,
			m.currency_id, COALESCE(i.name, ''), COALESCE(c.code, ''),
			m.created_at, m.updated_at
		FROM auto_maintenances m
		LEFT JOIN institutions i ON i.id = m.institution_id
		LEFT JOIN currencies c ON c.id = m.currency_id
		WHERE m.auto_id = ? AND m.id = ?
	`, autoID, id)

	item, err := scanAutoMaintenanceRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	item.Attachments = []models.AutoAttachment{}
	return item, nil
}

// Create inserta un mantenimiento.
func (s *AutoMaintenanceStorage) Create(
	ctx context.Context,
	autoID int64,
	performedAt string,
	mileage int64,
	institutionID *int64,
	workshopName, description string,
	laborCost, suppliesCost float64,
	currencyID int64,
) (*models.AutoMaintenance, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO auto_maintenances
			(auto_id, performed_at, mileage, institution_id, workshop_name, description, labor_cost, supplies_cost, currency_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, autoID, performedAt, mileage, institutionID, workshopName, description, laborCost, suppliesCost, currencyID)
	if err != nil {
		return nil, fmt.Errorf("crear mantenimiento: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de mantenimiento: %w", err)
	}
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

// Update actualiza un mantenimiento y devuelve si existía.
func (s *AutoMaintenanceStorage) Update(
	ctx context.Context,
	autoID, id int64,
	performedAt string,
	mileage int64,
	institutionID *int64,
	workshopName, description string,
	laborCost, suppliesCost float64,
	currencyID int64,
) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE auto_maintenances
		SET performed_at = ?, mileage = ?, institution_id = ?, workshop_name = ?,
			description = ?, labor_cost = ?, supplies_cost = ?, currency_id = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE auto_id = ? AND id = ?
	`, performedAt, mileage, institutionID, workshopName, description, laborCost, suppliesCost, currencyID, autoID, id)
	if err != nil {
		return false, fmt.Errorf("actualizar mantenimiento: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("verificar actualización de mantenimiento: %w", err)
	}
	return affected > 0, nil
}

// Delete elimina un mantenimiento y devuelve si existía.
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

// TotalsByCurrency devuelve la suma de mano de obra e insumos por moneda.
func (s *AutoMaintenanceStorage) TotalsByCurrency(ctx context.Context, autoID int64) ([]models.AutoMaintenanceCurrencyTotal, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.currency_id, COALESCE(c.code, ''), COALESCE(SUM(m.labor_cost + m.supplies_cost), 0)
		FROM auto_maintenances m
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

type autoMaintenanceScanner interface {
	Scan(dest ...any) error
}

func scanAutoMaintenance(scanner autoMaintenanceScanner) (models.AutoMaintenanceDetail, error) {
	var item models.AutoMaintenanceDetail
	var institutionID sql.NullInt64
	if err := scanner.Scan(
		&item.ID, &item.AutoID, &item.PerformedAt, &item.Mileage, &institutionID,
		&item.WorkshopName, &item.Description, &item.LaborCost, &item.SuppliesCost,
		&item.CurrencyID, &item.InstitutionName, &item.CurrencyCode,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return item, fmt.Errorf("escanear mantenimiento: %w", err)
	}
	if institutionID.Valid {
		item.InstitutionID = &institutionID.Int64
	}
	item.TotalCost = item.LaborCost + item.SuppliesCost
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
