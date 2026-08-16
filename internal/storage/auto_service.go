package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// AutoServiceStorage encapsula el acceso a la tabla auto_services.
type AutoServiceStorage struct {
	db *sql.DB
}

// NewAutoServiceStorage crea un nuevo AutoServiceStorage.
func NewAutoServiceStorage(db *sql.DB) *AutoServiceStorage {
	return &AutoServiceStorage{db: db}
}

// ListByAuto devuelve todos los seguros asociados a un auto, con info de servicio e institución.
func (s *AutoServiceStorage) ListByAuto(ctx context.Context, autoID int64) ([]models.AutoServiceDetail, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			asv.id, asv.auto_id, asv.service_id, asv.coverage_type,
			asv.policy_number, asv.certificate, asv.insurer_number,
			svc.name, COALESCE(inst.name, ''), svc.institution_id,
			svc.suggested_amount, svc.frequency, svc.icon_key, svc.active,
			svc.start_date, svc.end_date, svc.is_recurring,
			asv.created_at
		FROM auto_services asv
		JOIN services svc ON svc.id = asv.service_id
		LEFT JOIN institutions inst ON inst.id = svc.institution_id
		WHERE asv.auto_id = ?
		ORDER BY inst.name, svc.name
	`, autoID)
	if err != nil {
		return nil, fmt.Errorf("listar seguros del auto: %w", err)
	}
	defer rows.Close()

	var results []models.AutoServiceDetail
	for rows.Next() {
		var d models.AutoServiceDetail
		var institutionID sql.NullInt64
		var startDate, endDate, certificate sql.NullString
		var createdAt sql.NullTime
		if err := rows.Scan(
			&d.ID, &d.AutoID, &d.ServiceID, &d.CoverageType,
			&d.PolicyNumber, &certificate, &d.InsurerNumber,
			&d.ServiceName, &d.InstitutionName, &institutionID,
			&d.SuggestedAmount, &d.Frequency, &d.IconKey, &d.Active,
			&startDate, &endDate, &d.IsRecurring,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("escanear seguro: %w", err)
		}
		if institutionID.Valid {
			d.InstitutionID = &institutionID.Int64
		}
		if certificate.Valid {
			d.Certificate = &certificate.String
		}
		if startDate.Valid {
			d.StartDate = &startDate.String
		}
		if endDate.Valid {
			d.EndDate = &endDate.String
		}
		if createdAt.Valid {
			d.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z")
		}
		results = append(results, d)
	}
	return results, rows.Err()
}

// Create asocia un servicio a un auto como seguro.
func (s *AutoServiceStorage) Create(ctx context.Context, autoID, serviceID int64, coverageType, policyNumber, certificate, insurerNumber string) (*models.AutoService, error) {
	var certPtr *string
	if certificate != "" {
		certPtr = &certificate
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO auto_services (auto_id, service_id, coverage_type, policy_number, certificate, insurer_number) VALUES (?, ?, ?, ?, ?, ?)
	`, autoID, serviceID, coverageType, policyNumber, certPtr, insurerNumber)
	if err != nil {
		return nil, fmt.Errorf("asociar seguro: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de seguro: %w", err)
	}
	return &models.AutoService{ID: id, AutoID: autoID, ServiceID: serviceID, CoverageType: coverageType, PolicyNumber: policyNumber, Certificate: certPtr, InsurerNumber: insurerNumber}, nil
}

// Delete elimina la asociación de un seguro.
func (s *AutoServiceStorage) Delete(ctx context.Context, autoID, serviceID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM auto_services WHERE auto_id = ? AND service_id = ?
	`, autoID, serviceID)
	if err != nil {
		return fmt.Errorf("eliminar seguro: %w", err)
	}
	return nil
}

// ListAvailableServices devuelve servicios de la categoría "insurance" no asociados a un auto.
func (s *AutoServiceStorage) ListAvailableServices(ctx context.Context, autoID int64) ([]models.Service, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.home_id, s.name, s.institution, s.currency_id, s.frequency,
			s.suggested_amount, s.active, s.icon_key, s.billing_type, s.billing_day, s.auto_generate,
			s.institution_id, s.institution_analyzer_id,
			s.start_date, s.end_date, s.is_recurring,
			(SELECT b.status FROM bills b WHERE b.service_id = s.id AND (b.amount > 0 OR b.invoice_number != '') ORDER BY b.year DESC, b.month DESC, b.id DESC LIMIT 1) AS latest_bill_status,
			s.deleted_at, s.created_at, s.updated_at
		FROM services s
		JOIN institutions i ON s.institution_id = i.id
		JOIN institution_categories ic ON i.category_id = ic.id
		WHERE s.deleted_at IS NULL
		  AND ic.key = 'insurance'
		  AND s.id NOT IN (SELECT service_id FROM auto_services WHERE auto_id = ?)
		ORDER BY s.name
	`, autoID)
	if err != nil {
		return nil, fmt.Errorf("listar servicios disponibles: %w", err)
	}
	defer rows.Close()
	return scanServices(rows)
}
