package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

type ServiceStorage struct {
	db *sql.DB
}

func NewServiceStorage(db *sql.DB) *ServiceStorage {
	return &ServiceStorage{db: db}
}

const serviceColumns = `
	id, home_id, name, institution, currency_id, frequency,
	suggested_amount, active, icon_key, billing_type, billing_day, auto_generate,
	institution_id, institution_analyzer_id, deleted_at, created_at, updated_at
`

func (s *ServiceStorage) List(ctx context.Context, homeID *int64) ([]models.Service, error) {
	query := "SELECT " + serviceColumns + " FROM services WHERE deleted_at IS NULL"
	var args []any
	if homeID != nil {
		query += " AND home_id = ?"
		args = append(args, *homeID)
	}
	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listar servicios: %w", err)
	}
	defer rows.Close()

	return scanServices(rows)
}

func (s *ServiceStorage) GetByID(ctx context.Context, id int64) (*models.Service, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT "+serviceColumns+" FROM services WHERE id = ? AND deleted_at IS NULL", id,
	)
	return scanService(row)
}

func (s *ServiceStorage) Count(ctx context.Context, homeID *int64) (int64, error) {
	query := "SELECT COUNT(*) FROM services WHERE deleted_at IS NULL"
	var args []any
	if homeID != nil {
		query += " AND home_id = ?"
		args = append(args, *homeID)
	}

	var count int64
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("contar servicios: %w", err)
	}
	return count, nil
}

func (s *ServiceStorage) Create(ctx context.Context, svc *models.Service) (*models.Service, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, billing_day, auto_generate, institution_id, institution_analyzer_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, svc.HomeID, svc.Name, svc.Institution, svc.CurrencyID, svc.Frequency, svc.SuggestedAmount, svc.Active, svc.IconKey, svc.BillingType, svc.BillingDay, svc.AutoGenerate, svc.InstitutionID, svc.InstitutionAnalyzerID)
	if err != nil {
		return nil, fmt.Errorf("insertar servicio: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de servicio: %w", err)
	}
	return s.GetByID(ctx, id)
}

func (s *ServiceStorage) Update(ctx context.Context, svc *models.Service) (*models.Service, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE services
		SET home_id = ?, name = ?, institution = ?, currency_id = ?, frequency = ?,
		    suggested_amount = ?, active = ?, icon_key = ?, billing_type = ?, billing_day = ?, auto_generate = ?, institution_id = ?, institution_analyzer_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, svc.HomeID, svc.Name, svc.Institution, svc.CurrencyID, svc.Frequency,
		svc.SuggestedAmount, svc.Active, svc.IconKey, svc.BillingType, svc.BillingDay, svc.AutoGenerate, svc.InstitutionID, svc.InstitutionAnalyzerID, svc.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar servicio: %w", err)
	}
	return s.GetByID(ctx, svc.ID)
}

func (s *ServiceStorage) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE services SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar servicio: %w", err)
	}
	return nil
}

func scanService(row *sql.Row) (*models.Service, error) {
	var svc models.Service
	var deletedAt sql.NullTime
	var billingDay sql.NullInt64
	var institutionID, institutionAnalyzerID sql.NullInt64
	if err := row.Scan(&svc.ID, &svc.HomeID, &svc.Name, &svc.Institution, &svc.CurrencyID,
		&svc.Frequency, &svc.SuggestedAmount, &svc.Active, &svc.IconKey, &svc.BillingType, &billingDay, &svc.AutoGenerate,
		&institutionID, &institutionAnalyzerID, &deletedAt, &svc.CreatedAt, &svc.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear servicio: %w", err)
	}
	if billingDay.Valid {
		v := int(billingDay.Int64)
		svc.BillingDay = &v
	}
	if deletedAt.Valid {
		svc.DeletedAt = &deletedAt.Time
	}
	if institutionID.Valid {
		svc.InstitutionID = &institutionID.Int64
	}
	if institutionAnalyzerID.Valid {
		svc.InstitutionAnalyzerID = &institutionAnalyzerID.Int64
	}
	return &svc, nil
}

func scanServices(rows *sql.Rows) ([]models.Service, error) {
	var services []models.Service
	for rows.Next() {
		var svc models.Service
		var deletedAt sql.NullTime
		var billingDay sql.NullInt64
		var institutionID, institutionAnalyzerID sql.NullInt64
		if err := rows.Scan(&svc.ID, &svc.HomeID, &svc.Name, &svc.Institution, &svc.CurrencyID,
			&svc.Frequency, &svc.SuggestedAmount, &svc.Active, &svc.IconKey, &svc.BillingType, &billingDay, &svc.AutoGenerate,
			&institutionID, &institutionAnalyzerID, &deletedAt, &svc.CreatedAt, &svc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear servicio: %w", err)
		}
		if billingDay.Valid {
			v := int(billingDay.Int64)
			svc.BillingDay = &v
		}
		if deletedAt.Valid {
			svc.DeletedAt = &deletedAt.Time
		}
		if institutionID.Valid {
			svc.InstitutionID = &institutionID.Int64
		}
		if institutionAnalyzerID.Valid {
			svc.InstitutionAnalyzerID = &institutionAnalyzerID.Int64
		}
		services = append(services, svc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return services, nil
}
