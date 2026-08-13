package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// HomeService contiene la lógica de negocio para hogares.
type HomeService struct {
	storage *storage.HomeStorage
}

// NewHomeService crea un nuevo HomeService.
func NewHomeService(st *storage.HomeStorage) *HomeService {
	return &HomeService{storage: st}
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
func (s *HomeService) Create(ctx context.Context, name, address string) (*models.Home, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("el nombre del hogar es requerido")
	}
	return s.storage.Create(ctx, name, strings.TrimSpace(address))
}

// Update actualiza un hogar existente.
func (s *HomeService) Update(ctx context.Context, id int64, name, address string) (*models.Home, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("el nombre del hogar es requerido")
	}
	return s.storage.Update(ctx, id, name, strings.TrimSpace(address))
}

// Delete elimina lógicamente un hogar.
func (s *HomeService) Delete(ctx context.Context, id int64) error {
	return s.storage.SoftDelete(ctx, id)
}
