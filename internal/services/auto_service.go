package services

import (
	"context"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// AutoServiceService contiene la lógica de negocio para seguros de autos.
type AutoServiceService struct {
	autoServiceStorage *storage.AutoServiceStorage
}

// NewAutoServiceService crea un nuevo AutoServiceService.
func NewAutoServiceService(st *storage.AutoServiceStorage) *AutoServiceService {
	return &AutoServiceService{autoServiceStorage: st}
}

// ListByAuto devuelve los seguros de un auto.
func (s *AutoServiceService) ListByAuto(ctx context.Context, autoID int64) ([]models.AutoServiceDetail, error) {
	return s.autoServiceStorage.ListByAuto(ctx, autoID)
}

// Create asocia un servicio como seguro a un auto.
func (s *AutoServiceService) Create(ctx context.Context, autoID, serviceID int64, coverageType string) (*models.AutoService, error) {
	if coverageType != "daños_a_terceros" && coverageType != "full_cover" {
		return nil, fmt.Errorf("el tipo de cobertura debe ser 'daños_a_terceros' o 'full_cover'")
	}
	return s.autoServiceStorage.Create(ctx, autoID, serviceID, coverageType)
}

// Delete elimina un seguro de un auto.
func (s *AutoServiceService) Delete(ctx context.Context, autoID, serviceID int64) error {
	return s.autoServiceStorage.Delete(ctx, autoID, serviceID)
}

// ListAvailableServices devuelve servicios disponibles para asociar.
func (s *AutoServiceService) ListAvailableServices(ctx context.Context, autoID int64) ([]models.Service, error) {
	return s.autoServiceStorage.ListAvailableServices(ctx, autoID)
}
