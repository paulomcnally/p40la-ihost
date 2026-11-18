package creditcard

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrBankNotFound indica que el banco no existe.
	ErrBankNotFound = errors.New("banco no encontrado")
	// ErrCardNotFound indica que la tarjeta no existe.
	ErrCardNotFound = errors.New("tarjeta de crédito no encontrada")
	// ErrStatementNotFound indica que el estado de cuenta no existe.
	ErrStatementNotFound = errors.New("estado de cuenta no encontrado")
	// ErrTransactionNotFound indica que el movimiento no existe.
	ErrTransactionNotFound = errors.New("movimiento no encontrado")
	// ErrDuplicateCard indica un alias repetido en el mismo banco.
	ErrDuplicateCard = errors.New("ya existe una tarjeta con ese alias en el banco")
	// ErrStatementExists indica un corte repetido para la misma tarjeta.
	ErrStatementExists = errors.New("ya existe un estado de cuenta para esa fecha de corte")
)

// Storage es el único punto con SQL del módulo (ARCH-MODULES).
type Storage struct {
	db *sql.DB
}

// NewStorage crea el storage del módulo.
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

const cardColumns = `
	c.id, c.bank_id, b.name, c.alias, c.network, c.last4, c.holder,
	c.currency_id, COALESCE(cur.code, ''), c.secondary_currency_id, COALESCE(scur.code, ''),
	c.credit_limit, c.credit_limit_usd, c.statement_day, c.payment_due_day, c.min_payment_pct,
	c.interest_current_rate, c.interest_late_rate, c.interest_current_rate_usd, c.interest_late_rate_usd,
	c.color, c.icon, c.active, c.notes,
	(SELECT COUNT(*) FROM card_statements s WHERE s.card_id = c.id),
	c.created_at, c.updated_at
`

const cardFrom = `
	FROM credit_cards c
	JOIN banks b ON b.id = c.bank_id
	LEFT JOIN currencies cur ON cur.id = c.currency_id
	LEFT JOIN currencies scur ON scur.id = c.secondary_currency_id
`

type rowScanner interface {
	Scan(dest ...any) error
}

// BankExists indica si el banco existe.
func (s *Storage) BankExists(ctx context.Context, bankID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM banks WHERE id = ?`, bankID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("verificar banco: %w", err)
	}
	return count > 0, nil
}

// CurrencyExists indica si la moneda existe.
func (s *Storage) CurrencyExists(ctx context.Context, currencyID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM currencies WHERE id = ? AND deleted_at IS NULL`, currencyID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("verificar moneda: %w", err)
	}
	return count > 0, nil
}

