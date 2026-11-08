package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// AutoInspectionStorage encapsula el acceso a auto_inspections (SPEC-124).
type AutoInspectionStorage struct {
	db *sql.DB
}

// NewAutoInspectionStorage crea un nuevo AutoInspectionStorage.
func NewAutoInspectionStorage(db *sql.DB) *AutoInspectionStorage {
	return &AutoInspectionStorage{db: db}
}

// ListByAuto devuelve las inspecciones de un auto con el nombre de la institución.
func (s *AutoInspectionStorage) ListByAuto(ctx context.Context, autoID int64) ([]models.AutoInspectionDetail, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			i.id, i.auto_id, i.performed_at, i.inspection_year, i.institution_id,
			i.result, i.notes, COALESCE(inst.name, ''), i.created_at, i.updated_at
		FROM auto_inspections i
		LEFT JOIN institutions inst ON inst.id = i.institution_id
		WHERE i.auto_id = ?
		ORDER BY i.inspection_year DESC, i.performed_at DESC, i.id DESC
	`, autoID)
	if err != nil {
		return nil, fmt.Errorf("listar inspecciones: %w", err)
	}
	defer rows.Close()

	items := make([]models.AutoInspectionDetail, 0)
	for rows.Next() {
		item, err := scanAutoInspection(rows)
		if err != nil {
			return nil, err
		}
		item.Attachments = []models.AutoAttachment{}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetByID busca una inspección verificando que pertenezca al auto.
func (s *AutoInspectionStorage) GetByID(ctx context.Context, autoID, id int64) (*models.AutoInspectionDetail, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			i.id, i.auto_id, i.performed_at, i.inspection_year, i.institution_id,
			i.result, i.notes, COALESCE(inst.name, ''), i.created_at, i.updated_at
		FROM auto_inspections i
		LEFT JOIN institutions inst ON inst.id = i.institution_id
		WHERE i.auto_id = ? AND i.id = ?
	`, autoID, id)

	item, err := scanAutoInspectionRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	item.Attachments = []models.AutoAttachment{}
	return item, nil
}

// Create inserta una inspección.
func (s *AutoInspectionStorage) Create(
	ctx context.Context,
	autoID int64,
	performedAt string,
	inspectionYear, institutionID int64,
	result, notes string,
) (*models.AutoInspection, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO auto_inspections
			(auto_id, performed_at, inspection_year, institution_id, result, notes)
		VALUES (?, ?, ?, ?, ?, ?)
	`, autoID, performedAt, inspectionYear, institutionID, result, notes)
	if err != nil {
		return nil, fmt.Errorf("crear inspección: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de inspección: %w", err)
	}
	item, err := s.GetByID(ctx, autoID, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("inspección no encontrada después de crear")
	}
	base := item.AutoInspection
	return &base, nil
}

// Update actualiza una inspección y devuelve si existía.
func (s *AutoInspectionStorage) Update(
	ctx context.Context,
	autoID, id int64,
	performedAt string,
	inspectionYear, institutionID int64,
	result, notes string,
) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE auto_inspections
		SET performed_at = ?, inspection_year = ?, institution_id = ?, result = ?,
			notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE auto_id = ? AND id = ?
	`, performedAt, inspectionYear, institutionID, result, notes, autoID, id)
	if err != nil {
		return false, fmt.Errorf("actualizar inspección: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("verificar actualización de inspección: %w", err)
	}
	return affected > 0, nil
}

// Delete elimina una inspección y devuelve si existía.
func (s *AutoInspectionStorage) Delete(ctx context.Context, autoID, id int64) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM auto_inspections WHERE auto_id = ? AND id = ?
	`, autoID, id)
	if err != nil {
		return false, fmt.Errorf("eliminar inspección: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("verificar eliminación de inspección: %w", err)
	}
	return affected > 0, nil
}

// YearsByResult devuelve los años de inspección distintos para un resultado.
func (s *AutoInspectionStorage) YearsByResult(ctx context.Context, autoID int64, result string) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT inspection_year
		FROM auto_inspections
		WHERE auto_id = ? AND result = ?
		ORDER BY inspection_year
	`, autoID, result)
	if err != nil {
		return nil, fmt.Errorf("listar años de inspección: %w", err)
	}
	defer rows.Close()

	years := make([]int64, 0)
	for rows.Next() {
		var year int64
		if err := rows.Scan(&year); err != nil {
			return nil, fmt.Errorf("escanear año de inspección: %w", err)
		}
		years = append(years, year)
	}
	return years, rows.Err()
}

// ApprovedYearsByAuto devuelve un mapa auto_id -> años aprobados.
func (s *AutoInspectionStorage) ApprovedYearsByAuto(ctx context.Context) (map[int64][]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT auto_id, inspection_year
		FROM auto_inspections
		WHERE result = 'approved'
		ORDER BY auto_id, inspection_year
	`)
	if err != nil {
		return nil, fmt.Errorf("listar años aprobados por auto: %w", err)
	}
	defer rows.Close()

	years := make(map[int64][]int64)
	for rows.Next() {
		var autoID, year int64
		if err := rows.Scan(&autoID, &year); err != nil {
			return nil, fmt.Errorf("escanear año aprobado: %w", err)
		}
		years[autoID] = append(years[autoID], year)
	}
	return years, rows.Err()
}

type autoInspectionScanner interface {
	Scan(dest ...any) error
}

func scanAutoInspection(scanner autoInspectionScanner) (models.AutoInspectionDetail, error) {
	var item models.AutoInspectionDetail
	if err := scanner.Scan(
		&item.ID, &item.AutoID, &item.PerformedAt, &item.InspectionYear, &item.InstitutionID,
		&item.Result, &item.Notes, &item.InstitutionName, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return item, fmt.Errorf("escanear inspección: %w", err)
	}
	return item, nil
}

func scanAutoInspectionRow(row *sql.Row) (*models.AutoInspectionDetail, error) {
	item, err := scanAutoInspection(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &item, nil
}
