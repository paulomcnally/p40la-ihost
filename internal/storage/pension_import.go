package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// PensionImportStorage encapsula las tablas del importador de pensión (SPEC-119).
type PensionImportStorage struct {
	db *sql.DB
}

// NewPensionImportStorage crea un nuevo PensionImportStorage.
func NewPensionImportStorage(db *sql.DB) *PensionImportStorage {
	return &PensionImportStorage{db: db}
}

// ImportRecordInput es una fila ya decidida por el usuario para confirmar.
type ImportRecordInput struct {
	LineNo       int
	CategoryID   int64
	Amount       float64
	Status       string
	SharePercent int
	Currency     string
}

// ImportKeywordInput es una palabra clave aprendida al confirmar.
type ImportKeywordInput struct {
	CategoryID int64
	Keyword    string
	Normalized string
}

// CreateBatch guarda un lote en estado preview y devuelve su ID.
func (s *PensionImportStorage) CreateBatch(ctx context.Context, filename, fileHash string, year, month, totalLines int) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO pension_import_batches (filename, file_hash, period_year, period_month, status, total_lines, declared_total, computed_total)
		VALUES (?, ?, ?, ?, 'preview', ?, 0, 0)
	`, filename, fileHash, year, month, totalLines)
	if err != nil {
		return 0, fmt.Errorf("crear lote de importación: %w", err)
	}
	return res.LastInsertId()
}

// SaveRows persiste las filas parseadas de un lote.
func (s *PensionImportStorage) SaveRows(ctx context.Context, batchID int64, rows []models.PensionImportRow) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO pension_import_rows
			(batch_id, line_no, raw_text, concept, amount_total, amount_to_record, rounding_adjustment,
			 share_percent, currency, suggested_category_id, chosen_category_id, child_id,
			 confidence, match_reason, validation_status, validation_code, action)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("preparar filas de importación: %w", err)
	}
	defer stmt.Close()

	for _, r := range rows {
		if _, err := stmt.ExecContext(ctx,
			batchID, r.LineNo, r.RawText, r.Concept, r.AmountTotal, r.AmountToRecord, r.RoundingAdjustment,
			r.SharePercent, r.Currency, r.SuggestedCategoryID, r.ChosenCategoryID, r.ChildID,
			r.Confidence, r.MatchReason, r.ValidationStatus, r.ValidationCode, r.Action,
		); err != nil {
			return fmt.Errorf("guardar fila %d: %w", r.LineNo, err)
		}
	}
	return tx.Commit()
}

// UpdateBatchTotals guarda los totales calculados del preview.
func (s *PensionImportStorage) UpdateBatchTotals(ctx context.Context, id int64, declared, computed float64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE pension_import_batches SET declared_total = ?, computed_total = ? WHERE id = ?
	`, declared, computed, id)
	if err != nil {
		return fmt.Errorf("actualizar totales del lote: %w", err)
	}
	return nil
}

