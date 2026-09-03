package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// ChildSupportConfigService contiene la lógica de negocio de configs de pensión.
type ChildSupportConfigService struct {
	storage         *storage.ChildSupportConfigStorage
	childStorage    *storage.ChildStorage
	categoryStorage *storage.PensionCategoryStorage
}

// NewChildSupportConfigService crea un nuevo ChildSupportConfigService.
func NewChildSupportConfigService(st *storage.ChildSupportConfigStorage, childStorage *storage.ChildStorage, categoryStorage *storage.PensionCategoryStorage) *ChildSupportConfigService {
	return &ChildSupportConfigService{storage: st, childStorage: childStorage, categoryStorage: categoryStorage}
}

// List devuelve todas las configs.
func (s *ChildSupportConfigService) List(ctx context.Context) ([]models.ChildSupportConfig, error) {
	configs, err := s.storage.List(ctx)
	if err != nil {
		return nil, err
	}
	if configs == nil {
		return []models.ChildSupportConfig{}, nil
	}
	return configs, nil
}

// ListByChild devuelve las configs de un hijo.
func (s *ChildSupportConfigService) ListByChild(ctx context.Context, childID int64) ([]models.ChildSupportConfig, error) {
	return s.storage.ListByChild(ctx, childID)
}

// GetByID busca una config por ID.
func (s *ChildSupportConfigService) GetByID(ctx context.Context, id int64) (*models.ChildSupportConfig, error) {
	return s.storage.GetByID(ctx, id)
}

// Create crea una nueva config validando hijo/categoría/amount/currency.
func (s *ChildSupportConfigService) Create(ctx context.Context, childID, categoryID int64, amount float64, currency string, isActive, autoGenerate bool) (*models.ChildSupportConfig, error) {
	if err := s.validate(ctx, childID, categoryID, amount, currency); err != nil {
		return nil, err
	}
	exists, err := s.storage.Exists(ctx, childID, categoryID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("ya existe una config para este hijo y categoría")
	}
	return s.storage.Create(ctx, &models.ChildSupportConfig{
		ChildID:           childID,
		PensionCategoryID: categoryID,
		Amount:            amount,
		Currency:          normalizeCurrency(currency),
		IsActive:          isActive,
		AutoGenerate:      autoGenerate,
	})
}

// Update actualiza una config existente.
func (s *ChildSupportConfigService) Update(ctx context.Context, id int64, categoryID int64, amount float64, currency string, isActive, autoGenerate bool) (*models.ChildSupportConfig, error) {
	existing, err := s.storage.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("config no encontrada")
	}
	if err := s.validate(ctx, existing.ChildID, categoryID, amount, currency); err != nil {
		return nil, err
	}
	return s.storage.Update(ctx, id, &models.ChildSupportConfig{
		PensionCategoryID: categoryID,
		Amount:            amount,
		Currency:          normalizeCurrency(currency),
		IsActive:          isActive,
		AutoGenerate:      autoGenerate,
	})
}

// Delete elimina una config.
func (s *ChildSupportConfigService) Delete(ctx context.Context, id int64) error {
	return s.storage.Delete(ctx, id)
}

func (s *ChildSupportConfigService) validate(ctx context.Context, childID, categoryID int64, amount float64, currency string) error {
	child, err := s.childStorage.GetByID(ctx, childID)
	if err != nil {
		return fmt.Errorf("error al validar el hijo")
	}
	if child == nil {
		return fmt.Errorf("el hijo no existe")
	}
	cat, err := s.categoryStorage.GetByID(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("error al validar la categoría")
	}
	if cat == nil {
		return fmt.Errorf("la categoría no existe")
	}
	if amount <= 0 {
		return fmt.Errorf("el monto debe ser mayor a cero")
	}
	if c := strings.ToUpper(strings.TrimSpace(currency)); len(c) != 3 {
		return fmt.Errorf("la moneda debe tener 3 letras")
	}
	return nil
}
