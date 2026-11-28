package workrequest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
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
	w.reported_at, w.maintenance_id, w.converted_at, w.client_request_id,
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
		&item.ClientRequestID, &item.CreatedAt, &item.UpdatedAt, &item.AttachmentCount,
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

// Create inserta un reporte y devuelve su ID. Si el client_request_id ya
// existe en la vivienda, no escribe y devuelve ErrDuplicateClientRequest.
func (s *Storage) Create(ctx context.Context, homeID int64, input CreateInput) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO home_work_requests
			(home_id, title, description, category, priority, status, reported_at, client_request_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT DO NOTHING
	`, homeID, input.Title, input.Description, input.Category, input.Priority, StatusReported, input.ReportedAt, input.ClientRequestID)
	if err != nil {
		return 0, fmt.Errorf("crear reporte: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("filas creadas: %w", err)
	}
	if affected == 0 {
		if input.ClientRequestID != "" {
			return 0, ErrDuplicateClientRequest
		}
		return 0, fmt.Errorf("crear reporte: no se insertó la fila")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("id de reporte: %w", err)
	}
	return id, nil
}

// FindByClientRequestID busca un reporte por su clave de idempotencia.
func (s *Storage) FindByClientRequestID(ctx context.Context, homeID int64, clientRequestID string) (*WorkRequest, error) {
	if clientRequestID == "" {
		return nil, nil
	}
	row := s.db.QueryRowContext(ctx,
		`SELECT `+workRequestColumns+workRequestFrom+` WHERE w.home_id = ? AND w.client_request_id = ?`,
		homeID, clientRequestID)
	return scanWorkRequest(row)
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

const attachmentColumns = `id, work_request_id, media_type, file_name, mime_type, size_bytes, sha256, created_at`

func scanAttachment(row interface{ Scan(...any) error }) (*Attachment, error) {
	var item Attachment
	err := row.Scan(&item.ID, &item.WorkRequestID, &item.MediaType, &item.FileName,
		&item.MimeType, &item.SizeBytes, &item.SHA256, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear adjunto: %w", err)
	}
	return &item, nil
}

// ListAttachments devuelve la evidencia de un reporte.
func (s *Storage) ListAttachments(ctx context.Context, workRequestID int64) ([]Attachment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+attachmentColumns+`
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
		item, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		if item != nil {
			items = append(items, *item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer adjuntos: %w", err)
	}
	return items, nil
}

// GetAttachment devuelve un adjunto del reporte.
func (s *Storage) GetAttachment(ctx context.Context, workRequestID, id int64) (*Attachment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+attachmentColumns+`
		FROM home_work_request_attachments
		WHERE work_request_id = ? AND id = ?
	`, workRequestID, id)
	return scanAttachment(row)
}

