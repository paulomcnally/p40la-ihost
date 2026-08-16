package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// AutoService contiene la lógica de negocio para autos.
type AutoService struct {
	storage *storage.AutoStorage
}

// NewAutoService crea un nuevo AutoService.
func NewAutoService(st *storage.AutoStorage) *AutoService {
	return &AutoService{storage: st}
}

// List devuelve todos los autos.
func (s *AutoService) List(ctx context.Context) ([]models.Auto, error) {
	return s.storage.List(ctx)
}

// GetByID busca un auto por ID.
func (s *AutoService) GetByID(ctx context.Context, id int64) (*models.Auto, error) {
	return s.storage.GetByID(ctx, id)
}

// Create crea un nuevo auto.
func (s *AutoService) Create(ctx context.Context, year int64, model, brand, color, icon, motor, chasis, vin, placa string) (*models.Auto, error) {
	model = strings.TrimSpace(model)
	brand = strings.TrimSpace(brand)
	color = strings.TrimSpace(color)
	icon = strings.TrimSpace(icon)
	motor = strings.TrimSpace(motor)
	chasis = strings.TrimSpace(chasis)
	vin = strings.TrimSpace(vin)
	placa = strings.TrimSpace(placa)

	if model == "" {
		return nil, fmt.Errorf("el modelo es requerido")
	}
	if brand == "" {
		return nil, fmt.Errorf("la marca es requerida")
	}
	if color == "" {
		return nil, fmt.Errorf("el color es requerido")
	}
	if motor == "" {
		return nil, fmt.Errorf("el motor es requerido")
	}
	if chasis == "" {
		return nil, fmt.Errorf("el chasis es requerido")
	}
	if vin == "" {
		return nil, fmt.Errorf("el VIN es requerido")
	}
	if placa == "" {
		return nil, fmt.Errorf("la placa es requerida")
	}
	if year < 1900 || year > 2100 {
		return nil, fmt.Errorf("el año debe estar entre 1900 y 2100")
	}
	if icon == "" {
		icon = "vehicle"
	}

	return s.storage.Create(ctx, year, model, brand, color, icon, motor, chasis, vin, placa)
}

// Update actualiza un auto existente.
func (s *AutoService) Update(ctx context.Context, id int64, year int64, model, brand, color, icon, motor, chasis, vin, placa string) (*models.Auto, error) {
	model = strings.TrimSpace(model)
	brand = strings.TrimSpace(brand)
	color = strings.TrimSpace(color)
	icon = strings.TrimSpace(icon)
	motor = strings.TrimSpace(motor)
	chasis = strings.TrimSpace(chasis)
	vin = strings.TrimSpace(vin)
	placa = strings.TrimSpace(placa)

	if model == "" {
		return nil, fmt.Errorf("el modelo es requerido")
	}
	if brand == "" {
		return nil, fmt.Errorf("la marca es requerida")
	}
	if color == "" {
		return nil, fmt.Errorf("el color es requerido")
	}
	if motor == "" {
		return nil, fmt.Errorf("el motor es requerido")
	}
	if chasis == "" {
		return nil, fmt.Errorf("el chasis es requerido")
	}
	if vin == "" {
		return nil, fmt.Errorf("el VIN es requerido")
	}
	if placa == "" {
		return nil, fmt.Errorf("la placa es requerida")
	}
	if year < 1900 || year > 2100 {
		return nil, fmt.Errorf("el año debe estar entre 1900 y 2100")
	}
	if icon == "" {
		icon = "vehicle"
	}

	return s.storage.Update(ctx, id, year, model, brand, color, icon, motor, chasis, vin, placa)
}

// Delete elimina un auto.
func (s *AutoService) Delete(ctx context.Context, id int64) error {
	return s.storage.Delete(ctx, id)
}
