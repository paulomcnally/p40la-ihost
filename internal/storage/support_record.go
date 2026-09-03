package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// SupportRecordStorage encapsula el acceso a la tabla support_records.
type SupportRecordStorage struct {
	db *sql.DB
}

// NewSupportRecordStorage crea un nuevo SupportRecordStorage.
func NewSupportRecordStorage(db *sql.DB) *SupportRecordStorage {
	return &SupportRecordStorage{db: db}
}

const supportRecordSelectCols = `
		r.id, r.child_id, c.first_name, c.last_name,
		r.pension_category_id, pc.name,
		r.year, r.month, r.amount, r.currency, r.status,
		r.paid_at, r.payment_method, r.payment_reference, r.evidence_notes, r.notes,
		r.proof_file_name, r.original_amount, r.original_currency, r.exchange_rate,
		r.created_at, r.updated_at`

// ListByFilters devuelve los registros de un período, opcionalmente filtrados por hijo.
func (s *SupportRecordStorage) ListByFilters(ctx context.Context, year, month int, childID int64) ([]models.SupportRecord, error) {
	query := `
		SELECT` + supportRecordSelectCols + `
		FROM support_records r
		JOIN children c ON c.id = r.child_id
		JOIN pension_categories pc ON pc.id = r.pension_category_id
		WHERE r.year = ? AND r.month = ?`
	args := []any{year, month}

	if childID > 0 {
		query += ` AND r.child_id = ?`
		args = append(args, childID)
	}
	query += ` ORDER BY r.created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listar registros de manutención: %w", err)
	}
	defer rows.Close()
	return scanSupportRecords(rows)
}

// GetByID busca un registro por ID con sus relaciones resueltas.
func (s *SupportRecordStorage) GetByID(ctx context.Context, id int64) (*models.SupportRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT`+supportRecordSelectCols+`
		FROM support_records r
		JOIN children c ON c.id = r.child_id
		JOIN pension_categories pc ON pc.id = r.pension_category_id
		WHERE r.id = ?
	`, id)
	return scanSupportRecord(row)
}

// Create inserta un nuevo registro de manutención.
func (s *SupportRecordStorage) Create(ctx context.Context, data *models.SupportRecord) (*models.SupportRecord, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO support_records (child_id, pension_category_id, year, month, amount, currency, status, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, data.ChildID, data.PensionCategoryID, data.Year, data.Month, data.Amount, data.Currency, data.Status, nullString(data.Notes))
	if err != nil {
		return nil, fmt.Errorf("insertar registro de manutención: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("obtener id de registro: %w", err)
	}
	return s.GetByID(ctx, id)
}

// Exists verifica si ya existe un registro para hijo+categoría+período.
func (s *SupportRecordStorage) Exists(ctx context.Context, childID, categoryID int64, year, month int) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM support_records
		WHERE child_id = ? AND pension_category_id = ? AND year = ? AND month = ?
	`, childID, categoryID, year, month).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("verificar registro existente: %w", err)
	}
	return count > 0, nil
}

// Update actualiza monto, categoría y notas de un registro.
func (s *SupportRecordStorage) Update(ctx context.Context, id int64, amount float64, categoryID int64, notes *string) (*models.SupportRecord, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE support_records
		SET amount = ?, pension_category_id = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, amount, categoryID, nullString(notes), id)
	if err != nil {
		return nil, fmt.Errorf("actualizar registro de manutención: %w", err)
	}
	return s.GetByID(ctx, id)
}

// MarkPaid persiste el pago de un registro.
func (s *SupportRecordStorage) MarkPaid(ctx context.Context, id int64, paidAt time.Time, paymentMethod, paymentReference, evidenceNotes string, originalAmount *float64, originalCurrency string, exchangeRate *float64) (*models.SupportRecord, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE support_records
		SET status = 'paid', paid_at = ?, payment_method = ?, payment_reference = ?,
		    evidence_notes = ?, original_amount = ?, original_currency = ?, exchange_rate = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, paidAt, nullStringIfEmpty(paymentMethod), nullStringIfEmpty(paymentReference), nullStringIfEmpty(evidenceNotes),
		nullFloat(originalAmount), nullStringIfEmpty(originalCurrency), nullFloat(exchangeRate), id)
	if err != nil {
		return nil, fmt.Errorf("marcar registro como pagado: %w", err)
	}
	return s.GetByID(ctx, id)
}

// MarkPending devuelve el registro a estado pendiente, limpiando los campos de pago.
func (s *SupportRecordStorage) MarkPending(ctx context.Context, id int64) (*models.SupportRecord, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE support_records
		SET status = 'pending', paid_at = NULL, payment_method = NULL, payment_reference = NULL,
		    evidence_notes = NULL, original_amount = NULL, original_currency = NULL,
		    exchange_rate = NULL, proof_file_name = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	if err != nil {
		return nil, fmt.Errorf("marcar registro como pendiente: %w", err)
	}
	return s.GetByID(ctx, id)
}

// MarkRejected marca un registro como rechazado con el motivo en notes.
func (s *SupportRecordStorage) MarkRejected(ctx context.Context, id int64, reason string) (*models.SupportRecord, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE support_records
		SET status = 'rejected', notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, nullStringIfEmpty(reason), id)
	if err != nil {
		return nil, fmt.Errorf("marcar registro como rechazado: %w", err)
	}
	return s.GetByID(ctx, id)
}

// UpdateProofFileName guarda el nombre del comprobante subido.
func (s *SupportRecordStorage) UpdateProofFileName(ctx context.Context, id int64, fileName string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE support_records
		SET proof_file_name = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, fileName, id)
	if err != nil {
		return fmt.Errorf("actualizar comprobante de registro: %w", err)
	}
	return nil
}

// GetYearMonth devuelve el período (year, month) de un registro.
func (s *SupportRecordStorage) GetYearMonth(ctx context.Context, id int64) (year, month int, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT year, month FROM support_records WHERE id = ?`, id).Scan(&year, &month)
	if err != nil {
		return 0, 0, fmt.Errorf("obtener período de registro: %w", err)
	}
	return year, month, nil
}

