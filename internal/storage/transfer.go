package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// TransferStorage encapsula el acceso a la tabla transfers (SPEC-099).
type TransferStorage struct {
	db *sql.DB
}

// NewTransferStorage crea un nuevo TransferStorage.
func NewTransferStorage(db *sql.DB) *TransferStorage {
	return &TransferStorage{db: db}
}

const transferColumns = `
	t.id, t.from_account_id, COALESCE(fa.name, ''), t.to_account_id, COALESCE(ta.name, ''),
	t.currency_id, COALESCE(c.code, ''), t.date, t.payee, t.memo,
	t.amount, t.cleared, t.deleted_at, t.created_at, t.updated_at
`

const transferJoins = `
	FROM transfers t
	LEFT JOIN accounts fa ON fa.id = t.from_account_id
	LEFT JOIN accounts ta ON ta.id = t.to_account_id
	LEFT JOIN currencies c ON c.id = t.currency_id
`

// ListByMonth devuelve las transferencias de un mes (rango de fechas) no eliminadas.
func (s *TransferStorage) ListByMonth(ctx context.Context, year, month int) ([]models.Transfer, error) {
	start := fmt.Sprintf("%04d-%02d-01", year, month)
	end := fmt.Sprintf("%04d-%02d-31", year, month)
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+transferColumns+`
		`+transferJoins+`
		WHERE t.deleted_at IS NULL AND t.date >= ? AND t.date <= ?
		ORDER BY t.date DESC, t.id DESC
	`, start, end)
	if err != nil {
		return nil, fmt.Errorf("listar transferencias del mes: %w", err)
	}
	defer rows.Close()
	return scanTransfers(rows)
}

// List devuelve todas las transferencias no eliminadas.
func (s *TransferStorage) List(ctx context.Context) ([]models.Transfer, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+transferColumns+`
		`+transferJoins+`
		WHERE t.deleted_at IS NULL
		ORDER BY t.date DESC, t.id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listar transferencias: %w", err)
	}
	defer rows.Close()
	return scanTransfers(rows)
}

// GetByID busca una transferencia por ID.
func (s *TransferStorage) GetByID(ctx context.Context, id int64) (*models.Transfer, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+transferColumns+`
		`+transferJoins+`
		WHERE t.id = ? AND t.deleted_at IS NULL
	`, id)
	return scanTransfer(row)
}

// Create inserta una nueva transferencia.
func (s *TransferStorage) Create(ctx context.Context, t *models.Transfer) (*models.Transfer, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO transfers (from_account_id, to_account_id, currency_id, date,
		                       payee, memo, amount, cleared)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, t.FromAccountID, t.ToAccountID, t.CurrencyID, t.Date, t.Payee,
		t.Memo, t.Amount, boolToInt(t.Cleared))
	if err != nil {
		return nil, fmt.Errorf("insertar transferencia: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de transferencia: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza una transferencia existente.
func (s *TransferStorage) Update(ctx context.Context, t *models.Transfer) (*models.Transfer, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE transfers
		SET from_account_id = ?, to_account_id = ?, currency_id = ?, date = ?,
		    payee = ?, memo = ?, amount = ?, cleared = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, t.FromAccountID, t.ToAccountID, t.CurrencyID, t.Date, t.Payee,
		t.Memo, t.Amount, boolToInt(t.Cleared), t.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar transferencia: %w", err)
	}
	return s.GetByID(ctx, t.ID)
}

// SoftDelete marca una transferencia como eliminada.
func (s *TransferStorage) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE transfers SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar transferencia: %w", err)
	}
	return nil
}

// NetByAccount devuelve el neto de transferencias por cuenta (positivo = entró
// más de lo que salió) para cuentas no eliminadas.
func (s *TransferStorage) NetByAccount(ctx context.Context) (map[int64]float64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT account_id, SUM(net) FROM (
			SELECT from_account_id AS account_id, -amount AS net
			FROM transfers WHERE deleted_at IS NULL
			UNION ALL
			SELECT to_account_id AS account_id, amount AS net
			FROM transfers WHERE deleted_at IS NULL
		)
		GROUP BY account_id
	`)
	if err != nil {
		return nil, fmt.Errorf("sumar neto de transferencias por cuenta: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]float64)
	for rows.Next() {
		var accountID int64
		var net float64
		if err := rows.Scan(&accountID, &net); err != nil {
			return nil, fmt.Errorf("escanear neto de transferencias: %w", err)
		}
		result[accountID] = net
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanTransfer(row *sql.Row) (*models.Transfer, error) {
	var t models.Transfer
	var deletedAt sql.NullTime
	var fromName, toName, code sql.NullString
	var cleared int
	if err := row.Scan(&t.ID, &t.FromAccountID, &fromName, &t.ToAccountID, &toName,
		&t.CurrencyID, &code, &t.Date, &t.Payee, &t.Memo,
		&t.Amount, &cleared, &deletedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear transferencia: %w", err)
	}
	t.FromAccountName = fromName.String
	t.ToAccountName = toName.String
	t.CurrencyCode = code.String
	t.Cleared = cleared == 1
	if deletedAt.Valid {
		t.DeletedAt = &deletedAt.Time
	}
	return &t, nil
}

func scanTransfers(rows *sql.Rows) ([]models.Transfer, error) {
	var transfers []models.Transfer
	for rows.Next() {
		var t models.Transfer
		var deletedAt sql.NullTime
		var fromName, toName, code sql.NullString
		var cleared int
		if err := rows.Scan(&t.ID, &t.FromAccountID, &fromName, &t.ToAccountID, &toName,
			&t.CurrencyID, &code, &t.Date, &t.Payee, &t.Memo,
			&t.Amount, &cleared, &deletedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear transferencia: %w", err)
		}
		t.FromAccountName = fromName.String
		t.ToAccountName = toName.String
		t.CurrencyCode = code.String
		t.Cleared = cleared == 1
		if deletedAt.Valid {
			t.DeletedAt = &deletedAt.Time
		}
		transfers = append(transfers, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return transfers, nil
}
