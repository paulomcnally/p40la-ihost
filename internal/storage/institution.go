package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

type InstitutionStorage struct {
	db *sql.DB
}

func NewInstitutionStorage(db *sql.DB) *InstitutionStorage {
	return &InstitutionStorage{db: db}
}

func (s *InstitutionStorage) List(ctx context.Context) ([]models.Institution, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, created_at, updated_at FROM institutions ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("listar instituciones: %w", err)
	}
	defer rows.Close()

	var institutions []models.Institution
	for rows.Next() {
		var inst models.Institution
		if err := rows.Scan(&inst.ID, &inst.Name, &inst.CreatedAt, &inst.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear institución: %w", err)
		}
		institutions = append(institutions, inst)
	}
	return institutions, rows.Err()
}

func (s *InstitutionStorage) GetByID(ctx context.Context, id int64) (*models.Institution, error) {
	var inst models.Institution
	if err := s.db.QueryRowContext(ctx,
		"SELECT id, name, created_at, updated_at FROM institutions WHERE id = ?", id,
	).Scan(&inst.ID, &inst.Name, &inst.CreatedAt, &inst.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("obtener institución: %w", err)
	}
	return &inst, nil
}

func (s *InstitutionStorage) Create(ctx context.Context, inst *models.Institution) (*models.Institution, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO institutions (name) VALUES (?)", inst.Name,
	)
	if err != nil {
		return nil, fmt.Errorf("crear institución: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener ID: %w", err)
	}
	return s.GetByID(ctx, id)
}

func (s *InstitutionStorage) Update(ctx context.Context, inst *models.Institution) (*models.Institution, error) {
	_, err := s.db.ExecContext(ctx,
		"UPDATE institutions SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", inst.Name, inst.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("actualizar institución: %w", err)
	}
	return s.GetByID(ctx, inst.ID)
}

func (s *InstitutionStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM institutions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("eliminar institución: %w", err)
	}
	return nil
}

func (s *InstitutionStorage) SetAnalyzers(ctx context.Context, institutionID int64, analyzerIDs []string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM institution_analyzers WHERE institution_id = ?", institutionID)
	if err != nil {
		return fmt.Errorf("limpiar analyzers: %w", err)
	}
	for _, aid := range analyzerIDs {
		_, err := s.db.ExecContext(ctx,
			"INSERT INTO institution_analyzers (institution_id, analyzer_id) VALUES (?, ?)", institutionID, aid,
		)
		if err != nil {
			return fmt.Errorf("insertar analyzer %s: %w", aid, err)
		}
	}
	return nil
}

func (s *InstitutionStorage) GetAnalyzers(ctx context.Context, institutionID int64) ([]models.InstitutionAnalyzer, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, institution_id, analyzer_id, created_at FROM institution_analyzers WHERE institution_id = ?", institutionID,
	)
	if err != nil {
		return nil, fmt.Errorf("obtener analyzers: %w", err)
	}
	defer rows.Close()

	var analyzers []models.InstitutionAnalyzer
	for rows.Next() {
		var a models.InstitutionAnalyzer
		if err := rows.Scan(&a.ID, &a.InstitutionID, &a.AnalyzerID, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("escanear analyzer: %w", err)
		}
		analyzers = append(analyzers, a)
	}
	return analyzers, rows.Err()
}

func (s *InstitutionStorage) GetAnalyzerOptions(ctx context.Context, institutionID int64) ([]models.InstitutionAnalyzer, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, institution_id, analyzer_id, created_at FROM institution_analyzers WHERE institution_id = ?", institutionID,
	)
	if err != nil {
		return nil, fmt.Errorf("obtener analyzer options: %w", err)
	}
	defer rows.Close()

	var options []models.InstitutionAnalyzer
	for rows.Next() {
		var a models.InstitutionAnalyzer
		if err := rows.Scan(&a.ID, &a.InstitutionID, &a.AnalyzerID, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("escanear analyzer: %w", err)
		}
		options = append(options, a)
	}
	return options, rows.Err()
}

func (s *InstitutionStorage) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM institutions").Scan(&count); err != nil {
		return 0, fmt.Errorf("contar instituciones: %w", err)
	}
	return count, nil
}