// CategoryExists indica si la categoría existe.
func (s *Storage) CategoryExists(ctx context.Context, categoryID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM categories WHERE id = ? AND deleted_at IS NULL`, categoryID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("verificar categoría: %w", err)
	}
	return count > 0, nil
}

// ListCards devuelve las tarjetas de un banco.
func (s *Storage) ListCards(ctx context.Context, bankID int64) ([]CreditCard, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+cardColumns+cardFrom+`
		WHERE c.bank_id = ?
		ORDER BY c.alias COLLATE NOCASE
	`, bankID)
	if err != nil {
		return nil, fmt.Errorf("listar tarjetas: %w", err)
	}
	defer rows.Close()

	var cards []CreditCard
	for rows.Next() {
		card, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		cards = append(cards, *card)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar tarjetas: %w", err)
	}
	return cards, nil
}

// GetCard devuelve una tarjeta o nil si no existe.
func (s *Storage) GetCard(ctx context.Context, id int64) (*CreditCard, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+cardColumns+cardFrom+`
		WHERE c.id = ?
	`, id)
	card, err := scanCard(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return card, nil
}

// CreateCard inserta una tarjeta y devuelve la fila creada.
func (s *Storage) CreateCard(ctx context.Context, card *CreditCard) (*CreditCard, error) {
	var secondary any
	if card.SecondaryCurrencyID != nil {
		secondary = *card.SecondaryCurrencyID
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO credit_cards (
			bank_id, alias, network, last4, holder, currency_id, secondary_currency_id,
			credit_limit, credit_limit_usd, statement_day, payment_due_day, min_payment_pct,
			interest_current_rate, interest_late_rate, interest_current_rate_usd,
			interest_late_rate_usd, color, icon, active, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, card.BankID, card.Alias, card.Network, card.Last4, card.Holder, card.CurrencyID,
		secondary, card.CreditLimit, card.CreditLimitUSD, card.StatementDay, card.PaymentDueDay,
		card.MinPaymentPct, card.InterestCurrentRate, card.InterestLateRate,
		card.InterestCurrentRateUSD, card.InterestLateRateUSD, card.Color, card.Icon,
		card.Active, card.Notes)
	if err != nil {
		if isUniqueError(err) {
			return nil, ErrDuplicateCard
		}
		return nil, fmt.Errorf("crear tarjeta: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("id de tarjeta: %w", err)
	}
	return s.GetCard(ctx, id)
}

// UpdateCard actualiza los campos mutables y conserva el banco.
func (s *Storage) UpdateCard(ctx context.Context, card *CreditCard) (*CreditCard, error) {
	var secondary any
	if card.SecondaryCurrencyID != nil {
		secondary = *card.SecondaryCurrencyID
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE credit_cards SET
			alias = ?, network = ?, last4 = ?, holder = ?, currency_id = ?,
			secondary_currency_id = ?, credit_limit = ?, credit_limit_usd = ?,
			statement_day = ?, payment_due_day = ?, min_payment_pct = ?,
			interest_current_rate = ?, interest_late_rate = ?,
			interest_current_rate_usd = ?, interest_late_rate_usd = ?,
			color = ?, icon = ?, active = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, card.Alias, card.Network, card.Last4, card.Holder, card.CurrencyID, secondary,
		card.CreditLimit, card.CreditLimitUSD, card.StatementDay, card.PaymentDueDay,
		card.MinPaymentPct, card.InterestCurrentRate, card.InterestLateRate,
		card.InterestCurrentRateUSD, card.InterestLateRateUSD, card.Color, card.Icon,
		card.Active, card.Notes, card.ID)
	if err != nil {
		if isUniqueError(err) {
			return nil, ErrDuplicateCard
		}
		return nil, fmt.Errorf("actualizar tarjeta: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("filas de tarjeta: %w", err)
	}
	if affected == 0 {
		return nil, ErrCardNotFound
	}
	return s.GetCard(ctx, card.ID)
}

// DeleteCard elimina la tarjeta y, por cascada, sus estados y movimientos.
func (s *Storage) DeleteCard(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM credit_cards WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("eliminar tarjeta: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("filas de tarjeta: %w", err)
	}
	if affected == 0 {
		return ErrCardNotFound
	}
	return nil
}

const statementColumns = `
	s.id, s.card_id, s.period_start, s.period_end, s.min_payment_due_date,
	s.full_payment_due_date, s.previous_balance, s.previous_balance_usd,
	s.interest_current, s.interest_current_usd,
	s.interest_bonifiable, s.interest_bonifiable_usd,
	s.mora_amount, s.mora_amount_usd, s.arrears_count,
	s.over_limit_amount, s.over_limit_amount_usd,
	s.points_previous, s.points_earned, s.points_used, s.points_available,
	s.min_payment, s.min_payment_usd,
	s.total_payment, s.total_payment_usd, s.new_balance, s.new_balance_usd,
	s.status, s.source, s.source_module_id, s.import_hash, s.notes,
	(SELECT COUNT(*) FROM card_transactions t WHERE t.statement_id = s.id),
	s.created_at, s.updated_at
`

// ListStatements devuelve los estados de una tarjeta, del más reciente al más viejo.
func (s *Storage) ListStatements(ctx context.Context, cardID int64) ([]Statement, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+statementColumns+`
		FROM card_statements s
		WHERE s.card_id = ?
		ORDER BY s.period_end DESC
	`, cardID)
	if err != nil {
		return nil, fmt.Errorf("listar estados de cuenta: %w", err)
	}
	defer rows.Close()

	var statements []Statement
	for rows.Next() {
		st, err := scanStatement(rows)
		if err != nil {
			return nil, err
		}
		statements = append(statements, *st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar estados de cuenta: %w", err)
	}
	return statements, nil
}

// GetStatement devuelve un estado o nil si no existe.
func (s *Storage) GetStatement(ctx context.Context, id int64) (*Statement, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+statementColumns+`
		FROM card_statements s
		WHERE s.id = ?
	`, id)
	st, err := scanStatement(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return st, nil
}

// StatementExists indica si ya hay un corte para la tarjeta.
func (s *Storage) StatementExists(ctx context.Context, cardID int64, periodEnd string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM card_statements WHERE card_id = ? AND period_end = ?
	`, cardID, periodEnd).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("verificar corte: %w", err)
	}
	return count > 0, nil
}

