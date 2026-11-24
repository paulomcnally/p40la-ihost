package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// ErrHomeHasActiveRental indica que no se puede cambiar el uso de una vivienda
// mientras tenga un contrato o una reserva activa (SPEC-143).
var ErrHomeHasActiveRental = errors.New("la vivienda tiene un alquiler o reserva activa")

// ErrInvalidHomeUsage indica un valor de uso no permitido (SPEC-143).
var ErrInvalidHomeUsage = errors.New("uso inválido")

// HomeRentalGuard consulta el estado de alquiler de una vivienda. Lo implementa
// el módulo realestate.rental y lo inyecta main.go (SPEC-143).
type HomeRentalGuard interface {
	HasActiveRental(ctx context.Context, homeID int64) (bool, error)
}

// HomeService contiene la lógica de negocio para hogares.
type HomeService struct {
	storage     *storage.HomeStorage
	rentalGuard HomeRentalGuard
}

// NewHomeService crea un nuevo HomeService.
func NewHomeService(st *storage.HomeStorage) *HomeService {
	return &HomeService{storage: st}
}

// SetRentalGuard conecta el guard de alquileres (SPEC-143).
func (s *HomeService) SetRentalGuard(guard HomeRentalGuard) {
	s.rentalGuard = guard
}

// List devuelve todos los hogares activos.
func (s *HomeService) List(ctx context.Context) ([]models.Home, error) {
	return s.storage.List(ctx)
}

// GetByID busca un hogar por ID.
func (s *HomeService) GetByID(ctx context.Context, id int64) (*models.Home, error) {
	return s.storage.GetByID(ctx, id)
}

// Count devuelve la cantidad de hogares activos.
func (s *HomeService) Count(ctx context.Context) (int64, error) {
	return s.storage.Count(ctx)
}

// Create crea un nuevo hogar.
func (s *HomeService) Create(ctx context.Context, name, address string, usage ...string) (*models.Home, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("el nombre del hogar es requerido")
	}
	normalized := models.HomeUsageRental
	if len(usage) > 0 {
		var err error
		normalized, err = normalizeHomeUsage(usage[0])
		if err != nil {
			return nil, err
		}
	}
	return s.storage.Create(ctx, name, strings.TrimSpace(address), normalized)
}

// Update actualiza un hogar existente. Si no se indica uso, conserva el actual.
func (s *HomeService) Update(ctx context.Context, id int64, name, address string, usage ...string) (*models.Home, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("el nombre del hogar es requerido")
	}

	current, err := s.storage.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, nil
	}

	normalized := current.Usage
	if len(usage) > 0 {
		normalized, err = normalizeHomeUsage(usage[0])
		if err != nil {
			return nil, err
		}
	}

	// Cambiar el uso fuera de «rental» mientras hay contrato o reserva activa
	// dejaría datos contradictorios (ADR-003).
	if normalized != models.HomeUsageRental && normalized != current.Usage && s.rentalGuard != nil {
		active, err := s.rentalGuard.HasActiveRental(ctx, id)
		if err != nil {
			return nil, err
		}
		if active {
			return nil, ErrHomeHasActiveRental
		}
	}

	return s.storage.Update(ctx, id, name, strings.TrimSpace(address), normalized)
}

// Delete elimina lógicamente un hogar.
func (s *HomeService) Delete(ctx context.Context, id int64) error {
	return s.storage.SoftDelete(ctx, id)
}

// Reorder persiste el orden de los hogares según el array de IDs recibido.
func (s *HomeService) Reorder(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return fmt.Errorf("la lista de IDs es requerida")
	}
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return fmt.Errorf("ID duplicado en la lista: %d", id)
		}
		seen[id] = true
	}
	return s.storage.Reorder(ctx, ids)
}

// normalizeHomeUsage valida y normaliza el uso de una vivienda. Vacío equivale
// a «rental», el estado inicial de toda casa nueva.
func normalizeHomeUsage(usage string) (string, error) {
	usage = strings.ToLower(strings.TrimSpace(usage))
	if usage == "" {
		return models.HomeUsageRental, nil
	}
	for _, valid := range models.HomeUsages {
		if usage == valid {
			return usage, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrInvalidHomeUsage, usage)
}
