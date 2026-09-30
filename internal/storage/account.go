package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// AccountStorage encapsula el acceso a la tabla accounts.
type AccountStorage struct {
	db *sql.DB
}

// NewAccountStorage crea un nuevo AccountStorage.
func NewAccountStorage(db *sql.DB) *AccountStorage {
	return &AccountStorage{db: db}
}

// Balance: starting_balance + transacciones (inflow - outflow) + neto de
// transferencias (SPEC-099). Se calcula en la consulta, no se persiste.
const accountBalanceExpr = `
	a.starting_balance
	+ COALESCE((SELECT SUM(t.inflow - t.outflow)
	            FROM transactions t
	            WHERE t.account_id = a.id AND t.deleted_at IS NULL), 0)
	+ COALESCE((SELECT SUM(CASE WHEN tr.to_account_id = a.id THEN tr.amount
	                            WHEN tr.from_account_id = a.id THEN -tr.amount END)
	            FROM transfers tr
	            WHERE (tr.from_account_id = a.id OR tr.to_account_id = a.id)
	              AND tr.deleted_at IS NULL), 0)
`

const accountColumns = `
	a.id, a.name, a.type, a.currency_id, COALESCE(c.code, ''),
	a.starting_balance, ` + accountBalanceExpr + ` AS balance,
	a.deleted_at, a.created_at, a.updated_at
`

// List devuelve todas las cuentas no eliminadas con su balance calculado.
func (s *AccountStorage) List(ctx context.Context) ([]models.Account, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+accountColumns+`
		FROM accounts a
		LEFT JOIN currencies c ON c.id = a.currency_id
		WHERE a.deleted_at IS NULL
		ORDER BY a.name
	`)
	if err != nil {
		return nil, fmt.Errorf("listar cuentas: %w", err)
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var a models.Account
		var deletedAt sql.NullTime
		var code sql.NullString
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.CurrencyID, &code,
			&a.StartingBalance, &a.Balance, &deletedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear cuenta: %w", err)
		}
		a.CurrencyCode = code.String
		if deletedAt.Valid {
			a.DeletedAt = &deletedAt.Time
		}
		accounts = append(accounts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return accounts, nil
}

// GetByID busca una cuenta por ID con su balance calculado.
func (s *AccountStorage) GetByID(ctx context.Context, id int64) (*models.Account, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+accountColumns+`
		FROM accounts a
		LEFT JOIN currencies c ON c.id = a.currency_id
		WHERE a.id = ? AND a.deleted_at IS NULL
	`, id)
	return scanAccount(row)
}

// Create inserta una nueva cuenta.
func (s *AccountStorage) Create(ctx context.Context, account *models.Account) (*models.Account, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (name, type, currency_id, starting_balance)
		VALUES (?, ?, ?, ?)
	`, account.Name, account.Type, account.CurrencyID, account.StartingBalance)
	if err != nil {
		return nil, fmt.Errorf("insertar cuenta: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de cuenta: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza una cuenta existente.
func (s *AccountStorage) Update(ctx context.Context, account *models.Account) (*models.Account, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE accounts
		SET name = ?, type = ?, currency_id = ?, starting_balance = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, account.Name, account.Type, account.CurrencyID, account.StartingBalance, account.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar cuenta: %w", err)
	}
	return s.GetByID(ctx, account.ID)
}

// SoftDelete marca una cuenta como eliminada.
func (s *AccountStorage) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE accounts SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar cuenta: %w", err)
	}
	return nil
}

func scanAccount(row *sql.Row) (*models.Account, error) {
	var a models.Account
	var deletedAt sql.NullTime
	var code sql.NullString
	if err := row.Scan(&a.ID, &a.Name, &a.Type, &a.CurrencyID, &code,
		&a.StartingBalance, &a.Balance, &deletedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear cuenta: %w", err)
	}
	a.CurrencyCode = code.String
	if deletedAt.Valid {
		a.DeletedAt = &deletedAt.Time
	}
	return &a, nil
}
