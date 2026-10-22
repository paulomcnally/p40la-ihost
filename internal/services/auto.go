package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// AutoColors son los códigos canónicos de color admitidos para un auto
// (SPEC-118). Debe coincidir con frontend/src/constants/autoColors.ts.
var AutoColors = []string{
	"white",
	"black",
	"gray",
	"silver",
	"red",
	"blue",
	"green",
	"yellow",
	"orange",
	"brown",
	"beige",
	"gold",
	"burgundy",
	"light_blue",
	"pink",
	"purple",
}

var autoColorSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(AutoColors))
	for _, c := range AutoColors {
		set[c] = struct{}{}
	}
	return set
}()

// normalizeAutoColor valida el color contra la paleta canónica y devuelve el
// código en minúsculas. El color es obligatorio (SPEC-118 REQ-004).
func normalizeAutoColor(color string) (string, error) {
	color = strings.ToLower(strings.TrimSpace(color))
	if color == "" {
		return "", fmt.Errorf("el color es requerido")
	}
	if _, ok := autoColorSet[color]; !ok {
		return "", fmt.Errorf("el color no es válido")
	}
	return color, nil
}

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

// Create crea un nuevo auto. motor, chasis, VIN y placa son opcionales.
func (s *AutoService) Create(ctx context.Context, year int64, model, brand, color, icon, motor, chasis, vin, placa string) (*models.Auto, error) {
	model = strings.TrimSpace(model)
	brand = strings.TrimSpace(brand)
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
	normalizedColor, err := normalizeAutoColor(color)
	if err != nil {
		return nil, err
	}
	if year < 1900 || year > 2100 {
		return nil, fmt.Errorf("el año debe estar entre 1900 y 2100")
	}
	if icon == "" {
		icon = "vehicle"
	}

	return s.storage.Create(ctx, year, model, brand, normalizedColor, icon, motor, chasis, vin, placa)
}

// Update actualiza un auto existente. motor, chasis, VIN y placa son opcionales
// y pueden limpiarse enviando cadena vacía.
func (s *AutoService) Update(ctx context.Context, id int64, year int64, model, brand, color, icon, motor, chasis, vin, placa string) (*models.Auto, error) {
	model = strings.TrimSpace(model)
	brand = strings.TrimSpace(brand)
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
	normalizedColor, err := normalizeAutoColor(color)
	if err != nil {
		return nil, err
	}
	if year < 1900 || year > 2100 {
		return nil, fmt.Errorf("el año debe estar entre 1900 y 2100")
	}
	if icon == "" {
		icon = "vehicle"
	}

	return s.storage.Update(ctx, id, year, model, brand, normalizedColor, icon, motor, chasis, vin, placa)
}

// Delete elimina un auto.
func (s *AutoService) Delete(ctx context.Context, id int64) error {
	return s.storage.Delete(ctx, id)
}
