package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// Allowed service frequencies.
const (
	FrequencyMonthly = "monthly"
	FrequencyYearly  = "yearly"
)

// Predefined service icon keys.
var AllowedIconKeys = map[string]bool{
	// Utilidades
	"internet": true, "water": true, "electricity": true, "gas": true,
	"phone": true, "tv": true, "trash": true, "delete": true, "recycling": true,
	"sewer": true, "wifi": true, "cable": true, "satellite": true,
	// Hogar
	"home": true, "apartment": true, "building": true, "key": true,
	"lock": true, "door": true, "window": true, "garden": true,
	"garage": true, "parking": true, "pool": true, "roof": true,
	// Seguridad
	"shield": true, "alarm": true, "camera": true, "fire": true,
	"smoke": true, "extinguisher": true,
	// Limpieza
	"cleaning": true, "broom": true, "mop": true, "laundry": true,
	"dishwasher": true, "vacuum": true,
	// Mantenimiento
	"wrench": true, "hammer": true, "screwdriver": true, "paint": true,
	"plumbing": true, "electrical": true, "pest": true,
	// Salud
	"health": true, "hospital": true, "pharmacy": true, "ambulance": true,
	"medical": true, "dental": true, "vision": true,
	// Educación
	"school": true, "university": true, "library": true, "books": true,
	// Transporte
	"bus": true, "taxi": true, "car": true, "motorcycle": true,
	"bicycle": true, "fuel": true, "toll": true, "insurance_car": true,
	// Finanzas
	"bank": true, "insurance": true, "credit": true, "tax": true,
	"pension": true, "investment": true, "savings": true,
	// Comunicación
	"mail": true, "newspaper": true, "radio": true, "podcast": true,
	// Otros
	"other": true, "star": true, "heart": true, "calendar": true,
	"clock": true, "bell": true, "search": true, "user": true,
	"group": true, "gift": true, "coffee": true, "restaurant": true,
	"shopping": true, "pet": true, "baby": true, "elderly": true,
}

// ServiceService contiene la lógica de negocio para servicios.
type ServiceService struct {
	services   *storage.ServiceStorage
	homes      *storage.HomeStorage
	currencies *storage.CurrencyStorage
	bills      *storage.BillStorage
}

// NewServiceService crea un nuevo ServiceService.
func NewServiceService(
	services *storage.ServiceStorage,
	homes *storage.HomeStorage,
	currencies *storage.CurrencyStorage,
	bills *storage.BillStorage,
) *ServiceService {
	return &ServiceService{
		services:   services,
		homes:      homes,
		currencies: currencies,
		bills:      bills,
	}
}

// List devuelve los servicios activos, opcionalmente filtrados por home_id.
func (s *ServiceService) List(ctx context.Context, homeID *int64) ([]models.Service, error) {
	return s.services.List(ctx, homeID)
}

// GetByID busca un servicio por ID.
func (s *ServiceService) GetByID(ctx context.Context, id int64) (*models.Service, error) {
	return s.services.GetByID(ctx, id)
}

// Create crea un nuevo servicio y genera su factura inicial si está activo.
func (s *ServiceService) Create(ctx context.Context, svc *models.Service) (*models.Service, error) {
	if err := s.validate(ctx, svc); err != nil {
		return nil, err
	}

	created, err := s.services.Create(ctx, svc)
	if err != nil {
		return nil, err
	}

	if created.Active {
		_ = s.generateCurrentBill(ctx, created)
	}

	return created, nil
}

// Update actualiza un servicio existente.
func (s *ServiceService) Update(ctx context.Context, svc *models.Service) (*models.Service, error) {
	if svc.ID == 0 {
		return nil, fmt.Errorf("id de servicio requerido")
	}
	if err := s.validate(ctx, svc); err != nil {
		return nil, err
	}

	updated, err := s.services.Update(ctx, svc)
	if err != nil {
		return nil, err
	}

	if updated.Active {
		_ = s.generateCurrentBill(ctx, updated)
	}

	return updated, nil
}

// Delete elimina lógicamente un servicio.
func (s *ServiceService) Delete(ctx context.Context, id int64) error {
	return s.services.SoftDelete(ctx, id)
}

// ReconcileBills genera facturas pendientes para el periodo actual de todos los servicios activos.
func (s *ServiceService) ReconcileBills(ctx context.Context) error {
	services, err := s.services.List(ctx, nil)
	if err != nil {
		return err
	}
	for _, svc := range services {
		if svc.Active {
			_ = s.generateCurrentBill(ctx, &svc)
		}
	}
	return nil
}

func (s *ServiceService) validate(ctx context.Context, svc *models.Service) error {
	svc.Name = strings.TrimSpace(svc.Name)
	svc.Institution = strings.TrimSpace(svc.Institution)
	svc.Frequency = strings.ToLower(strings.TrimSpace(svc.Frequency))
	svc.IconKey = strings.ToLower(strings.TrimSpace(svc.IconKey))
	svc.BillingType = strings.ToLower(strings.TrimSpace(svc.BillingType))

	if svc.BillingType == "" {
		svc.BillingType = "variable"
	}
	if svc.BillingType != "fixed" && svc.BillingType != "variable" {
		return fmt.Errorf("el tipo de facturación debe ser 'fixed' o 'variable'")
	}
	if svc.BillingDay < 1 || svc.BillingDay > 31 {
		svc.BillingDay = 1
	}

	if svc.Name == "" {
		return fmt.Errorf("el nombre del servicio es requerido")
	}
	if svc.HomeID == 0 {
		return fmt.Errorf("debe seleccionar un hogar")
	}
	if svc.CurrencyID == 0 {
		return fmt.Errorf("debe seleccionar una moneda")
	}
	if svc.Frequency != FrequencyMonthly && svc.Frequency != FrequencyYearly {
		return fmt.Errorf("la frecuencia debe ser mensual o anual")
	}
	if svc.SuggestedAmount < 0 {
		return fmt.Errorf("el monto sugerido no puede ser negativo")
	}
	if !AllowedIconKeys[svc.IconKey] {
		return fmt.Errorf("el icono seleccionado no es válido")
	}

	home, err := s.homes.GetByID(ctx, svc.HomeID)
	if err != nil {
		return fmt.Errorf("validar hogar: %w", err)
	}
	if home == nil {
		return fmt.Errorf("el hogar seleccionado no existe")
	}

	currency, err := s.currencies.GetByID(ctx, svc.CurrencyID)
	if err != nil {
		return fmt.Errorf("validar moneda: %w", err)
	}
	if currency == nil {
		return fmt.Errorf("la moneda seleccionada no existe")
	}

	return nil
}

func (s *ServiceService) generateCurrentBill(ctx context.Context, svc *models.Service) error {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if svc.Frequency == FrequencyYearly {
		month = 0
	}

	existing, err := s.bills.FindByServicePeriod(ctx, svc.ID, year, month)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	bill := &models.Bill{
		ServiceID: svc.ID,
		Year:      year,
		Month:     month,
		Amount:    svc.SuggestedAmount,
		Status:    "pending",
	}
	_, err = s.bills.Create(ctx, bill)
	return err
}
