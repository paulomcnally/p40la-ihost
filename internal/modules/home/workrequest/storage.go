package workrequest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Storage es el único punto con SQL del módulo (ARCH-LAYERS).
type Storage struct {
	db *sql.DB
}

// NewStorage crea el storage del módulo.
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

const workRequestColumns = `
	w.id, w.home_id, w.title, w.description, w.category, w.priority, w.status,
	w.reported_at, w.maintenance_id, w.converted_at,
	w.created_at, w.updated_at,
	(SELECT COUNT(*) FROM home_work_request_attachments a WHERE a.work_request_id = w.id)
`

const workRequestFrom = ` FROM home_work_requests w `

func scanWorkRequest(row interface{ Scan(...any) error }) (*WorkRequest, error) {
	var item WorkRequest
	var maintenanceID sql.NullInt64
	var convertedAt sql.NullString
	err := row.Scan(
		&item.ID, &item.HomeID, &item.Title, &item.Description, &item.Category,
		&item.Priority, &item.Status, &item.ReportedAt, &maintenanceID, &convertedAt,
		&item.CreatedAt, &item.UpdatedAt, &item.AttachmentCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear reporte: %w", err)
	}
	if maintenanceID.Valid {
		item.MaintenanceID = &maintenanceID.Int64
	}
	if convertedAt.Valid {
		item.ConvertedAt = &convertedAt.String
	}
	return &item, nil
}

// HomeExists verifica que la vivienda exista y no esté eliminada.
func (s *Storage) HomeExists(ctx context.Context, homeID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM homes WHERE id = ? AND deleted_at IS NULL`, homeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("verificar vivienda: %w", err)
	}
	return count > 0, nil
}

func filterClause(homeID int64, filter Filter) (string, []any) {
	where := []string{"w.home_id = ?"}
	args := []any{homeID}
	if filter.Status != "" {
		where = append(where, "w.status = ?")
		args = append(args, filter.Status)
	}
	if filter.Category != "" {
		where = append(where, "w.category = ?")
		args = append(args, filter.Category)
	}
	return " WHERE " + strings.Join(where, " AND "), args
}

// List devuelve los reportes de una vivienda con el conteo de adjuntos.
func (s *Storage) List(ctx context.Context, homeID int64, filter Filter) ([]WorkRequest, error) {
	clause, args := filterClause(homeID, filter)
	query := `SELECT ` + workRequestColumns + workRequestFrom + clause + `
		ORDER BY w.reported_at DESC, w.id DESC
	`
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listar reportes: %w", err)
	}
	defer rows.Close()

	items := make([]WorkRequest, 0)
	for rows.Next() {
		item, err := scanWorkRequest(rows)
		if err != nil {
			return nil, err
		}
		if item != nil {
			item.Attachments = []Attachment{}
			items = append(items, *item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer reportes: %w", err)
	}
	return items, nil
}

// PendingCount cuenta los reportes en estado reported de una vivienda.
func (s *Storage) PendingCount(ctx context.Context, homeID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM home_work_requests WHERE home_id = ? AND status = ?`,
		homeID, StatusReported).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("contar reportes pendientes: %w", err)
	}
	return count, nil
}

// Get devuelve un reporte por vivienda.
func (s *Storage) Get(ctx context.Context, homeID, id int64) (*WorkRequest, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+workRequestColumns+workRequestFrom+` WHERE w.home_id = ? AND w.id = ?`,
		homeID, id)
	return scanWorkRequest(row)
}

// Create inserta un reporte y devuelve su ID.
func (s *Storage) Create(ctx context.Context, homeID int64, input Input) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO home_work_requests
			(home_id, title, description, category, priority, status, reported_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, homeID, input.Title, input.Description, input.Category, input.Priority, StatusReported, input.ReportedAt)
	if err != nil {
		return 0, fmt.Errorf("crear reporte: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("id de reporte: %w", err)
	}
	return id, nil
}

