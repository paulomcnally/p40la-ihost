package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// DebtBillStorage encapsula el acceso a la tabla debt_bills (cuotas).
type DebtBillStorage struct {
	db *sql.DB
}

// NewDebtBillStorage crea un nuevo DebtBillStorage.
func NewDebtBillStorage(db *sql.DB) *DebtBillStorage {
	return &DebtBillStorage{db: db}
}

const debtBillColumns = `
	db.id, db.debt_id, COALESCE(d.description, ''), COALESCE(i.name, ''),
	COALESCE(c.code, ''), db.installment_number, db.due_date, db.amount,
	db.status, db.paid_at, db.payment_reference, db.deleted_at,
	db.created_at, db.updated_at
`

const debtBillJoins = `
	FROM debt_bills db
	LEFT JOIN debts d ON d.id = db.debt_id AND d.deleted_at IS NULL
	LEFT JOIN institutions i ON i.id = d.institution_id
	LEFT JOIN currencies c ON c.id = d.currency_id
`

// ListByDebt devuelve las cuotas no eliminadas de una deuda.
func (s *DebtBillStorage) ListByDebt(ctx context.Context, debtID int64) ([]models.DebtBill, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+debtBillColumns+`
		`+debtBillJoins+`
		WHERE db.debt_id = ? AND db.deleted_at IS NULL
		ORDER BY db.installment_number
	`, debtID)
	if err != nil {
		return nil, fmt.Errorf("listar cuotas de deuda: %w", err)
	}
	defer rows.Close()

	return scanDebtBills(rows)
}

// GetByID busca una cuota por su ID.
func (s *DebtBillStorage) GetByID(ctx context.Context, id int64) (*models.DebtBill, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+debtBillColumns+`
		`+debtBillJoins+`
		WHERE db.id = ? AND db.deleted_at IS NULL
	`, id)
	return scanDebtBill(row)
}

// FindByDebtInstallment busca una cuota existente por deuda y número (dedup).
func (s *DebtBillStorage) FindByDebtInstallment(ctx context.Context, debtID int64, number int) (*models.DebtBill, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+debtBillColumns+`
		`+debtBillJoins+`
		WHERE db.debt_id = ? AND db.installment_number = ? AND db.deleted_at IS NULL
	`, debtID, number)
	return scanDebtBill(row)
}

// ListDueOnDate devuelve las cuotas pending que vencen en una fecha dada
// (email diario agrupado, SPEC-054).
func (s *DebtBillStorage) ListDueOnDate(ctx context.Context, date string) ([]models.DebtBill, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+debtBillColumns+`
		`+debtBillJoins+`
		WHERE db.due_date = ? AND db.status = 'pending' AND db.deleted_at IS NULL
		ORDER BY db.amount DESC
	`, date)
	if err != nil {
		return nil, fmt.Errorf("listar cuotas por vencimiento: %w", err)
	}
	defer rows.Close()

	return scanDebtBills(rows)
}

// ListByMonth devuelve las cuotas no eliminadas cuyo vencimiento cae en un
// mes (YYYY-MM), para la vista Calendario (SPEC-054).
func (s *DebtBillStorage) ListByMonth(ctx context.Context, year, month int) ([]models.DebtBill, error) {
	prefix := fmt.Sprintf("%04d-%02d", year, month)
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+debtBillColumns+`
		`+debtBillJoins+`
		WHERE substr(db.due_date, 1, 7) = ? AND db.deleted_at IS NULL
		ORDER BY db.due_date, db.installment_number
	`, prefix)
	if err != nil {
		return nil, fmt.Errorf("listar cuotas del mes: %w", err)
	}
	defer rows.Close()

	return scanDebtBills(rows)
}

// Create inserta una nueva cuota.
func (s *DebtBillStorage) Create(ctx context.Context, bill *models.DebtBill) (*models.DebtBill, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO debt_bills (debt_id, installment_number, due_date, amount, status)
		VALUES (?, ?, ?, ?, ?)
	`, bill.DebtID, bill.InstallmentNumber, bill.DueDate, bill.Amount, bill.Status)
	if err != nil {
		return nil, fmt.Errorf("insertar cuota: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de cuota: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Pay marca una cuota como pagada persistiendo fecha de pago y referencia.
func (s *DebtBillStorage) Pay(ctx context.Context, id int64, paidAt time.Time, paymentReference string) (*models.DebtBill, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE debt_bills
		SET status = 'paid', paid_at = ?, payment_reference = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`, paidAt, paymentReference, id)
	if err != nil {
		return nil, fmt.Errorf("marcar cuota como pagada: %w", err)
	}
	return s.GetByID(ctx, id)
}

// SoftDeleteByDebt marca como eliminadas todas las cuotas de una deuda.
func (s *DebtBillStorage) SoftDeleteByDebt(ctx context.Context, debtID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE debt_bills SET deleted_at = CURRENT_TIMESTAMP
		WHERE debt_id = ? AND deleted_at IS NULL
	`, debtID)
	if err != nil {
		return fmt.Errorf("eliminar cuotas de deuda: %w", err)
	}
	return nil
}

func scanDebtBill(row *sql.Row) (*models.DebtBill, error) {
	var b models.DebtBill
	var deletedAt, paidAt sql.NullTime
	var debtDescription, institutionName, currencyCode, paymentReference sql.NullString
	if err := row.Scan(&b.ID, &b.DebtID, &debtDescription, &institutionName,
		&currencyCode, &b.InstallmentNumber, &b.DueDate, &b.Amount,
		&b.Status, &paidAt, &paymentReference, &deletedAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear cuota: %w", err)
	}
	b.DebtDescription = debtDescription.String
	b.InstitutionName = institutionName.String
	b.CurrencyCode = currencyCode.String
	b.PaymentReference = paymentReference.String
	if deletedAt.Valid {
		b.DeletedAt = &deletedAt.Time
	}
	if paidAt.Valid {
		b.PaidAt = &paidAt.Time
	}
	return &b, nil
}

func scanDebtBills(rows *sql.Rows) ([]models.DebtBill, error) {
	var bills []models.DebtBill
	for rows.Next() {
		var b models.DebtBill
		var deletedAt, paidAt sql.NullTime
		var debtDescription, institutionName, currencyCode, paymentReference sql.NullString
		if err := rows.Scan(&b.ID, &b.DebtID, &debtDescription, &institutionName,
			&currencyCode, &b.InstallmentNumber, &b.DueDate, &b.Amount,
			&b.Status, &paidAt, &paymentReference, &deletedAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear cuota: %w", err)
		}
		b.DebtDescription = debtDescription.String
		b.InstitutionName = institutionName.String
		b.CurrencyCode = currencyCode.String
		b.PaymentReference = paymentReference.String
		if deletedAt.Valid {
			b.DeletedAt = &deletedAt.Time
		}
		if paidAt.Valid {
			b.PaidAt = &paidAt.Time
		}
		bills = append(bills, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return bills, nil
}
