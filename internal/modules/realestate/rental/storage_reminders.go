package rental

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const reminderColumns = `
	id, home_id, lease_id, kind, title, description, schedule_type, interval_n,
	anchor_date, next_due_at, lead_days, notify_roles, extra_emails, enabled,
	created_at, updated_at
`

func scanReminder(row interface{ Scan(...any) error }) (*Reminder, error) {
	var r Reminder
	var homeID, leaseID sql.NullInt64
	var enabled int
	err := row.Scan(&r.ID, &homeID, &leaseID, &r.Kind, &r.Title, &r.Description,
		&r.ScheduleType, &r.IntervalN, &r.AnchorDate, &r.NextDueAt, &r.LeadDays,
		&r.NotifyRoles, &r.ExtraEmails, &enabled, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear recordatorio: %w", err)
	}
	if homeID.Valid {
		r.HomeID = &homeID.Int64
	}
	if leaseID.Valid {
		r.LeaseID = &leaseID.Int64
	}
	r.Enabled = enabled == 1
	return &r, nil
}

// ListReminders lista recordatorios por casa o contrato.
func (s *Storage) ListReminders(ctx context.Context, homeID, leaseID int64) ([]Reminder, error) {
	q := `SELECT ` + reminderColumns + ` FROM rental_reminders WHERE 1=1`
	args := []any{}
	if homeID > 0 {
		q += ` AND home_id = ?`
		args = append(args, homeID)
	}
	if leaseID > 0 {
		q += ` AND lease_id = ?`
		args = append(args, leaseID)
	}
	q += ` ORDER BY next_due_at ASC, id ASC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listar recordatorios: %w", err)
	}
	defer rows.Close()
	var out []Reminder
	for rows.Next() {
		r, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// GetReminder devuelve un recordatorio o nil.
func (s *Storage) GetReminder(ctx context.Context, id int64) (*Reminder, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+reminderColumns+` FROM rental_reminders WHERE id = ?`, id)
	return scanReminder(row)
}

// CreateReminder inserta un recordatorio.
func (s *Storage) CreateReminder(ctx context.Context, r *Reminder) (*Reminder, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO rental_reminders (
			home_id, lease_id, kind, title, description, schedule_type, interval_n,
			anchor_date, next_due_at, lead_days, notify_roles, extra_emails, enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, nullableID(r.HomeID), nullableID(r.LeaseID), r.Kind, r.Title, r.Description,
		r.ScheduleType, r.IntervalN, r.AnchorDate, r.NextDueAt, r.LeadDays,
		r.NotifyRoles, r.ExtraEmails, boolInt(r.Enabled))
	if err != nil {
		return nil, fmt.Errorf("insertar recordatorio: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetReminder(ctx, id)
}

// UpdateReminder actualiza un recordatorio.
func (s *Storage) UpdateReminder(ctx context.Context, r *Reminder) (*Reminder, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rental_reminders SET
			kind = ?, title = ?, description = ?, schedule_type = ?, interval_n = ?,
			anchor_date = ?, next_due_at = ?, lead_days = ?, notify_roles = ?,
			extra_emails = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, r.Kind, r.Title, r.Description, r.ScheduleType, r.IntervalN,
		r.AnchorDate, r.NextDueAt, r.LeadDays, r.NotifyRoles, r.ExtraEmails,
		boolInt(r.Enabled), r.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar recordatorio: %w", err)
	}
	return s.GetReminder(ctx, r.ID)
}

// DeleteReminder elimina un recordatorio.
func (s *Storage) DeleteReminder(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM rental_reminders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("eliminar recordatorio: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrReminderNotFound
	}
	return nil
}

// ListEnabledReminders devuelve los recordatorios activos (para el scheduler).
func (s *Storage) ListEnabledReminders(ctx context.Context) ([]Reminder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+reminderColumns+` FROM rental_reminders WHERE enabled = 1 ORDER BY next_due_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("listar recordatorios activos: %w", err)
	}
	defer rows.Close()
	var out []Reminder
	for rows.Next() {
		r, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// AdvanceReminder actualiza la próxima fecha del recordatorio.
func (s *Storage) AdvanceReminder(ctx context.Context, id int64, nextDue string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rental_reminders SET next_due_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, nextDue, id)
	if err != nil {
		return fmt.Errorf("avanzar recordatorio: %w", err)
	}
	return nil
}

// ReminderEventExists indica si ya se registró una ocurrencia.
func (s *Storage) ReminderEventExists(ctx context.Context, reminderID int64, occurrence string) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM rental_reminder_events WHERE reminder_id = ? AND occurrence_key = ?
	`, reminderID, occurrence).Scan(&n); err != nil {
		return false, fmt.Errorf("verificar evento de recordatorio: %w", err)
	}
	return n > 0, nil
}

// InsertReminderEvent registra una ocurrencia enviada o fallida.
func (s *Storage) InsertReminderEvent(ctx context.Context, reminderID int64, occurrence, dueAt, result, errorCode string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO rental_reminder_events (reminder_id, occurrence_key, due_at, sent_at, result, error_code)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?, ?)
		ON CONFLICT(reminder_id, occurrence_key) DO NOTHING
	`, reminderID, occurrence, dueAt, result, errorCode)
	if err != nil {
		return fmt.Errorf("registrar evento de recordatorio: %w", err)
	}
	return nil
}

// EmailLogExists indica si ya se envió un correo con esa clave.
func (s *Storage) EmailLogExists(ctx context.Context, kind string, ownerID int64, occurrence string) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM rental_email_log WHERE kind = ? AND owner_id = ? AND occurrence_key = ?
	`, kind, ownerID, occurrence).Scan(&n); err != nil {
		return false, fmt.Errorf("verificar envío de email: %w", err)
	}
	return n > 0, nil
}

// InsertEmailLog registra un envío de email.
func (s *Storage) InsertEmailLog(ctx context.Context, kind string, ownerID int64, occurrence string, recipients int, roles, result, errorCode string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO rental_email_log (kind, owner_id, occurrence_key, recipients_count, recipient_roles, result, error_code)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(kind, owner_id, occurrence_key) DO NOTHING
	`, kind, ownerID, occurrence, recipients, roles, result, errorCode)
	if err != nil {
		return fmt.Errorf("registrar email de alquiler: %w", err)
	}
	return nil
}

// RemindersForHome devuelve recordatorios activos de una casa, incluyendo los de sus contratos.
func (s *Storage) RemindersForHome(ctx context.Context, homeID int64) ([]Reminder, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+reminderColumns+` FROM rental_reminders
		WHERE enabled = 1 AND (
			home_id = ? OR lease_id IN (SELECT id FROM rental_leases WHERE home_id = ? AND deleted_at IS NULL)
		)
		ORDER BY next_due_at ASC
	`, homeID, homeID)
	if err != nil {
		return nil, fmt.Errorf("listar recordatorios de vivienda: %w", err)
	}
	defer rows.Close()
	var out []Reminder
	for rows.Next() {
		r, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}
