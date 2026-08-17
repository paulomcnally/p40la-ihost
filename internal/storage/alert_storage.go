package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// AlertStorage interactúa con la tabla `alerts` (SQLite).
type AlertStorage struct {
	db *sql.DB
}

func NewAlertStorage(db *sql.DB) *AlertStorage {
	return &AlertStorage{db: db}
}

// List devuelve todas las alertas sembradas.
func (s *AlertStorage) List(ctx context.Context) ([]models.Alert, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, key, title, description, mail_enabled, voice_enabled, speech, created_at, updated_at
		 FROM alerts ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("listar alerts: %w", err)
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var a models.Alert
		if err := rows.Scan(&a.ID, &a.Key, &a.Title, &a.Description, &a.MailEnabled, &a.VoiceEnabled, &a.Speech, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear alert: %w", err)
		}
		alerts = append(alerts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar alerts: %w", err)
	}
	return alerts, nil
}

// GetByKey devuelve una alerta por su key.
func (s *AlertStorage) GetByKey(ctx context.Context, key string) (*models.Alert, error) {
	var a models.Alert
	err := s.db.QueryRowContext(ctx,
		`SELECT id, key, title, description, mail_enabled, voice_enabled, speech, created_at, updated_at
		 FROM alerts WHERE key = ?`, key,
	).Scan(&a.ID, &a.Key, &a.Title, &a.Description, &a.MailEnabled, &a.VoiceEnabled, &a.Speech, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("obtener alert %s: %w", key, err)
	}
	return &a, nil
}

// SetFlags actualiza solo los flags indicados (punteros nulos = no se tocan).
func (s *AlertStorage) SetFlags(ctx context.Context, key string, mailEnabled, voiceEnabled *bool) error {
	if mailEnabled == nil && voiceEnabled == nil {
		return nil
	}

	parts := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if mailEnabled != nil {
		parts = append(parts, "mail_enabled = ?")
		args = append(args, boolToInt(*mailEnabled))
	}
	if voiceEnabled != nil {
		parts = append(parts, "voice_enabled = ?")
		args = append(args, boolToInt(*voiceEnabled))
	}
	args = append(args, time.Now().UTC().Format("2006-01-02 15:04:05"), key)

	query := fmt.Sprintf("UPDATE alerts SET %s, updated_at = ? WHERE key = ?", strings.Join(parts, ", "))

	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("actualizar alert %s: %w", key, err)
	}
	return nil
}

// Seed inserta las alertas del catálogo que no existan aún (INSERT OR IGNORE).
// No sobrescribe los flags existentes del usuario.
func (s *AlertStorage) Seed(ctx context.Context, alerts []models.Alert) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transacción seed: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO alerts (key, title, description, mail_enabled, voice_enabled, speech, created_at, updated_at)
		 VALUES (?, ?, ?, 0, 0, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("preparar seed: %w", err)
	}
	defer stmt.Close()

	for _, a := range alerts {
		if _, err := stmt.ExecContext(ctx, a.Key, a.Title, a.Description, a.Speech, now, now); err != nil {
			return fmt.Errorf("seed alert %s: %w", a.Key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar seed: %w", err)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
