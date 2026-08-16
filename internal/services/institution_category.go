package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

var keyRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type InstitutionCategoryService struct {
	catStorage *storage.InstitutionCategoryStorage
}

func NewInstitutionCategoryService(catStorage *storage.InstitutionCategoryStorage) *InstitutionCategoryService {
	return &InstitutionCategoryService{catStorage: catStorage}
}

func (s *InstitutionCategoryService) List(ctx context.Context) ([]models.InstitutionCategory, error) {
	return s.catStorage.List(ctx)
}

func (s *InstitutionCategoryService) GetByID(ctx context.Context, id int64) (*models.InstitutionCategory, error) {
	return s.catStorage.GetByID(ctx, id)
}

func (s *InstitutionCategoryService) GetByKey(ctx context.Context, key string) (*models.InstitutionCategory, error) {
	return s.catStorage.GetByKey(ctx, key)
}

func (s *InstitutionCategoryService) Create(ctx context.Context, cat *models.InstitutionCategory) (*models.InstitutionCategory, error) {
	cat.Key = strings.TrimSpace(cat.Key)
	cat.Name = strings.TrimSpace(cat.Name)
	if cat.Key == "" {
		return nil, fmt.Errorf("la key es requerida")
	}
	if cat.Name == "" {
		return nil, fmt.Errorf("el nombre es requerido")
	}
	if !keyRegex.MatchString(cat.Key) {
		return nil, fmt.Errorf("la key debe ser lowercase, sin espacios, solo letras, números y guiones bajos")
	}
	existing, _ := s.catStorage.GetByKey(ctx, cat.Key)
	if existing != nil {
		return nil, fmt.Errorf("ya existe una categoría con esa key")
	}
	if cat.IconKey == "" {
		cat.IconKey = "other"
	}
	return s.catStorage.Create(ctx, cat)
}

func (s *InstitutionCategoryService) Update(ctx context.Context, cat *models.InstitutionCategory) (*models.InstitutionCategory, error) {
	cat.Name = strings.TrimSpace(cat.Name)
	if cat.Name == "" {
		return nil, fmt.Errorf("el nombre es requerido")
	}
	existing, _ := s.catStorage.GetByID(ctx, cat.ID)
	if existing == nil {
		return nil, fmt.Errorf("categoría no encontrada")
	}
	if cat.IconKey == "" {
		cat.IconKey = existing.IconKey
	}
	return s.catStorage.Update(ctx, cat)
}

func (s *InstitutionCategoryService) Delete(ctx context.Context, id int64) error {
	return s.catStorage.Delete(ctx, id)
}
