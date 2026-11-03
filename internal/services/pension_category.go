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

// Create crea una categoría con los valores por defecto (50 %, NIO).
func (s *PensionCategoryService) Create(ctx context.Context, name, description string, autoGenerate bool) (*models.PensionCategory, error) {
	return s.CreateFull(ctx, name, description, autoGenerate, 50, "NIO")
}

// CreateFull crea una categoría con reparto y moneda explícitos.
func (s *PensionCategoryService) CreateFull(ctx context.Context, name, description string, autoGenerate bool, sharePercent int, currency string) (*models.PensionCategory, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("el nombre es requerido")
	}
	sharePercent, currency, err := normalizeShareAndCurrency(sharePercent, currency)
	if err != nil {
		return nil, err
	}
	return s.storage.Create(ctx, strings.TrimSpace(name), strings.TrimSpace(description), autoGenerate, sharePercent, currency)
}

// Update actualiza una categoría conservando el reparto y la moneda actuales.
func (s *PensionCategoryService) Update(ctx context.Context, id int64, name, description string, autoGenerate bool) (*models.PensionCategory, error) {
	current, err := s.storage.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, nil
	}
	return s.UpdateFull(ctx, id, name, description, autoGenerate, current.SharePercent, current.Currency)
}

// UpdateFull actualiza una categoría con reparto y moneda explícitos.
func (s *PensionCategoryService) UpdateFull(ctx context.Context, id int64, name, description string, autoGenerate bool, sharePercent int, currency string) (*models.PensionCategory, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("el nombre es requerido")
	}
	sharePercent, currency, err := normalizeShareAndCurrency(sharePercent, currency)
	if err != nil {
		return nil, err
	}
	return s.storage.Update(ctx, id, strings.TrimSpace(name), strings.TrimSpace(description), autoGenerate, sharePercent, currency)
}

func normalizeShareAndCurrency(sharePercent int, currency string) (int, string, error) {
	if sharePercent != 50 && sharePercent != 100 {
		return 0, "", fmt.Errorf("el reparto debe ser 50 o 100")
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		currency = "NIO"
	}
	if len(currency) != 3 {
		return 0, "", fmt.Errorf("la moneda debe tener 3 letras")
	}
	return sharePercent, currency, nil
}

// Delete elimina una categoría de pensión.
func (s *PensionCategoryService) Delete(ctx context.Context, id int64) error {
	return s.storage.Delete(ctx, id)
}
