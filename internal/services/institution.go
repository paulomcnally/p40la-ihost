package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/analyzers"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type InstitutionService struct {
	instStorage *storage.InstitutionStorage
}

func NewInstitutionService(instStorage *storage.InstitutionStorage) *InstitutionService {
	return &InstitutionService{instStorage: instStorage}
}

func (s *InstitutionService) List(ctx context.Context) ([]models.Institution, error) {
	return s.instStorage.List(ctx)
}

func (s *InstitutionService) GetByID(ctx context.Context, id int64) (*models.Institution, error) {
	return s.instStorage.GetByID(ctx, id)
}

func (s *InstitutionService) Create(ctx context.Context, inst *models.Institution) (*models.Institution, error) {
	inst.Name = strings.TrimSpace(inst.Name)
	if inst.Name == "" {
		return nil, fmt.Errorf("el nombre de la institución es requerido")
	}
	return s.instStorage.Create(ctx, inst)
}

func (s *InstitutionService) Update(ctx context.Context, inst *models.Institution) (*models.Institution, error) {
	inst.Name = strings.TrimSpace(inst.Name)
	if inst.Name == "" {
		return nil, fmt.Errorf("el nombre de la institución es requerido")
	}
	return s.instStorage.Update(ctx, inst)
}

func (s *InstitutionService) Delete(ctx context.Context, id int64) error {
	return s.instStorage.Delete(ctx, id)
}

func (s *InstitutionService) SetAnalyzers(ctx context.Context, institutionID int64, analyzerIDs []string) error {
	for _, aid := range analyzerIDs {
		if _, ok := analyzers.Get(aid); !ok {
			return fmt.Errorf("analyzer %q no está registrado", aid)
		}
	}
	return s.instStorage.SetAnalyzers(ctx, institutionID, analyzerIDs)
}

func (s *InstitutionService) GetAnalyzers(ctx context.Context, institutionID int64) ([]models.InstitutionAnalyzer, error) {
	return s.instStorage.GetAnalyzers(ctx, institutionID)
}