// GetAttachmentBySHA256 busca un adjunto por su hash dentro del reporte.
func (s *Storage) GetAttachmentBySHA256(ctx context.Context, workRequestID int64, sha256 string) (*Attachment, error) {
	if sha256 == "" {
		return nil, nil
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT `+attachmentColumns+`
		FROM home_work_request_attachments
		WHERE work_request_id = ? AND sha256 = ?
	`, workRequestID, sha256)
	return scanAttachment(row)
}

// CreateEvidenceAttachment inserta evidencia en una transacción BEGIN IMMEDIATE:
// valida el cupo global y la unicidad de sha256 antes de escribir, cerrando la
// carrera entre cargas concurrentes. Si el hash ya existe devuelve el adjunto
// existente sin insertar (deduplicación).
func (s *Storage) CreateEvidenceAttachment(ctx context.Context, workRequestID int64, mediaType, fileName, mimeType string, size int64, sha string) (int64, *Attachment, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("conexión para adjunto: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "PRAGMA busy_timeout = 5000"); err != nil {
		return 0, nil, fmt.Errorf("configurar espera de SQLite: %w", err)
	}
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return 0, nil, fmt.Errorf("iniciar transacción de adjunto: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	if sha != "" {
		existing, err := scanAttachment(conn.QueryRowContext(ctx, `
			SELECT `+attachmentColumns+`
			FROM home_work_request_attachments
			WHERE work_request_id = ? AND sha256 = ?
		`, workRequestID, sha))
		if err != nil {
			return 0, nil, err
		}
		if existing != nil {
			return 0, existing, nil
		}
	}

	var count int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM home_work_request_attachments WHERE work_request_id = ?`,
		workRequestID).Scan(&count); err != nil {
		return 0, nil, fmt.Errorf("contar adjuntos: %w", err)
	}
	if count >= MaxAttachmentsPerRequest {
		return 0, nil, invalid("attachment_limit", "El reporte alcanzó el máximo de 10 adjuntos")
	}

	res, err := conn.ExecContext(ctx, `
		INSERT INTO home_work_request_attachments
			(work_request_id, media_type, file_name, mime_type, size_bytes, sha256)
		VALUES (?, ?, ?, ?, ?, ?)
	`, workRequestID, mediaType, fileName, mimeType, size, sha)
	if err != nil {
		if sha != "" && strings.Contains(err.Error(), "UNIQUE constraint failed") {
			existing, lookupErr := s.GetAttachmentBySHA256(ctx, workRequestID, sha)
			if lookupErr == nil && existing != nil {
				return 0, existing, nil
			}
		}
		return 0, nil, fmt.Errorf("crear adjunto: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, nil, fmt.Errorf("id de adjunto: %w", err)
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return 0, nil, fmt.Errorf("confirmar adjunto: %w", err)
	}
	committed = true
	return id, nil, nil
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

// CreateUploadToken persiste la fila efímera de un jti firmado.
func (s *Storage) CreateUploadToken(ctx context.Context, token UploadToken) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO work_request_upload_tokens
			(jti, work_request_id, max_files, files_used, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, token.JTI, token.WorkRequestID, token.MaxFiles, token.FilesUsed,
		token.ExpiresAt.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return fmt.Errorf("crear token de carga: %w", err)
	}
	return nil
}

// GetUploadToken devuelve la fila de un jti.
func (s *Storage) GetUploadToken(ctx context.Context, jti string) (*UploadToken, error) {
	var token UploadToken
	var expires time.Time
	err := s.db.QueryRowContext(ctx, `
		SELECT jti, work_request_id, max_files, files_used, expires_at
		FROM work_request_upload_tokens
		WHERE jti = ?
	`, jti).Scan(&token.JTI, &token.WorkRequestID, &token.MaxFiles, &token.FilesUsed, &expires)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("leer token de carga: %w", err)
	}
	token.ExpiresAt = expires.UTC()
	return &token, nil
}

// ConsumeUploadSlot incrementa files_used solo si queda cupo en el token.
func (s *Storage) ConsumeUploadSlot(ctx context.Context, jti string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE work_request_upload_tokens
		SET files_used = files_used + 1
		WHERE jti = ? AND files_used < max_files
	`, jti)
	if err != nil {
		return false, fmt.Errorf("consumir cupo del token: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("filas del token: %w", err)
	}
	return affected > 0, nil
}

// ReleaseUploadSlot devuelve un cupo consumido por un archivo que no se guardó.
func (s *Storage) ReleaseUploadSlot(ctx context.Context, jti string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE work_request_upload_tokens
		SET files_used = CASE WHEN files_used > 0 THEN files_used - 1 ELSE 0 END
		WHERE jti = ?
	`, jti)
	if err != nil {
		return fmt.Errorf("liberar cupo del token: %w", err)
	}
	return nil
}

// DeleteExpiredUploadTokens limpia filas vencidas hace más de 24 horas.
func (s *Storage) DeleteExpiredUploadTokens(ctx context.Context, before time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM work_request_upload_tokens WHERE expires_at < ?`,
		before.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return fmt.Errorf("limpiar tokens vencidos: %w", err)
	}
	return nil
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
