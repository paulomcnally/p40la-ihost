package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// TransactionStorage encapsula el acceso a la tabla transactions.
type TransactionStorage struct {
	db *sql.DB
}

// NewTransactionStorage crea un nuevo TransactionStorage.
func NewTransactionStorage(db *sql.DB) *TransactionStorage {
	return &TransactionStorage{db: db}
}

const transactionColumns = `
	t.id, t.account_id, COALESCE(a.name, ''), t.category_id, COALESCE(cat.name, ''),
	t.currency_id, COALESCE(c.code, ''), t.date, t.payee, t.memo,
	t.outflow, t.inflow, t.cleared, t.deleted_at, t.created_at, t.updated_at
`

// ListByMonth devuelve las transacciones de un mes (rango de fechas).
func (s *TransactionStorage) ListByMonth(ctx context.Context, year, month int) ([]models.Transaction, error) {
	start := fmt.Sprintf("%04d-%02d-01", year, month)
	end := fmt.Sprintf("%04d-%02d-31", year, month)
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+transactionColumns+`
		FROM transactions t
		LEFT JOIN accounts a ON a.id = t.account_id
		LEFT JOIN categories cat ON cat.id = t.category_id
		LEFT JOIN currencies c ON c.id = t.currency_id
		WHERE t.deleted_at IS NULL AND t.date >= ? AND t.date <= ?
		ORDER BY t.date DESC, t.id DESC
	`, start, end)
	if err != nil {
		return nil, fmt.Errorf("listar transacciones del mes: %w", err)
	}
	defer rows.Close()
	return scanTransactions(rows)
}

// ListByCategoryMonth devuelve las transacciones de una categoría en un mes
// (historial simple en el modal de asignación).
func (s *TransactionStorage) ListByCategoryMonth(ctx context.Context, categoryID int64, year, month int) ([]models.Transaction, error) {
	start := fmt.Sprintf("%04d-%02d-01", year, month)
	end := fmt.Sprintf("%04d-%02d-31", year, month)
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+transactionColumns+`
		FROM transactions t
		LEFT JOIN accounts a ON a.id = t.account_id
		LEFT JOIN categories cat ON cat.id = t.category_id
		LEFT JOIN currencies c ON c.id = t.currency_id
		WHERE t.deleted_at IS NULL AND t.category_id = ? AND t.date >= ? AND t.date <= ?
		ORDER BY t.date DESC, t.id DESC
	`, categoryID, start, end)
	if err != nil {
		return nil, fmt.Errorf("listar transacciones de categoría: %w", err)
	}
	defer rows.Close()
	return scanTransactions(rows)
}

// GetByID busca una transacción por ID.
func (s *TransactionStorage) GetByID(ctx context.Context, id int64) (*models.Transaction, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+transactionColumns+`
		FROM transactions t
		LEFT JOIN accounts a ON a.id = t.account_id
		LEFT JOIN categories cat ON cat.id = t.category_id
		LEFT JOIN currencies c ON c.id = t.currency_id
		WHERE t.id = ? AND t.deleted_at IS NULL
	`, id)
	return scanTransaction(row)
}

// Create inserta una nueva transacción.
func (s *TransactionStorage) Create(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO transactions (account_id, category_id, currency_id, date, payee,
		                          memo, outflow, inflow, cleared)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tx.AccountID, tx.CategoryID, tx.CurrencyID, tx.Date, tx.Payee,
		tx.Memo, tx.Outflow, tx.Inflow, boolToInt(tx.Cleared))
	if err != nil {
		return nil, fmt.Errorf("insertar transacción: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de transacción: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Update actualiza una transacción existente.
func (s *TransactionStorage) Update(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE transactions
		SET account_id = ?, category_id = ?, currency_id = ?, date = ?, payee = ?,
		    memo = ?, outflow = ?, inflow = ?, cleared = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, tx.AccountID, tx.CategoryID, tx.CurrencyID, tx.Date, tx.Payee,
		tx.Memo, tx.Outflow, tx.Inflow, boolToInt(tx.Cleared), tx.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar transacción: %w", err)
	}
	return s.GetByID(ctx, tx.ID)
}

// SoftDelete marca una transacción como eliminada.
func (s *TransactionStorage) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE transactions SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("eliminar transacción: %w", err)
	}
	return nil
}

