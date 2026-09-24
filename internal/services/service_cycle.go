package services

import (
	"context"
	"fmt"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// ServiceCycleService contiene la lógica de negocio de los ciclos de servicio
// y su renovación (SPEC-091).
type ServiceCycleService struct {
	services *storage.ServiceStorage
	cycles   *storage.ServiceCycleStorage
}

// NewServiceCycleService crea un nuevo ServiceCycleService.
func NewServiceCycleService(services *storage.ServiceStorage, cycles *storage.ServiceCycleStorage) *ServiceCycleService {
	return &ServiceCycleService{services: services, cycles: cycles}
}

// ListByService devuelve los ciclos de un servicio.
func (s *ServiceCycleService) ListByService(ctx context.Context, serviceID int64) ([]models.ServiceCycle, error) {
	return s.cycles.ListByService(ctx, serviceID)
}

// Renew renueva un servicio creando un ciclo nuevo (sequence max+1) y
// actualizando su vigencia y monto sugerido. Conserva el webhook_uuid y el
// historial; las facturas existentes mantienen su cycle_id (rastro del ciclo
// anterior). Aplica a todos los servicios con período renovable.
func (s *ServiceCycleService) Renew(ctx context.Context, serviceID int64, startDate, endDate string, suggestedAmount *float64) (*models.Service, *models.ServiceCycle, error) {
	svc, err := s.services.GetByID(ctx, serviceID)
	if err != nil {
		return nil, nil, err
	}
	if svc == nil {
		return nil, nil, fmt.Errorf("servicio no encontrado")
	}

	start, errStart := time.Parse("2006-01-02", startDate)
	end, errEnd := time.Parse("2006-01-02", endDate)
	if errStart != nil || errEnd != nil {
		return nil, nil, fmt.Errorf("start_date y end_date deben ser fechas válidas (YYYY-MM-DD)")
	}
	if !start.Before(end) {
		return nil, nil, fmt.Errorf("start_date debe ser anterior a end_date")
	}
	if suggestedAmount != nil && *suggestedAmount < 0 {
		return nil, nil, fmt.Errorf("el monto sugerido no puede ser negativo")
	}

	sequence, err := s.cycles.NextSequence(ctx, serviceID)
	if err != nil {
		return nil, nil, err
	}
	cycle, err := s.cycles.Create(ctx, &models.ServiceCycle{
		ServiceID: serviceID,
		Sequence:  sequence,
		StartDate: &startDate,
		EndDate:   &endDate,
	})
	if err != nil {
		return nil, nil, err
	}

	svc.StartDate = &startDate
	svc.EndDate = &endDate
	if suggestedAmount != nil {
		svc.SuggestedAmount = *suggestedAmount
	}

	updated, err := s.services.Update(ctx, svc)
	if err != nil {
		_ = s.cycles.Delete(ctx, cycle.ID)
		return nil, nil, err
	}
	return updated, cycle, nil
}
