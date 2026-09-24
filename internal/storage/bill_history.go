package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// BillHistoryStorage encapsula el acceso a la tabla bill_history (SPEC-070).
type BillHistoryStorage struct {
	db *sql.DB
}

// NewBillHistoryStorage crea un nuevo BillHistoryStorage.
func NewBillHistoryStorage(db *sql.DB) *BillHistoryStorage {
	return &BillHistoryStorage{db: db}
}

// Record persiste un evento de auditoría de una factura (SPEC-070).
func (s *BillHistoryStorage) Record(ctx context.Context, event *models.BillHistory) (*models.BillHistory, error) {
	var changes []byte
	if len(event.Changes) > 0 {
		var err error
		changes, err = json.Marshal(event.Changes)
		if err != nil {
			return nil, fmt.Errorf("serializar cambios de historial: %w", err)
		}
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO bill_history (bill_id, action, source, changes)
		VALUES (?, ?, ?, ?)
	`, event.BillID, event.Action, event.Source, nullableString(changes))
	if err != nil {
		return nil, fmt.Errorf("insertar historial de factura: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de historial: %w", err)
	}
	event.ID = id
	return event, nil
}

// ListByBill devuelve los eventos de auditoría de una factura ordenados por
// fecha descendente (más reciente primero) (SPEC-070).
func (s *BillHistoryStorage) ListByBill(ctx context.Context, billID int64) ([]models.BillHistory, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, bill_id, action, source, changes, created_at
		FROM bill_history
		WHERE bill_id = ?
		ORDER BY created_at DESC, id DESC
	`, billID)
	if err != nil {
		return nil, fmt.Errorf("listar historial de factura: %w", err)
	}
	defer rows.Close()

	var history []models.BillHistory
	for rows.Next() {
		var h models.BillHistory
		var changes sql.NullString
		if err := rows.Scan(&h.ID, &h.BillID, &h.Action, &h.Source, &changes, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("escanear historial de factura: %w", err)
		}
		if changes.Valid && changes.String != "" {
			if err := json.Unmarshal([]byte(changes.String), &h.Changes); err != nil {
				return nil, fmt.Errorf("parsear cambios de historial: %w", err)
			}
		}
		history = append(history, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar historial de facturas: %w", err)
	}
	return history, nil
}

func nullableString(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}
