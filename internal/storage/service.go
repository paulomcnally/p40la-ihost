package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
	institution_id, institution_analyzer_id,
	start_date, end_date, is_recurring, webhook_uuid,
	last_webhook_request,
	(SELECT b.status FROM bills b WHERE b.service_id = services.id AND (b.amount > 0 OR b.invoice_number != '') ORDER BY b.year DESC, b.month DESC, b.id DESC LIMIT 1) AS latest_bill_status,
	EXISTS (
		SELECT 1 FROM institutions i JOIN institution_categories ic ON ic.id = i.category_id
		WHERE i.id = services.institution_id AND ic.key = 'insurance'
	) AS is_insurance,
	deleted_at, created_at, updated_at
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

// FindByWebhookUUID busca un servicio activo por su webhook_uuid (SPEC-069).
func (s *ServiceStorage) FindByWebhookUUID(ctx context.Context, uuid string) (*models.Service, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT "+serviceColumns+" FROM services WHERE webhook_uuid = ? AND deleted_at IS NULL", uuid,
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
		INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, billing_day, auto_generate, institution_id, institution_analyzer_id, start_date, end_date, is_recurring, webhook_uuid)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, svc.HomeID, svc.Name, svc.Institution, svc.CurrencyID, svc.Frequency, svc.SuggestedAmount, svc.Active, svc.IconKey, svc.BillingType, svc.BillingDay, svc.AutoGenerate, svc.InstitutionID, svc.InstitutionAnalyzerID, svc.StartDate, svc.EndDate, svc.IsRecurring, svc.WebhookUUID)
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
		    suggested_amount = ?, active = ?, icon_key = ?, billing_type = ?, billing_day = ?, auto_generate = ?,
		    institution_id = ?, institution_analyzer_id = ?, start_date = ?, end_date = ?, is_recurring = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, svc.HomeID, svc.Name, svc.Institution, svc.CurrencyID, svc.Frequency,
		svc.SuggestedAmount, svc.Active, svc.IconKey, svc.BillingType, svc.BillingDay, svc.AutoGenerate,
		svc.InstitutionID, svc.InstitutionAnalyzerID, svc.StartDate, svc.EndDate, svc.IsRecurring, svc.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar servicio: %w", err)
	}
	return s.GetByID(ctx, svc.ID)
}

// SetWebhookUUID asigna (o reemplaza) el webhook_uuid de un servicio (SPEC-069).
func (s *ServiceStorage) SetWebhookUUID(ctx context.Context, id int64, uuid string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE services SET webhook_uuid = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, uuid, id)
	if err != nil {
		return fmt.Errorf("asignar webhook_uuid a servicio: %w", err)
	}
	return nil
}

// SetLastWebhookRequest registra la fecha del último request de webhook
// recibido por el servicio (SPEC-074).
func (s *ServiceStorage) SetLastWebhookRequest(ctx context.Context, id int64, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE services SET last_webhook_request = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, at.UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("registrar último request de webhook: %w", err)
	}
	return nil
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
	var latestBillStatus sql.NullString
	var startDate, endDate sql.NullString
	var institution sql.NullString
	var webhookUUID sql.NullString
	var lastWebhookRequest sql.NullTime
	if err := row.Scan(&svc.ID, &svc.HomeID, &svc.Name, &institution, &svc.CurrencyID,
		&svc.Frequency, &svc.SuggestedAmount, &svc.Active, &svc.IconKey, &svc.BillingType, &billingDay, &svc.AutoGenerate,
		&institutionID, &institutionAnalyzerID, &startDate, &endDate, &svc.IsRecurring, &webhookUUID,
		&lastWebhookRequest, &latestBillStatus, &svc.IsInsurance, &deletedAt, &svc.CreatedAt, &svc.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear servicio: %w", err)
	}
	svc.Institution = institution.String
	svc.WebhookUUID = webhookUUID.String
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
	if startDate.Valid {
		svc.StartDate = &startDate.String
	}
	if endDate.Valid {
		svc.EndDate = &endDate.String
	}
	if latestBillStatus.Valid {
		svc.LatestBillStatus = &latestBillStatus.String
	}
	if lastWebhookRequest.Valid {
		svc.LastWebhookRequest = &lastWebhookRequest.Time
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
		var latestBillStatus sql.NullString
		var startDate, endDate sql.NullString
		var institution sql.NullString
		var webhookUUID sql.NullString
		var lastWebhookRequest sql.NullTime
		if err := rows.Scan(&svc.ID, &svc.HomeID, &svc.Name, &institution, &svc.CurrencyID,
			&svc.Frequency, &svc.SuggestedAmount, &svc.Active, &svc.IconKey, &svc.BillingType, &billingDay, &svc.AutoGenerate,
			&institutionID, &institutionAnalyzerID, &startDate, &endDate, &svc.IsRecurring, &webhookUUID,
			&lastWebhookRequest, &latestBillStatus, &svc.IsInsurance, &deletedAt, &svc.CreatedAt, &svc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear servicio: %w", err)
		}
		svc.Institution = institution.String
		svc.WebhookUUID = webhookUUID.String
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
		if startDate.Valid {
			svc.StartDate = &startDate.String
		}
		if endDate.Valid {
			svc.EndDate = &endDate.String
		}
		if latestBillStatus.Valid {
			svc.LatestBillStatus = &latestBillStatus.String
		}
		if lastWebhookRequest.Valid {
			svc.LastWebhookRequest = &lastWebhookRequest.Time
		}
		services = append(services, svc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return services, nil
}
