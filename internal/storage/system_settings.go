package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

type SystemSettingsStorage struct {
	db *sql.DB
}

func NewSystemSettingsStorage(db *sql.DB) *SystemSettingsStorage {
	return &SystemSettingsStorage{db: db}
}

func (s *SystemSettingsStorage) Get(ctx context.Context, key string) (string, error) {
	var value string
	if err := s.db.QueryRowContext(ctx,
		"SELECT value FROM system_settings WHERE key = ?", key,
	).Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("obtener system setting %s: %w", key, err)
	}
	return value, nil
}

func (s *SystemSettingsStorage) Set(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO system_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP",
		key, value,
	)
	if err != nil {
		return fmt.Errorf("guardar system setting %s: %w", key, err)
	}
	return nil
}

func (s *SystemSettingsStorage) GetSetting(ctx context.Context, key string) (*models.SystemSetting, error) {
	var st models.SystemSetting
	if err := s.db.QueryRowContext(ctx,
		"SELECT id, key, value, created_at, updated_at FROM system_settings WHERE key = ?", key,
	).Scan(&st.ID, &st.Key, &st.Value, &st.CreatedAt, &st.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("obtener system setting %s: %w", key, err)
	}
	return &st, nil
}