// ActivityByCategoryGrouped devuelve el activity (SUM(outflow-inflow)) por
// categoría y moneda para un mes.
func (s *TransactionStorage) ActivityByCategoryGrouped(ctx context.Context, year, month int) (map[int64]map[int64]float64, error) {
	start := fmt.Sprintf("%04d-%02d-01", year, month)
	end := fmt.Sprintf("%04d-%02d-31", year, month)
	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(category_id, 0), currency_id, SUM(outflow - inflow)
		FROM transactions
		WHERE deleted_at IS NULL AND date >= ? AND date <= ?
		GROUP BY category_id, currency_id
	`, start, end)
	if err != nil {
		return nil, fmt.Errorf("sumar actividad del mes: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]map[int64]float64)
	for rows.Next() {
		var categoryID, currencyID int64
		var sum float64
		if err := rows.Scan(&categoryID, &currencyID, &sum); err != nil {
			return nil, fmt.Errorf("escanear suma de actividad: %w", err)
		}
		if result[categoryID] == nil {
			result[categoryID] = make(map[int64]float64)
		}
		result[categoryID][currencyID] = sum
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// IncomeByMonthGrouped devuelve el total de ingresos (inflow) por moneda del mes.
func (s *TransactionStorage) IncomeByMonthGrouped(ctx context.Context, year, month int) (map[int64]float64, error) {
	start := fmt.Sprintf("%04d-%02d-01", year, month)
	end := fmt.Sprintf("%04d-%02d-31", year, month)
	rows, err := s.db.QueryContext(ctx, `
		SELECT currency_id, SUM(inflow)
		FROM transactions
		WHERE deleted_at IS NULL AND date >= ? AND date <= ?
		GROUP BY currency_id
	`, start, end)
	if err != nil {
		return nil, fmt.Errorf("sumar ingresos del mes: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]float64)
	for rows.Next() {
		var currencyID int64
		var sum float64
		if err := rows.Scan(&currencyID, &sum); err != nil {
			return nil, fmt.Errorf("escanear suma de ingresos: %w", err)
		}
		result[currencyID] = sum
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// SumByMonthGrouped devuelve el total de gastos (outflow) por categoría y moneda.
func (s *TransactionStorage) SumByMonthGrouped(ctx context.Context, year, month int) (map[int64]map[int64]float64, error) {
	return s.ActivityByCategoryGrouped(ctx, year, month)
}

func scanTransaction(row *sql.Row) (*models.Transaction, error) {
	var t models.Transaction
	var deletedAt sql.NullTime
	var accountName, categoryName, code sql.NullString
	var categoryID sql.NullInt64
	var cleared int
	if err := row.Scan(&t.ID, &t.AccountID, &accountName, &categoryID, &categoryName,
		&t.CurrencyID, &code, &t.Date, &t.Payee, &t.Memo,
		&t.Outflow, &t.Inflow, &cleared, &deletedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear transacción: %w", err)
	}
	t.AccountName = accountName.String
	t.CategoryName = categoryName.String
	t.CurrencyCode = code.String
	if categoryID.Valid {
		cid := categoryID.Int64
		t.CategoryID = &cid
	}
	t.Cleared = cleared == 1
	if deletedAt.Valid {
		t.DeletedAt = &deletedAt.Time
	}
	return &t, nil
}

func scanTransactions(rows *sql.Rows) ([]models.Transaction, error) {
	var transactions []models.Transaction
	for rows.Next() {
		var t models.Transaction
		var deletedAt sql.NullTime
		var accountName, categoryName, code sql.NullString
		var categoryID sql.NullInt64
		var cleared int
		if err := rows.Scan(&t.ID, &t.AccountID, &accountName, &categoryID, &categoryName,
			&t.CurrencyID, &code, &t.Date, &t.Payee, &t.Memo,
			&t.Outflow, &t.Inflow, &cleared, &deletedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear transacción: %w", err)
		}
		t.AccountName = accountName.String
		t.CategoryName = categoryName.String
		t.CurrencyCode = code.String
		if categoryID.Valid {
			cid := categoryID.Int64
			t.CategoryID = &cid
		}
		t.Cleared = cleared == 1
		if deletedAt.Valid {
			t.DeletedAt = &deletedAt.Time
		}
		transactions = append(transactions, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return transactions, nil
}