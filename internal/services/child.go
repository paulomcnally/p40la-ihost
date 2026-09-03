package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// ChildService contiene la lógica de negocio para hijos.
type ChildService struct {
	storage *storage.ChildStorage
}

// NewChildService crea un nuevo ChildService.
func NewChildService(st *storage.ChildStorage) *ChildService {
	return &ChildService{storage: st}
}

// List devuelve todos los hijos.
func (s *ChildService) List(ctx context.Context) ([]models.Child, error) {
	return s.storage.List(ctx)
}

// GetByID busca un hijo por ID.
func (s *ChildService) GetByID(ctx context.Context, id int64) (*models.Child, error) {
	return s.storage.GetByID(ctx, id)
}

// Create crea un nuevo hijo.
func (s *ChildService) Create(ctx context.Context, firstName, lastName, birthDate, notes string) (*models.Child, error) {
	if err := validateChild(firstName, lastName, birthDate); err != nil {
		return nil, err
	}
	return s.storage.Create(ctx, strings.TrimSpace(firstName), strings.TrimSpace(lastName), birthDate, strings.TrimSpace(notes))
}

// Update actualiza un hijo existente.
func (s *ChildService) Update(ctx context.Context, id int64, firstName, lastName, birthDate, notes string) (*models.Child, error) {
	if err := validateChild(firstName, lastName, birthDate); err != nil {
		return nil, err
	}
	return s.storage.Update(ctx, id, strings.TrimSpace(firstName), strings.TrimSpace(lastName), birthDate, strings.TrimSpace(notes))
}

// Delete elimina un hijo.
func (s *ChildService) Delete(ctx context.Context, id int64) error {
	return s.storage.Delete(ctx, id)
}

func validateChild(firstName, lastName, birthDate string) error {
	if strings.TrimSpace(firstName) == "" {
		return fmt.Errorf("los nombres son requeridos")
	}
	if strings.TrimSpace(lastName) == "" {
		return fmt.Errorf("los apellidos son requeridos")
	}
	if strings.TrimSpace(birthDate) == "" {
		return fmt.Errorf("la fecha de nacimiento es requerida")
	}
	parsed, err := time.Parse("2006-01-02", birthDate)
	if err != nil {
		return fmt.Errorf("la fecha de nacimiento debe tener formato AAAA-MM-DD")
	}
	today := time.Now()
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	if parsed.After(todayStart) {
		return fmt.Errorf("la fecha de nacimiento no puede ser futura")
	}
	return nil
}