// GetBatch busca un lote por ID.
func (s *PensionImportStorage) GetBatch(ctx context.Context, id int64) (*models.PensionImportBatch, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, filename, file_hash, period_year, period_month, status,
		       total_lines, declared_total, computed_total, created_count, skipped_count, created_at, confirmed_at
		FROM pension_import_batches WHERE id = ?
	`, id)
	var b models.PensionImportBatch
	var confirmedAt sql.NullTime
	if err := row.Scan(&b.ID, &b.Filename, &b.FileHash, &b.PeriodYear, &b.PeriodMonth, &b.Status,
		&b.TotalLines, &b.DeclaredTotal, &b.ComputedTotal, &b.CreatedCount, &b.SkippedCount, &b.CreatedAt, &confirmedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("buscar lote de importación: %w", err)
	}
	if confirmedAt.Valid {
		b.ConfirmedAt = &confirmedAt.Time
	}
	return &b, nil
}

// ListBatches devuelve los últimos lotes.
func (s *PensionImportStorage) ListBatches(ctx context.Context, limit int) ([]models.PensionImportBatch, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, filename, file_hash, period_year, period_month, status,
		       total_lines, declared_total, computed_total, created_count, skipped_count, created_at, confirmed_at
		FROM pension_import_batches
		ORDER BY id DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("listar lotes de importación: %w", err)
	}
	defer rows.Close()

	var batches []models.PensionImportBatch
	for rows.Next() {
		var b models.PensionImportBatch
		var confirmedAt sql.NullTime
		if err := rows.Scan(&b.ID, &b.Filename, &b.FileHash, &b.PeriodYear, &b.PeriodMonth, &b.Status,
			&b.TotalLines, &b.DeclaredTotal, &b.ComputedTotal, &b.CreatedCount, &b.SkippedCount, &b.CreatedAt, &confirmedAt); err != nil {
			return nil, err
		}
		if confirmedAt.Valid {
			b.ConfirmedAt = &confirmedAt.Time
		}
		batches = append(batches, b)
	}
	return batches, rows.Err()
}

// ListRows devuelve las filas de un lote.
func (s *PensionImportStorage) ListRows(ctx context.Context, batchID int64) ([]models.PensionImportRow, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.batch_id, r.line_no, r.raw_text, r.concept, r.amount_total, r.amount_to_record,
		       r.rounding_adjustment, r.share_percent, r.currency, r.suggested_category_id,
		       r.chosen_category_id, r.child_id, r.confidence, r.match_reason,
		       r.validation_status, r.validation_code, r.action, COALESCE(pc.name, ''), r.created_at
		FROM pension_import_rows r
		LEFT JOIN pension_categories pc ON pc.id = r.chosen_category_id
		WHERE r.batch_id = ?
		ORDER BY r.line_no
	`, batchID)
	if err != nil {
		return nil, fmt.Errorf("listar filas de importación: %w", err)
	}
	defer rows.Close()

	var out []models.PensionImportRow
	for rows.Next() {
		var r models.PensionImportRow
		var suggested, chosen, child sql.NullInt64
		if err := rows.Scan(&r.ID, &r.BatchID, &r.LineNo, &r.RawText, &r.Concept, &r.AmountTotal,
			&r.AmountToRecord, &r.RoundingAdjustment, &r.SharePercent, &r.Currency,
			&suggested, &chosen, &child, &r.Confidence, &r.MatchReason,
			&r.ValidationStatus, &r.ValidationCode, &r.Action, &r.CategoryName, &r.CreatedAt); err != nil {
			return nil, err
		}
		if suggested.Valid {
			r.SuggestedCategoryID = &suggested.Int64
		}
		if chosen.Valid {
			r.ChosenCategoryID = &chosen.Int64
		}
		if child.Valid {
			r.ChildID = &child.Int64
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateRowDecision guarda la decisión final de una fila.
func (s *PensionImportStorage) UpdateRowDecision(ctx context.Context, batchID int64, r ImportRecordInput, childID int64, validationStatus, validationCode, action string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE pension_import_rows
		SET chosen_category_id = ?, child_id = ?, amount_to_record = ?, share_percent = ?,
		    validation_status = ?, validation_code = ?, action = ?
		WHERE batch_id = ? AND line_no = ?
	`, r.CategoryID, childID, r.Amount, r.SharePercent, validationStatus, validationCode, action, batchID, r.LineNo)
	if err != nil {
		return fmt.Errorf("actualizar fila %d: %w", r.LineNo, err)
	}
	return nil
}

// ConfirmImport crea los registros, aprende palabras clave y actualiza el lote
// dentro de una única transacción. Es idempotente por (hijo, categoría, período):
// si existe el mismo monto lo omite; si el monto difiere devuelve conflicto.
func (s *PensionImportStorage) ConfirmImport(
	ctx context.Context,
	batchID, childID int64,
	year, month int,
	records []ImportRecordInput,
	keywords []ImportKeywordInput,
) (int, int, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	created, skipped := 0, 0
	for _, r := range records {
		var existingAmount float64
		err := tx.QueryRowContext(ctx, `
			SELECT amount FROM support_records
			WHERE child_id = ? AND pension_category_id = ? AND year = ? AND month = ?
		`, childID, r.CategoryID, year, month).Scan(&existingAmount)
		switch {
		case err == sql.ErrNoRows:
			status := r.Status
			if status == "" {
				status = "pending"
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO support_records (child_id, pension_category_id, year, month, amount, currency, status)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, childID, r.CategoryID, year, month, r.Amount, r.Currency, status); err != nil {
				return 0, 0, 0, fmt.Errorf("crear registro de la línea %d: %w", r.LineNo, err)
			}
			created++
		case err != nil:
			return 0, 0, 0, fmt.Errorf("verificar registro de la línea %d: %w", r.LineNo, err)
		default:
			if !sameMoney(existingAmount, r.Amount) {
				return 0, 0, 0, fmt.Errorf("la línea %d ya existe con otro monto (%.2f vs %.2f)", r.LineNo, existingAmount, r.Amount)
			}
			skipped++
		}
	}

	learned := 0
	for _, k := range keywords {
		if k.Normalized == "" {
			continue
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
			VALUES (?, ?, ?, 'learned')
			ON CONFLICT(normalized_keyword) DO NOTHING
		`, k.CategoryID, k.Keyword, k.Normalized)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("aprender palabra clave %q: %w", k.Keyword, err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			learned++
		}
	}

	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
		UPDATE pension_import_batches
		SET status = 'confirmed', created_count = ?, skipped_count = ?, confirmed_at = ?
		WHERE id = ?
	`, created, skipped, now, batchID); err != nil {
		return 0, 0, 0, fmt.Errorf("confirmar lote: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, 0, err
	}
	return created, skipped, learned, nil
}

// ListKeywords devuelve todas las palabras clave con el nombre de la categoría.
func (s *PensionImportStorage) ListKeywords(ctx context.Context) ([]models.PensionCategoryKeyword, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, pension_category_id, keyword, normalized_keyword, source, created_at, updated_at
		FROM pension_category_keywords
		ORDER BY normalized_keyword
	`)
	if err != nil {
		return nil, fmt.Errorf("listar palabras clave: %w", err)
	}
	defer rows.Close()

	var out []models.PensionCategoryKeyword
	for rows.Next() {
		var k models.PensionCategoryKeyword
		if err := rows.Scan(&k.ID, &k.PensionCategoryID, &k.Keyword, &k.NormalizedKeyword, &k.Source, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// CreateKeyword agrega una palabra clave manual.
func (s *PensionImportStorage) CreateKeyword(ctx context.Context, categoryID int64, keyword, normalized, source string) (*models.PensionCategoryKeyword, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(normalized_keyword) DO UPDATE SET pension_category_id = excluded.pension_category_id,
		    keyword = excluded.keyword, source = excluded.source, updated_at = CURRENT_TIMESTAMP
	`, categoryID, keyword, normalized, source)
	if err != nil {
		return nil, fmt.Errorf("crear palabra clave: %w", err)
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		if err := s.db.QueryRowContext(ctx, `SELECT id FROM pension_category_keywords WHERE normalized_keyword = ?`, normalized).Scan(&id); err != nil {
			return nil, err
		}
	}
	return s.getKeyword(ctx, id)
}

func (s *PensionImportStorage) getKeyword(ctx context.Context, id int64) (*models.PensionCategoryKeyword, error) {
	var k models.PensionCategoryKeyword
	err := s.db.QueryRowContext(ctx, `
		SELECT id, pension_category_id, keyword, normalized_keyword, source, created_at, updated_at
		FROM pension_category_keywords WHERE id = ?
	`, id).Scan(&k.ID, &k.PensionCategoryID, &k.Keyword, &k.NormalizedKeyword, &k.Source, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

// DeleteKeyword elimina una palabra clave por ID.
func (s *PensionImportStorage) DeleteKeyword(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pension_category_keywords WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("eliminar palabra clave: %w", err)
	}
	return nil
}

func sameMoney(a, b float64) bool {
	return int64(a*100+0.5) == int64(b*100+0.5)
}