// Update actualiza los campos editables de un reporte.
func (s *Storage) Update(ctx context.Context, homeID, id int64, input Input) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE home_work_requests
		SET title = ?, description = ?, category = ?, priority = ?, reported_at = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE home_id = ? AND id = ?
	`, input.Title, input.Description, input.Category, input.Priority, input.ReportedAt, homeID, id)
	if err != nil {
		return false, fmt.Errorf("actualizar reporte: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("filas actualizadas: %w", err)
	}
	return affected > 0, nil
}

// Delete elimina un reporte (los adjuntos caen por cascada).
func (s *Storage) Delete(ctx context.Context, homeID, id int64) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM home_work_requests WHERE home_id = ? AND id = ?`, homeID, id)
	if err != nil {
		return false, fmt.Errorf("eliminar reporte: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("filas eliminadas: %w", err)
	}
	return affected > 0, nil
}

// SetConverted marca un reporte como convertido y lo enlaza a la reparación.
func (s *Storage) SetConverted(ctx context.Context, homeID, id, maintenanceID int64, convertedAt string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE home_work_requests
		SET status = ?, maintenance_id = ?, converted_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE home_id = ? AND id = ? AND status = ?
	`, StatusConverted, maintenanceID, convertedAt, homeID, id, StatusReported)
	if err != nil {
		return false, fmt.Errorf("convertir reporte: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("filas convertidas: %w", err)
	}
	return affected > 0, nil
}

// SetStatus cambia el estado de un reporte.
func (s *Storage) SetStatus(ctx context.Context, homeID, id int64, status string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE home_work_requests
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE home_id = ? AND id = ?
	`, status, homeID, id)
	if err != nil {
		return false, fmt.Errorf("cambiar estado del reporte: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("filas actualizadas: %w", err)
	}
	return affected > 0, nil
}

// ListAttachments devuelve la evidencia de un reporte.
func (s *Storage) ListAttachments(ctx context.Context, workRequestID int64) ([]Attachment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, work_request_id, media_type, file_name, mime_type, size_bytes, created_at
		FROM home_work_request_attachments
		WHERE work_request_id = ?
		ORDER BY id
	`, workRequestID)
	if err != nil {
		return nil, fmt.Errorf("listar adjuntos: %w", err)
	}
	defer rows.Close()

	items := make([]Attachment, 0)
	for rows.Next() {
		var item Attachment
		if err := rows.Scan(&item.ID, &item.WorkRequestID, &item.MediaType, &item.FileName,
			&item.MimeType, &item.SizeBytes, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("escanear adjunto: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer adjuntos: %w", err)
	}
	return items, nil
}

// GetAttachment devuelve un adjunto del reporte.
func (s *Storage) GetAttachment(ctx context.Context, workRequestID, id int64) (*Attachment, error) {
	var item Attachment
	err := s.db.QueryRowContext(ctx, `
		SELECT id, work_request_id, media_type, file_name, mime_type, size_bytes, created_at
		FROM home_work_request_attachments
		WHERE work_request_id = ? AND id = ?
	`, workRequestID, id).Scan(&item.ID, &item.WorkRequestID, &item.MediaType,
		&item.FileName, &item.MimeType, &item.SizeBytes, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("leer adjunto: %w", err)
	}
	return &item, nil
}

// CountAttachments cuenta la evidencia de un reporte.
func (s *Storage) CountAttachments(ctx context.Context, workRequestID int64) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM home_work_request_attachments WHERE work_request_id = ?`,
		workRequestID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("contar adjuntos: %w", err)
	}
	return count, nil
}

// CreateAttachment inserta los metadatos de un adjunto.
func (s *Storage) CreateAttachment(ctx context.Context, workRequestID int64, mediaType, fileName, mimeType string, size int64) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO home_work_request_attachments
			(work_request_id, media_type, file_name, mime_type, size_bytes)
		VALUES (?, ?, ?, ?, ?)
	`, workRequestID, mediaType, fileName, mimeType, size)
	if err != nil {
		return 0, fmt.Errorf("crear adjunto: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("id de adjunto: %w", err)
	}
	return id, nil
}

// DeleteAttachment elimina los metadatos de un adjunto.
func (s *Storage) DeleteAttachment(ctx context.Context, workRequestID, id int64) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM home_work_request_attachments WHERE work_request_id = ? AND id = ?`,
		workRequestID, id)
	if err != nil {
		return false, fmt.Errorf("eliminar adjunto: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("filas eliminadas: %w", err)
	}
	return affected > 0, nil
}