// CreateStatement inserta un estado de cuenta manual.
func (s *Storage) CreateStatement(ctx context.Context, st *Statement) (*Statement, error) {
	var periodStart any
	if st.PeriodStart != nil && *st.PeriodStart != "" {
		periodStart = *st.PeriodStart
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO card_statements (
			card_id, period_start, period_end, min_payment_due_date, full_payment_due_date,
			previous_balance, previous_balance_usd, interest_current, interest_current_usd,
			interest_bonifiable, interest_bonifiable_usd, mora_amount, mora_amount_usd,
			arrears_count, over_limit_amount, over_limit_amount_usd,
			points_previous, points_earned, points_used, points_available,
			min_payment, min_payment_usd, total_payment, total_payment_usd,
			new_balance, new_balance_usd, status, source, source_module_id,
			import_hash, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, st.CardID, periodStart, st.PeriodEnd, st.MinPaymentDueDate, st.FullPaymentDueDate,
		st.PreviousBalance, st.PreviousBalanceUSD, st.InterestCurrent, st.InterestCurrentUSD,
		st.InterestBonifiable, st.InterestBonifiableUSD, st.MoraAmount, st.MoraAmountUSD,
		st.ArrearsCount, st.OverLimitAmount, st.OverLimitAmountUSD,
		st.PointsPrevious, st.PointsEarned, st.PointsUsed, st.PointsAvailable,
		st.MinPayment, st.MinPaymentUSD, st.TotalPayment, st.TotalPaymentUSD,
		st.NewBalance, st.NewBalanceUSD, st.Status, st.Source, st.SourceModuleID,
		st.ImportHash, st.Notes)
	if err != nil {
		if isUniqueError(err) {
			return nil, ErrStatementExists
		}
		return nil, fmt.Errorf("crear estado de cuenta: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("id de estado de cuenta: %w", err)
	}
	return s.GetStatement(ctx, id)
}

// UpdateStatement actualiza los campos mutables del estado.
func (s *Storage) UpdateStatement(ctx context.Context, st *Statement) (*Statement, error) {
	var periodStart any
	if st.PeriodStart != nil && *st.PeriodStart != "" {
		periodStart = *st.PeriodStart
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE card_statements SET
			period_start = ?, period_end = ?, min_payment_due_date = ?,
			full_payment_due_date = ?, previous_balance = ?, previous_balance_usd = ?,
			interest_current = ?, interest_current_usd = ?,
			interest_bonifiable = ?, interest_bonifiable_usd = ?,
			mora_amount = ?, mora_amount_usd = ?, arrears_count = ?,
			over_limit_amount = ?, over_limit_amount_usd = ?,
			points_previous = ?, points_earned = ?, points_used = ?, points_available = ?,
			min_payment = ?, min_payment_usd = ?, total_payment = ?, total_payment_usd = ?,
			new_balance = ?, new_balance_usd = ?, status = ?, notes = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, periodStart, st.PeriodEnd, st.MinPaymentDueDate, st.FullPaymentDueDate,
		st.PreviousBalance, st.PreviousBalanceUSD, st.InterestCurrent, st.InterestCurrentUSD,
		st.InterestBonifiable, st.InterestBonifiableUSD, st.MoraAmount, st.MoraAmountUSD,
		st.ArrearsCount, st.OverLimitAmount, st.OverLimitAmountUSD,
		st.PointsPrevious, st.PointsEarned, st.PointsUsed, st.PointsAvailable,
		st.MinPayment, st.MinPaymentUSD, st.TotalPayment, st.TotalPaymentUSD,
		st.NewBalance, st.NewBalanceUSD, st.Status, st.Notes, st.ID)
	if err != nil {
		if isUniqueError(err) {
			return nil, ErrStatementExists
		}
		return nil, fmt.Errorf("actualizar estado de cuenta: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("filas de estado de cuenta: %w", err)
	}
	if affected == 0 {
		return nil, ErrStatementNotFound
	}
	return s.GetStatement(ctx, st.ID)
}

// DeleteStatement elimina el estado y sus movimientos por cascada.
func (s *Storage) DeleteStatement(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM card_statements WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("eliminar estado de cuenta: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("filas de estado de cuenta: %w", err)
	}
	if affected == 0 {
		return ErrStatementNotFound
	}
	return nil
}

// ImportStatement inserta el estado y sus movimientos en una sola transacción.
func (s *Storage) ImportStatement(ctx context.Context, st *Statement, transactions []Transaction) (*Statement, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar importación: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var periodStart any
	if st.PeriodStart != nil && *st.PeriodStart != "" {
		periodStart = *st.PeriodStart
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO card_statements (
			card_id, period_start, period_end, min_payment_due_date, full_payment_due_date,
			previous_balance, previous_balance_usd, interest_current, interest_current_usd,
			interest_bonifiable, interest_bonifiable_usd, mora_amount, mora_amount_usd,
			arrears_count, over_limit_amount, over_limit_amount_usd,
			points_previous, points_earned, points_used, points_available,
			min_payment, min_payment_usd, total_payment, total_payment_usd,
			new_balance, new_balance_usd, status, source, source_module_id,
			import_hash, notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, st.CardID, periodStart, st.PeriodEnd, st.MinPaymentDueDate, st.FullPaymentDueDate,
		st.PreviousBalance, st.PreviousBalanceUSD, st.InterestCurrent, st.InterestCurrentUSD,
		st.InterestBonifiable, st.InterestBonifiableUSD, st.MoraAmount, st.MoraAmountUSD,
		st.ArrearsCount, st.OverLimitAmount, st.OverLimitAmountUSD,
		st.PointsPrevious, st.PointsEarned, st.PointsUsed, st.PointsAvailable,
		st.MinPayment, st.MinPaymentUSD, st.TotalPayment, st.TotalPaymentUSD,
		st.NewBalance, st.NewBalanceUSD, st.Status, st.Source, st.SourceModuleID,
		st.ImportHash, st.Notes)
	if err != nil {
		if isUniqueError(err) {
			return nil, ErrStatementExists
		}
		return nil, fmt.Errorf("crear estado importado: %w", err)
	}
	statementID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("id de estado importado: %w", err)
	}

	for i := range transactions {
		t := &transactions[i]
		var category any
		if t.CategoryID != nil {
			category = *t.CategoryID
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO card_transactions (
				card_id, statement_id, posted_at, concept, category_id, type,
				amount_local, amount_usd, currency_original, status, external_ref, note
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, st.CardID, statementID, t.PostedAt, t.Concept, category, t.Type,
			t.AmountLocal, t.AmountUSD, t.CurrencyOriginal, t.Status, t.ExternalRef, t.Note); err != nil {
			if isUniqueError(err) {
				return nil, ErrStatementExists
			}
			return nil, fmt.Errorf("crear movimiento importado: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmar importación: %w", err)
	}
	return s.GetStatement(ctx, statementID)
}

const transactionColumns = `
	t.id, t.card_id, t.statement_id, t.posted_at, t.concept, t.category_id,
	COALESCE(cat.name, ''), t.type, t.amount_local, t.amount_usd,
	t.currency_original, t.status, t.external_ref, t.note, t.created_at, t.updated_at
`

const transactionFrom = `
	FROM card_transactions t
	LEFT JOIN categories cat ON cat.id = t.category_id
`

// ListTransactions devuelve los movimientos de un estado con filtros opcionales.
func (s *Storage) ListTransactions(ctx context.Context, statementID int64, txType, query string) ([]Transaction, error) {
	sqlQuery := `SELECT ` + transactionColumns + transactionFrom + ` WHERE t.statement_id = ?`
	args := []any{statementID}
	if txType != "" {
		sqlQuery += ` AND t.type = ?`
		args = append(args, txType)
	}
	if query != "" {
		sqlQuery += ` AND t.concept LIKE ?`
		args = append(args, "%"+query+"%")
	}
	sqlQuery += ` ORDER BY t.posted_at ASC, t.id ASC`

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("listar movimientos: %w", err)
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		t, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar movimientos: %w", err)
	}
	return transactions, nil
}

// GetTransaction devuelve un movimiento o nil si no existe.
func (s *Storage) GetTransaction(ctx context.Context, id int64) (*Transaction, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+transactionColumns+transactionFrom+` WHERE t.id = ?`, id)
	t, err := scanTransaction(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

// CreateTransaction inserta un movimiento.
func (s *Storage) CreateTransaction(ctx context.Context, t *Transaction) (*Transaction, error) {
	var category any
	if t.CategoryID != nil {
		category = *t.CategoryID
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO card_transactions (
			card_id, statement_id, posted_at, concept, category_id, type,
			amount_local, amount_usd, currency_original, status, external_ref, note
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.CardID, t.StatementID, t.PostedAt, t.Concept, category, t.Type,
		t.AmountLocal, t.AmountUSD, t.CurrencyOriginal, t.Status, t.ExternalRef, t.Note)
	if err != nil {
		if isUniqueError(err) {
			return nil, ErrStatementExists
		}
		return nil, fmt.Errorf("crear movimiento: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("id de movimiento: %w", err)
	}
	return s.GetTransaction(ctx, id)
}

// UpdateTransaction actualiza los campos mutables del movimiento.
func (s *Storage) UpdateTransaction(ctx context.Context, t *Transaction) (*Transaction, error) {
	var category any
	if t.CategoryID != nil {
		category = *t.CategoryID
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE card_transactions SET
			posted_at = ?, concept = ?, category_id = ?, type = ?,
			amount_local = ?, amount_usd = ?, currency_original = ?, status = ?,
			note = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, t.PostedAt, t.Concept, category, t.Type, t.AmountLocal, t.AmountUSD,
		t.CurrencyOriginal, t.Status, t.Note, t.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar movimiento: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("filas de movimiento: %w", err)
	}
	if affected == 0 {
		return nil, ErrTransactionNotFound
	}
	return s.GetTransaction(ctx, t.ID)
}

// DeleteTransaction elimina un movimiento.
func (s *Storage) DeleteTransaction(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM card_transactions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("eliminar movimiento: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("filas de movimiento: %w", err)
	}
	if affected == 0 {
		return ErrTransactionNotFound
	}
	return nil
}

// StatementSummary agrega los movimientos del estado por tipo.
func (s *Storage) StatementSummary(ctx context.Context, statementID int64) ([]StatementSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT type, COALESCE(SUM(amount_local), 0), COALESCE(SUM(amount_usd), 0), COUNT(*)
		FROM card_transactions
		WHERE statement_id = ?
		GROUP BY type
		ORDER BY type
	`, statementID)
	if err != nil {
		return nil, fmt.Errorf("resumir movimientos: %w", err)
	}
	defer rows.Close()

	var summary []StatementSummary
	for rows.Next() {
		var item StatementSummary
		if err := rows.Scan(&item.Type, &item.AmountLocal, &item.AmountUSD, &item.Count); err != nil {
			return nil, fmt.Errorf("leer resumen: %w", err)
		}
		summary = append(summary, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar resumen: %w", err)
	}
	return summary, nil
}

// LatestStatementBalance devuelve el último corte y su saldo.
func (s *Storage) LatestStatementBalance(ctx context.Context, cardID int64) (id int64, periodEnd string, local, usd float64, found bool, err error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, period_end, new_balance, new_balance_usd
		FROM card_statements
		WHERE card_id = ?
		ORDER BY period_end DESC
		LIMIT 1
	`, cardID)
	err = row.Scan(&id, &periodEnd, &local, &usd)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", 0, 0, false, nil
	}
	if err != nil {
		return 0, "", 0, 0, false, fmt.Errorf("último corte: %w", err)
	}
	return id, periodEnd, local, usd, true, nil
}

// SumTransactionsAfter suma los movimientos posteriores a una fecha.
func (s *Storage) SumTransactionsAfter(ctx context.Context, cardID int64, periodEnd string) (local, usd float64, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount_local), 0), COALESCE(SUM(amount_usd), 0)
		FROM card_transactions
		WHERE card_id = ? AND posted_at > ?
	`, cardID, periodEnd).Scan(&local, &usd)
	if err != nil {
		return 0, 0, fmt.Errorf("sumar movimientos posteriores: %w", err)
	}
	return local, usd, nil
}

func scanCard(row rowScanner) (*CreditCard, error) {
	var card CreditCard
	var secondaryID sql.NullInt64
	var secondaryCode sql.NullString
	if err := row.Scan(
		&card.ID, &card.BankID, &card.BankName, &card.Alias, &card.Network,
		&card.Last4, &card.Holder, &card.CurrencyID, &card.CurrencyCode,
		&secondaryID, &secondaryCode, &card.CreditLimit, &card.CreditLimitUSD,
		&card.StatementDay, &card.PaymentDueDay, &card.MinPaymentPct,
		&card.InterestCurrentRate, &card.InterestLateRate,
		&card.InterestCurrentRateUSD, &card.InterestLateRateUSD,
		&card.Color, &card.Icon, &card.Active, &card.Notes, &card.StatementCount,
		&card.CreatedAt, &card.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if secondaryID.Valid {
		id := secondaryID.Int64
		card.SecondaryCurrencyID = &id
	}
	if secondaryCode.Valid {
		card.SecondaryCurrencyCode = secondaryCode.String
	}
	return &card, nil
}

func scanStatement(row rowScanner) (*Statement, error) {
	var st Statement
	var periodStart sql.NullString
	if err := row.Scan(
		&st.ID, &st.CardID, &periodStart, &st.PeriodEnd, &st.MinPaymentDueDate,
		&st.FullPaymentDueDate, &st.PreviousBalance, &st.PreviousBalanceUSD,
		&st.InterestCurrent, &st.InterestCurrentUSD,
		&st.InterestBonifiable, &st.InterestBonifiableUSD,
		&st.MoraAmount, &st.MoraAmountUSD, &st.ArrearsCount,
		&st.OverLimitAmount, &st.OverLimitAmountUSD,
		&st.PointsPrevious, &st.PointsEarned, &st.PointsUsed, &st.PointsAvailable,
		&st.MinPayment, &st.MinPaymentUSD,
		&st.TotalPayment, &st.TotalPaymentUSD, &st.NewBalance, &st.NewBalanceUSD,
		&st.Status, &st.Source, &st.SourceModuleID, &st.ImportHash, &st.Notes,
		&st.TransactionCount, &st.CreatedAt, &st.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if periodStart.Valid {
		value := periodStart.String
		st.PeriodStart = &value
	}
	return &st, nil
}

func scanTransaction(row rowScanner) (*Transaction, error) {
	var t Transaction
	var statementID, categoryID sql.NullInt64
	if err := row.Scan(
		&t.ID, &t.CardID, &statementID, &t.PostedAt, &t.Concept, &categoryID,
		&t.CategoryName, &t.Type, &t.AmountLocal, &t.AmountUSD, &t.CurrencyOriginal,
		&t.Status, &t.ExternalRef, &t.Note, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if statementID.Valid {
		id := statementID.Int64
		t.StatementID = &id
	}
	if categoryID.Valid {
		id := categoryID.Int64
		t.CategoryID = &id
	}
	return &t, nil
}

func isUniqueError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
