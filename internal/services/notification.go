package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// NotificationService contiene la lógica de negocio para notificaciones.
type NotificationService struct {
	storage *storage.NotificationStorage
}

// NewNotificationService crea un nuevo NotificationService.
func NewNotificationService(st *storage.NotificationStorage) *NotificationService {
	return &NotificationService{storage: st}
}

// List devuelve todos los registros de notificación.
func (s *NotificationService) List(ctx context.Context) ([]models.Notification, error) {
	return s.storage.List(ctx)
}

// GetByID busca un registro por ID.
func (s *NotificationService) GetByID(ctx context.Context, id int64) (*models.Notification, error) {
	return s.storage.GetByID(ctx, id)
}

// Create crea un nuevo registro de notificación.
func (s *NotificationService) Create(ctx context.Context, name, email string, active bool) (*models.Notification, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	if name == "" {
		return nil, fmt.Errorf("el nombre es requerido")
	}
	if email == "" {
		return nil, fmt.Errorf("el email es requerido")
	}
	if !emailRegex.MatchString(email) {
		return nil, fmt.Errorf("el email no es válido")
	}

	return s.storage.Create(ctx, name, email, active)
}

// Update actualiza un registro existente.
func (s *NotificationService) Update(ctx context.Context, id int64, name, email string, active bool) (*models.Notification, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	if name == "" {
		return nil, fmt.Errorf("el nombre es requerido")
	}
	if email == "" {
		return nil, fmt.Errorf("el email es requerido")
	}
	if !emailRegex.MatchString(email) {
		return nil, fmt.Errorf("el email no es válido")
	}

	return s.storage.Update(ctx, id, name, email, active)
}

// Delete elimina un registro.
func (s *NotificationService) Delete(ctx context.Context, id int64) error {
	return s.storage.Delete(ctx, id)
}
