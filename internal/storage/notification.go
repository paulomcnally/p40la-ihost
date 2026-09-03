package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// ErrDuplicateEmail indica que el email ya existe en la tabla notifications.
var ErrDuplicateEmail = fmt.Errorf("el email ya está registrado")

// NotificationStorage encapsula el acceso a la tabla notifications.
type NotificationStorage struct {
	db *sql.DB
}

// NewNotificationStorage crea un nuevo NotificationStorage.
func NewNotificationStorage(db *sql.DB) *NotificationStorage {
	return &NotificationStorage{db: db}
}

// List devuelve todos los registros de notificación, más recientes primero.
func (s *NotificationStorage) List(ctx context.Context) ([]models.Notification, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, email, active, created_at, updated_at
		FROM notifications
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listar notificaciones: %w", err)
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.Name, &n.Email, &n.Active, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear notificación: %w", err)
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return notifications, nil
}

// GetByID busca un registro por su ID.
func (s *NotificationStorage) GetByID(ctx context.Context, id int64) (*models.Notification, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, email, active, created_at, updated_at
		FROM notifications
		WHERE id = ?
	`, id)
	return scanNotification(row)
}

// Create inserta un nuevo registro.
func (s *NotificationStorage) Create(ctx context.Context, name, email string, active bool) (*models.Notification, error) {
	activeInt := 0
	if active {
		activeInt = 1
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (name, email, active) VALUES (?, ?, ?)
	`, name, email, activeInt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("insertar notificación: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de notificación: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza un registro existente.
func (s *NotificationStorage) Update(ctx context.Context, id int64, name, email string, active bool) (*models.Notification, error) {
	activeInt := 0
	if active {
		activeInt = 1
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE notifications
		SET name = ?, email = ?, active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, email, activeInt, id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("actualizar notificación: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Delete elimina un registro.
func (s *NotificationStorage) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM notifications WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar notificación: %w", err)
	}
	return nil
}

func scanNotification(row *sql.Row) (*models.Notification, error) {
	var n models.Notification
	if err := row.Scan(&n.ID, &n.Name, &n.Email, &n.Active, &n.CreatedAt, &n.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear notificación: %w", err)
	}
	return &n, nil
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
