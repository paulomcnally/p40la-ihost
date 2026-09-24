package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// ServiceCycleStorage encapsula el acceso a la tabla service_cycles (SPEC-091).
type ServiceCycleStorage struct {
	db *sql.DB
}

// NewServiceCycleStorage crea un nuevo ServiceCycleStorage.
func NewServiceCycleStorage(db *sql.DB) *ServiceCycleStorage {
	return &ServiceCycleStorage{db: db}
}

// Create inserta un ciclo nuevo para un servicio.
func (s *ServiceCycleStorage) Create(ctx context.Context, cycle *models.ServiceCycle) (*models.ServiceCycle, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO service_cycles (service_id, sequence, start_date, end_date)
		VALUES (?, ?, ?, ?)
	`, cycle.ServiceID, cycle.Sequence, cycle.StartDate, cycle.EndDate)
	if err != nil {
		return nil, fmt.Errorf("crear ciclo: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de ciclo: %w", err)
	}
	cycle.ID = id
	if cycle.CreatedAt.IsZero() {
		cycle.CreatedAt = time.Now()
	}
	return cycle, nil
}

// NextSequence devuelve la siguiente sequence para un servicio (max+1; base 1).
func (s *ServiceCycleStorage) NextSequence(ctx context.Context, serviceID int64) (int, error) {
	var max sql.NullInt64
	if err := s.db.QueryRowContext(ctx,
		`SELECT MAX(sequence) FROM service_cycles WHERE service_id = ?`, serviceID,
	).Scan(&max); err != nil {
		return 0, fmt.Errorf("obtener siguiente sequence de ciclo: %w", err)
	}
	if !max.Valid {
		return 1, nil
	}
	return int(max.Int64) + 1, nil
}

// Delete elimina un ciclo (solo se usa para revertir una renovación fallida).
func (s *ServiceCycleStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM service_cycles WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("eliminar ciclo: %w", err)
	}
	return nil
}

// ListByService devuelve los ciclos de un servicio ordenados por sequence.
func (s *ServiceCycleStorage) ListByService(ctx context.Context, serviceID int64) ([]models.ServiceCycle, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, service_id, sequence, start_date, end_date, created_at
		FROM service_cycles
		WHERE service_id = ?
		ORDER BY sequence
	`, serviceID)
	if err != nil {
		return nil, fmt.Errorf("listar ciclos: %w", err)
	}
	defer rows.Close()

	var cycles []models.ServiceCycle
	for rows.Next() {
		c, err := scanServiceCycle(rows)
		if err != nil {
			return nil, err
		}
		cycles = append(cycles, *c)
	}
	return cycles, rows.Err()
}

// LatestByService devuelve el ciclo de mayor sequence de un servicio, o nil.
func (s *ServiceCycleStorage) LatestByService(ctx context.Context, serviceID int64) (*models.ServiceCycle, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, service_id, sequence, start_date, end_date, created_at
		FROM service_cycles
		WHERE service_id = ?
		ORDER BY sequence DESC
		LIMIT 1
	`, serviceID)
	c, err := scanServiceCycle(row)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// FindForPeriod devuelve el ciclo que contiene el período (year, month) del
// servicio: el primer ciclo (por sequence) cuyo rango [start, end] contiene la
// fecha del período; si ninguno lo contiene, el último ciclo. Devuelve nil si
// el servicio no tiene ciclos (SPEC-091).
//
// La fecha del período es el primer día del mes (YYYY-MM-01), o el 1 de enero
// del año (YYYY-01-01) para facturas anuales (month == 0).
func (s *ServiceCycleStorage) FindForPeriod(ctx context.Context, serviceID int64, year, month int) (*models.ServiceCycle, error) {
	monthDate := month
	if monthDate == 0 {
		monthDate = 1
	}
	period := fmt.Sprintf("%04d-%02d-01", year, monthDate)

	row := s.db.QueryRowContext(ctx, `
		SELECT id, service_id, sequence, start_date, end_date, created_at
		FROM service_cycles
		WHERE service_id = ?
		  AND start_date IS NOT NULL AND start_date <= ?
		  AND (end_date IS NULL OR end_date >= ?)
		ORDER BY sequence DESC
		LIMIT 1
	`, serviceID, period, period)
	c, err := scanServiceCycle(row)
	if err != nil {
		return nil, err
	}
	if c != nil {
		return c, nil
	}
	return s.LatestByService(ctx, serviceID)
}

// BackfillExistingCycles crea el ciclo 1 para cada servicio con vigencia y
// asocia sus facturas existentes (idempotente para pruebas y re-ejecución).
func (s *ServiceCycleStorage) BackfillExistingCycles(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO service_cycles (service_id, sequence, start_date, end_date)
		SELECT id, 1, start_date, end_date
		FROM services
		WHERE deleted_at IS NULL AND start_date IS NOT NULL AND end_date IS NOT NULL
	`)
	if err != nil {
		return fmt.Errorf("backfill de ciclos: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE bills
		SET cycle_id = (
			SELECT sc.id FROM service_cycles sc
			WHERE sc.service_id = bills.service_id
			ORDER BY sc.sequence DESC LIMIT 1
		)
		WHERE cycle_id IS NULL
		  AND EXISTS (SELECT 1 FROM service_cycles sc WHERE sc.service_id = bills.service_id)
	`)
	if err != nil {
		return fmt.Errorf("asociar facturas al backfill de ciclos: %w", err)
	}
	return nil
}

type cycleScanner interface {
	Scan(dest ...any) error
}

func scanServiceCycle(sc cycleScanner) (*models.ServiceCycle, error) {
	var c models.ServiceCycle
	var startDate, endDate sql.NullString
	var createdAt sql.NullTime
	if err := sc.Scan(&c.ID, &c.ServiceID, &c.Sequence, &startDate, &endDate, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear ciclo: %w", err)
	}
	if startDate.Valid {
		c.StartDate = &startDate.String
	}
	if endDate.Valid {
		c.EndDate = &endDate.String
	}
	if createdAt.Valid {
		c.CreatedAt = createdAt.Time
	}
	return &c, nil
}
