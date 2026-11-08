package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// AutoAttachmentStorage encapsula el acceso a auto_attachments (SPEC-124).
type AutoAttachmentStorage struct {
	db *sql.DB
}

// NewAutoAttachmentStorage crea un nuevo AutoAttachmentStorage.
func NewAutoAttachmentStorage(db *sql.DB) *AutoAttachmentStorage {
	return &AutoAttachmentStorage{db: db}
}

// ListByAuto devuelve todos los adjuntos de un auto.
func (s *AutoAttachmentStorage) ListByAuto(ctx context.Context, autoID int64) ([]models.AutoAttachment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, auto_id, owner_type, owner_id, file_name, size_bytes, created_at
		FROM auto_attachments
		WHERE auto_id = ?
		ORDER BY owner_type, owner_id, id
	`, autoID)
	if err != nil {
		return nil, fmt.Errorf("listar adjuntos del auto: %w", err)
	}
	defer rows.Close()
	return scanAutoAttachments(rows)
}

// ListByOwner devuelve los adjuntos de un mantenimiento o inspección.
func (s *AutoAttachmentStorage) ListByOwner(ctx context.Context, ownerType string, ownerID int64) ([]models.AutoAttachment, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, auto_id, owner_type, owner_id, file_name, size_bytes, created_at
		FROM auto_attachments
		WHERE owner_type = ? AND owner_id = ?
		ORDER BY id
	`, ownerType, ownerID)
	if err != nil {
		return nil, fmt.Errorf("listar adjuntos: %w", err)
	}
	defer rows.Close()
	return scanAutoAttachments(rows)
}

// GetByID busca un adjunto.
func (s *AutoAttachmentStorage) GetByID(ctx context.Context, id int64) (*models.AutoAttachment, error) {
	var att models.AutoAttachment
	err := s.db.QueryRowContext(ctx, `
		SELECT id, auto_id, owner_type, owner_id, file_name, size_bytes, created_at
		FROM auto_attachments
		WHERE id = ?
	`, id).Scan(&att.ID, &att.AutoID, &att.OwnerType, &att.OwnerID, &att.FileName, &att.SizeBytes, &att.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("obtener adjunto: %w", err)
	}
	return &att, nil
}

// Create inserta los metadatos de un adjunto.
func (s *AutoAttachmentStorage) Create(ctx context.Context, autoID int64, ownerType string, ownerID int64, fileName string, sizeBytes int64) (*models.AutoAttachment, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO auto_attachments (auto_id, owner_type, owner_id, file_name, size_bytes)
		VALUES (?, ?, ?, ?, ?)
	`, autoID, ownerType, ownerID, fileName, sizeBytes)
	if err != nil {
		return nil, fmt.Errorf("crear adjunto: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de adjunto: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Delete elimina los metadatos de un adjunto.
func (s *AutoAttachmentStorage) Delete(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM auto_attachments WHERE id = ?`, id); err != nil {
		return fmt.Errorf("eliminar adjunto: %w", err)
	}
	return nil
}

// CountByOwner cuenta los adjuntos de un registro.
func (s *AutoAttachmentStorage) CountByOwner(ctx context.Context, ownerType string, ownerID int64) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM auto_attachments WHERE owner_type = ? AND owner_id = ?
	`, ownerType, ownerID).Scan(&count); err != nil {
		return 0, fmt.Errorf("contar adjuntos: %w", err)
	}
	return count, nil
}

// OwnerBelongsToAuto verifica que el registro exista y pertenezca al auto.
func (s *AutoAttachmentStorage) OwnerBelongsToAuto(ctx context.Context, ownerType string, ownerID, autoID int64) (bool, error) {
	var query string
	switch ownerType {
	case models.AutoAttachmentOwnerMaintenance:
		query = `SELECT COUNT(*) FROM auto_maintenances WHERE id = ? AND auto_id = ?`
	case models.AutoAttachmentOwnerInspection:
		query = `SELECT COUNT(*) FROM auto_inspections WHERE id = ? AND auto_id = ?`
	default:
		return false, nil
	}

	var count int64
	if err := s.db.QueryRowContext(ctx, query, ownerID, autoID).Scan(&count); err != nil {
		return false, fmt.Errorf("verificar propietario del adjunto: %w", err)
	}
	return count > 0, nil
}

func scanAutoAttachments(rows *sql.Rows) ([]models.AutoAttachment, error) {
	items := make([]models.AutoAttachment, 0)
	for rows.Next() {
		var att models.AutoAttachment
		if err := rows.Scan(&att.ID, &att.AutoID, &att.OwnerType, &att.OwnerID, &att.FileName, &att.SizeBytes, &att.CreatedAt); err != nil {
			return nil, fmt.Errorf("escanear adjunto: %w", err)
		}
		items = append(items, att)
	}
	return items, rows.Err()
}
