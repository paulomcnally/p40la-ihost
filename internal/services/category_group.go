package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// CategoryGroupService contiene la lógica de negocio para grupos de categorías
// de presupuesto (SPEC-093).
type CategoryGroupService struct {
	groups   *storage.CategoryGroupStorage
	catStore *storage.CategoryStorage
}

// NewCategoryGroupService crea un nuevo CategoryGroupService.
func NewCategoryGroupService(groups *storage.CategoryGroupStorage, catStore *storage.CategoryStorage) *CategoryGroupService {
	return &CategoryGroupService{groups: groups, catStore: catStore}
}

// List devuelve todos los grupos con sus categorías activas.
func (s *CategoryGroupService) List(ctx context.Context) ([]BudgetCategoryGroup, error) {
	groups, err := s.groups.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]BudgetCategoryGroup, 0, len(groups))
	for i := range groups {
		cats, err := s.catStore.ListByGroup(ctx, groups[i].ID)
		if err != nil {
			return nil, err
		}
		if cats == nil {
			cats = []models.Category{}
		}
		result = append(result, BudgetCategoryGroup{
			ID:         groups[i].ID,
			Name:       groups[i].Name,
			Icon:       groups[i].Icon,
			SortOrder:  groups[i].SortOrder,
			Categories: cats,
		})
	}
	return result, nil
}

// GetByID busca un grupo por ID.
func (s *CategoryGroupService) GetByID(ctx context.Context, id int64) (*models.CategoryGroup, error) {
	return s.groups.GetByID(ctx, id)
}

// Create crea un nuevo grupo.
func (s *CategoryGroupService) Create(ctx context.Context, name, icon string) (*models.CategoryGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("el nombre del grupo es requerido")
	}
	return s.groups.Create(ctx, name, icon, 0)
}

// Update actualiza un grupo existente.
func (s *CategoryGroupService) Update(ctx context.Context, id int64, name, icon string) (*models.CategoryGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("el nombre del grupo es requerido")
	}
	group, err := s.groups.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, fmt.Errorf("el grupo no existe")
	}
	return s.groups.Update(ctx, id, name, icon, group.SortOrder)
}

// Delete elimina un grupo si no tiene categorías activas.
func (s *CategoryGroupService) Delete(ctx context.Context, id int64) error {
	count, err := s.groups.CountCategories(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("el grupo tiene categorías; archívalas antes de eliminarlo")
	}
	return s.groups.Delete(ctx, id)
}

// Reorder actualiza el orden de los grupos.
func (s *CategoryGroupService) Reorder(ctx context.Context, ids []int64) error {
	return s.groups.Reorder(ctx, ids)
}