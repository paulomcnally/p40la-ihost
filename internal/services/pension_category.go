package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// PensionCategoryService contiene la lógica de negocio para categorías de pensión.
type PensionCategoryService struct {
	storage *storage.PensionCategoryStorage
}

// NewPensionCategoryService crea un nuevo PensionCategoryService.
func NewPensionCategoryService(st *storage.PensionCategoryStorage) *PensionCategoryService {
	return &PensionCategoryService{storage: st}
}

// List devuelve todas las categorías de pensión.
func (s *PensionCategoryService) List(ctx context.Context) ([]models.PensionCategory, error) {
	return s.storage.List(ctx)
}

// GetByID busca una categoría de pensión por ID.
func (s *PensionCategoryService) GetByID(ctx context.Context, id int64) (*models.PensionCategory, error) {
	return s.storage.GetByID(ctx, id)
}

// Create crea una nueva categoría de pensión.
func (s *PensionCategoryService) Create(ctx context.Context, name, description string, autoGenerate bool) (*models.PensionCategory, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("el nombre es requerido")
	}
	return s.storage.Create(ctx, strings.TrimSpace(name), strings.TrimSpace(description), autoGenerate)
}

// Update actualiza una categoría de pensión existente.
func (s *PensionCategoryService) Update(ctx context.Context, id int64, name, description string, autoGenerate bool) (*models.PensionCategory, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("el nombre es requerido")
	}
	return s.storage.Update(ctx, id, strings.TrimSpace(name), strings.TrimSpace(description), autoGenerate)
}

// Delete elimina una categoría de pensión.
func (s *PensionCategoryService) Delete(ctx context.Context, id int64) error {
	return s.storage.Delete(ctx, id)
}