func scanSupportRecord(row *sql.Row) (*models.SupportRecord, error) {
	var r models.SupportRecord
	var firstName, lastName string
	var paidAt sql.NullTime
	var paymentMethod, paymentReference, evidenceNotes, notes, proofFileName, originalCurrency sql.NullString
	var originalAmount, exchangeRate sql.NullFloat64

	if err := row.Scan(&r.ID, &r.ChildID, &firstName, &lastName,
		&r.PensionCategoryID, &r.CategoryName,
		&r.Year, &r.Month, &r.Amount, &r.Currency, &r.Status,
		&paidAt, &paymentMethod, &paymentReference, &evidenceNotes, &notes,
		&proofFileName, &originalAmount, &originalCurrency, &exchangeRate,
		&r.CreatedAt, &r.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear registro de manutención: %w", err)
	}

	r.ChildName = firstName + " " + lastName
	if paidAt.Valid {
		r.PaidAt = &paidAt.Time
	}
	if paymentMethod.Valid {
		r.PaymentMethod = &paymentMethod.String
	}
	if paymentReference.Valid {
		r.PaymentReference = &paymentReference.String
	}
	if evidenceNotes.Valid {
		r.EvidenceNotes = &evidenceNotes.String
	}
	if notes.Valid {
		r.Notes = &notes.String
	}
	if proofFileName.Valid {
		r.ProofFileName = &proofFileName.String
	}
	if originalAmount.Valid {
		r.OriginalAmount = &originalAmount.Float64
	}
	if originalCurrency.Valid {
		r.OriginalCurrency = &originalCurrency.String
	}
	if exchangeRate.Valid {
		r.ExchangeRate = &exchangeRate.Float64
	}
	return &r, nil
}

func scanSupportRecords(rows *sql.Rows) ([]models.SupportRecord, error) {
	var records []models.SupportRecord
	for rows.Next() {
		var r models.SupportRecord
		var firstName, lastName string
		var paidAt sql.NullTime
		var paymentMethod, paymentReference, evidenceNotes, notes, proofFileName, originalCurrency sql.NullString
		var originalAmount, exchangeRate sql.NullFloat64

		if err := rows.Scan(&r.ID, &r.ChildID, &firstName, &lastName,
			&r.PensionCategoryID, &r.CategoryName,
			&r.Year, &r.Month, &r.Amount, &r.Currency, &r.Status,
			&paidAt, &paymentMethod, &paymentReference, &evidenceNotes, &notes,
			&proofFileName, &originalAmount, &originalCurrency, &exchangeRate,
			&r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("escanear registro de manutención: %w", err)
		}

		r.ChildName = firstName + " " + lastName
		if paidAt.Valid {
			r.PaidAt = &paidAt.Time
		}
		if paymentMethod.Valid {
			r.PaymentMethod = &paymentMethod.String
		}
		if paymentReference.Valid {
			r.PaymentReference = &paymentReference.String
		}
		if evidenceNotes.Valid {
			r.EvidenceNotes = &evidenceNotes.String
		}
		if notes.Valid {
			r.Notes = &notes.String
		}
		if proofFileName.Valid {
			r.ProofFileName = &proofFileName.String
		}
		if originalAmount.Valid {
			r.OriginalAmount = &originalAmount.Float64
		}
		if originalCurrency.Valid {
			r.OriginalCurrency = &originalCurrency.String
		}
		if exchangeRate.Valid {
			r.ExchangeRate = &exchangeRate.Float64
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

// nullString convierte un *string en un valor sql nullable.
func nullString(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

// nullStringIfEmpty convierte una string en NULL si está vacía.
func nullStringIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullFloat convierte un *float64 en un valor sql nullable.
func nullFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}
