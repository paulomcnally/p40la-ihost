package rental

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ------------------------------------------------------------------ cargos

const chargeColumns = `
	id, lease_id, kind, period, description, amount, currency, due_date,
	status, invoice_id, created_at, updated_at
`

func scanCharge(row interface{ Scan(...any) error }) (*Charge, error) {
	var c Charge
	var invoiceID sql.NullInt64
	err := row.Scan(
		&c.ID, &c.LeaseID, &c.Kind, &c.Period, &c.Description, &c.Amount,
		&c.Currency, &c.DueDate, &c.Status, &invoiceID, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear cargo: %w", err)
	}
	if invoiceID.Valid {
		c.InvoiceID = &invoiceID.Int64
	}
	return &c, nil
}

// ListCharges lista los cargos de un contrato.
func (s *Storage) ListCharges(ctx context.Context, leaseID int64) ([]Charge, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+chargeColumns+` FROM rental_charges
		WHERE lease_id = ? ORDER BY period ASC, kind ASC, id ASC
	`, leaseID)
	if err != nil {
		return nil, fmt.Errorf("listar cargos: %w", err)
	}
	defer rows.Close()
	var out []Charge
	for rows.Next() {
		c, err := scanCharge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// ListChargesByStatus lista cargos de un contrato por estado.
func (s *Storage) ListChargesByStatus(ctx context.Context, leaseID int64, status string) ([]Charge, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+chargeColumns+` FROM rental_charges
		WHERE lease_id = ? AND status = ? ORDER BY period ASC, kind ASC, id ASC
	`, leaseID, status)
	if err != nil {
		return nil, fmt.Errorf("listar cargos: %w", err)
	}
	defer rows.Close()
	var out []Charge
	for rows.Next() {
		c, err := scanCharge(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// GetCharge devuelve un cargo o nil.
func (s *Storage) GetCharge(ctx context.Context, id int64) (*Charge, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+chargeColumns+` FROM rental_charges WHERE id = ?`, id)
	return scanCharge(row)
}

// CreateCharge inserta un cargo idempotente por (lease, period, kind).
// Devuelve inserted=false si ya existía.
func (s *Storage) CreateCharge(ctx context.Context, c *Charge) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO rental_charges (lease_id, kind, period, description, amount, currency, due_date, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(lease_id, period, kind) DO NOTHING
	`, c.LeaseID, c.Kind, c.Period, c.Description, c.Amount, c.Currency, c.DueDate, c.Status)
	if err != nil {
		return false, fmt.Errorf("insertar cargo: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// SumCharged suma los cargos no anulados.
func (s *Storage) SumCharged(ctx context.Context, leaseID int64) (float64, error) {
	var total float64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM rental_charges WHERE lease_id = ? AND status <> 'void'
	`, leaseID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sumar cargos: %w", err)
	}
	return total, nil
}

// SetChargesInvoiceStatus actualiza los cargos de una factura.
func (s *Storage) SetChargesInvoiceStatus(ctx context.Context, tx dbtx, invoiceID int64, status string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE rental_charges SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE invoice_id = ?
	`, status, invoiceID)
	if err != nil {
		return fmt.Errorf("actualizar estado de cargos: %w", err)
	}
	return nil
}

// VoidChargesByInvoice libera los cargos de una factura anulada.
func (s *Storage) VoidChargesByInvoice(ctx context.Context, tx dbtx, invoiceID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE rental_charges SET status = 'pending', invoice_id = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE invoice_id = ?
	`, invoiceID)
	if err != nil {
		return fmt.Errorf("liberar cargos de factura anulada: %w", err)
	}
	return nil
}

// ------------------------------------------------------------- numeración

// nextSequence reserva el siguiente número de una serie dentro de una transacción.
func (s *Storage) nextSequence(ctx context.Context, tx dbtx, kind, series string, year int) (int64, error) {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO rental_sequences (kind, series, year, next_number) VALUES (?, ?, ?, 2)
		ON CONFLICT(kind, series, year) DO UPDATE SET next_number = next_number + 1
	`, kind, series, year)
	if err != nil {
		return 0, fmt.Errorf("reservar número de %s: %w", kind, err)
	}
	var number int64
	if err := tx.QueryRowContext(ctx, `
		SELECT next_number - 1 FROM rental_sequences WHERE kind = ? AND series = ? AND year = ?
	`, kind, series, year).Scan(&number); err != nil {
		return 0, fmt.Errorf("leer número de %s: %w", kind, err)
	}
	return number, nil
}

// ------------------------------------------------------------------ facturas

const invoiceColumns = `
	id, lease_id, series, number, year, status, issue_date, due_date, currency,
	subtotal, total, snapshot_json, pdf_path, pdf_sha256, pdf_size_bytes,
	sent_at, void_reason, voided_at, created_at
`

func scanInvoice(row interface{ Scan(...any) error }) (*Invoice, error) {
	var inv Invoice
	var sentAt, voidedAt sql.NullTime
	err := row.Scan(
		&inv.ID, &inv.LeaseID, &inv.Series, &inv.Number, &inv.Year, &inv.Status,
		&inv.IssueDate, &inv.DueDate, &inv.Currency, &inv.Subtotal, &inv.Total,
		&inv.Snapshot, &inv.PDFPath, &inv.PDFSHA256, &inv.PDFSize,
		&sentAt, &inv.VoidReason, &voidedAt, &inv.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear factura: %w", err)
	}
	if sentAt.Valid {
		inv.SentAt = &sentAt.Time
	}
	if voidedAt.Valid {
		inv.VoidedAt = &voidedAt.Time
	}
	inv.NumberLabel = fmt.Sprintf("%s-%d-%04d", inv.Series, inv.Year, inv.Number)
	return &inv, nil
}

// ListInvoices lista facturas por contrato y estado.
func (s *Storage) ListInvoices(ctx context.Context, leaseID int64, status string) ([]Invoice, error) {
	q := `SELECT ` + invoiceColumns + ` FROM rental_invoices WHERE 1=1`
	args := []any{}
	if leaseID > 0 {
		q += ` AND lease_id = ?`
		args = append(args, leaseID)
	}
	if status != "" {
		q += ` AND status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY year DESC, number DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listar facturas: %w", err)
	}
	defer rows.Close()
	var out []Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *inv)
	}
	return out, rows.Err()
}

// GetInvoice devuelve una factura con líneas.
func (s *Storage) GetInvoice(ctx context.Context, id int64) (*Invoice, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+invoiceColumns+` FROM rental_invoices WHERE id = ?`, id)
	inv, err := scanInvoice(row)
	if err != nil || inv == nil {
		return inv, err
	}
	lines, err := s.ListInvoiceLines(ctx, id)
	if err != nil {
		return nil, err
	}
	inv.Lines = lines
	return inv, nil
}

// ListInvoiceLines lista las líneas de una factura.
func (s *Storage) ListInvoiceLines(ctx context.Context, invoiceID int64) ([]InvoiceLine, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, invoice_id, charge_id, description, amount
		FROM rental_invoice_lines WHERE invoice_id = ? ORDER BY id
	`, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("listar líneas: %w", err)
	}
	defer rows.Close()
	var out []InvoiceLine
	for rows.Next() {
		var l InvoiceLine
		var chargeID sql.NullInt64
		if err := rows.Scan(&l.ID, &l.InvoiceID, &chargeID, &l.Description, &l.Amount); err != nil {
			return nil, fmt.Errorf("escanear línea: %w", err)
		}
		if chargeID.Valid {
			l.ChargeID = &chargeID.Int64
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// CreateInvoiceTx inserta factura, líneas y marca cargos en una transacción.
func (s *Storage) CreateInvoiceTx(ctx context.Context, inv *Invoice, lines []InvoiceLine, chargeIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar factura: %w", err)
	}
	defer tx.Rollback()

	number, err := s.nextSequence(ctx, tx, "invoice", inv.Series, inv.Year)
	if err != nil {
		return err
	}
	inv.Number = number

	res, err := tx.ExecContext(ctx, `
		INSERT INTO rental_invoices (
			lease_id, series, number, year, status, issue_date, due_date, currency,
			subtotal, total, snapshot_json
		) VALUES (?, ?, ?, ?, 'issued', ?, ?, ?, ?, ?, ?)
	`, inv.LeaseID, inv.Series, number, inv.Year, inv.IssueDate, inv.DueDate,
		inv.Currency, inv.Subtotal, inv.Total, inv.Snapshot)
	if err != nil {
		return fmt.Errorf("insertar factura: %w", err)
	}
	invoiceID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("obtener id de factura: %w", err)
	}
	inv.ID = invoiceID
	inv.Status = "issued"
	inv.NumberLabel = fmt.Sprintf("%s-%d-%04d", inv.Series, inv.Year, number)

	for i := range lines {
		lines[i].InvoiceID = invoiceID
		res, err := tx.ExecContext(ctx, `
			INSERT INTO rental_invoice_lines (invoice_id, charge_id, description, amount)
			VALUES (?, ?, ?, ?)
		`, invoiceID, nullableID(lines[i].ChargeID), lines[i].Description, lines[i].Amount)
		if err != nil {
			return fmt.Errorf("insertar línea de factura: %w", err)
		}
		lines[i].ID, _ = res.LastInsertId()
	}
	for _, chargeID := range chargeIDs {
		if _, err := tx.ExecContext(ctx, `
			UPDATE rental_charges SET status = 'invoiced', invoice_id = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND lease_id = ? AND status = 'pending'
		`, invoiceID, chargeID, inv.LeaseID); err != nil {
			return fmt.Errorf("marcar cargo como facturado: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar factura: %w", err)
	}
	inv.Lines = lines
	return nil
}

// SetInvoicePDF guarda la ruta y el hash del PDF generado.
func (s *Storage) SetInvoicePDF(ctx context.Context, id int64, path, hash string, size int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rental_invoices SET pdf_path = ?, pdf_sha256 = ?, pdf_size_bytes = ? WHERE id = ?
	`, path, hash, size, id)
	if err != nil {
		return fmt.Errorf("guardar PDF de factura: %w", err)
	}
	return nil
}

// MarkInvoiceSent registra el envío de la factura.
func (s *Storage) MarkInvoiceSent(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rental_invoices SET sent_at = CURRENT_TIMESTAMP WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("marcar factura enviada: %w", err)
	}
	return nil
}

// VoidInvoiceTx anula la factura y libera sus cargos.
func (s *Storage) VoidInvoiceTx(ctx context.Context, id int64, reason string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar anulación: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		UPDATE rental_invoices
		SET status = 'void', void_reason = ?, voided_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'issued'
	`, reason, id)
	if err != nil {
		return fmt.Errorf("anular factura: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrInvoiceImmutable
	}
	if err := s.VoidChargesByInvoice(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

// --------------------------------------------------------------------- pagos

const paymentColumns = `
	id, lease_id, invoice_id, paid_at, amount, currency, method, reference, notes, created_at
`

func scanPayment(row interface{ Scan(...any) error }) (*Payment, error) {
	var p Payment
	var invoiceID sql.NullInt64
	err := row.Scan(&p.ID, &p.LeaseID, &invoiceID, &p.PaidAt, &p.Amount, &p.Currency,
		&p.Method, &p.Reference, &p.Notes, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear pago: %w", err)
	}
	if invoiceID.Valid {
		p.InvoiceID = &invoiceID.Int64
	}
	return &p, nil
}

// ListPayments lista pagos por contrato o factura.
func (s *Storage) ListPayments(ctx context.Context, leaseID, invoiceID int64) ([]Payment, error) {
	q := `SELECT ` + paymentColumns + ` FROM rental_payments WHERE 1=1`
	args := []any{}
	if leaseID > 0 {
		q += ` AND lease_id = ?`
		args = append(args, leaseID)
	}
	if invoiceID > 0 {
		q += ` AND invoice_id = ?`
		args = append(args, invoiceID)
	}
	q += ` ORDER BY paid_at DESC, id DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listar pagos: %w", err)
	}
	defer rows.Close()
	var out []Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// GetPayment devuelve un pago o nil.
func (s *Storage) GetPayment(ctx context.Context, id int64) (*Payment, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+paymentColumns+` FROM rental_payments WHERE id = ?`, id)
	return scanPayment(row)
}

// CreatePaymentTx inserta el pago y actualiza el estado de los cargos de la factura.
func (s *Storage) CreatePaymentTx(ctx context.Context, p *Payment) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar pago: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO rental_payments (lease_id, invoice_id, paid_at, amount, currency, method, reference, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, p.LeaseID, nullableID(p.InvoiceID), p.PaidAt, p.Amount, p.Currency, p.Method, p.Reference, p.Notes)
	if err != nil {
		return fmt.Errorf("insertar pago: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("obtener id de pago: %w", err)
	}
	p.ID = id

	if p.InvoiceID != nil {
		var total, paid float64
		if err := tx.QueryRowContext(ctx, `SELECT total FROM rental_invoices WHERE id = ?`, *p.InvoiceID).Scan(&total); err != nil {
			return fmt.Errorf("leer total de factura: %w", err)
		}
		if err := tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(amount), 0) FROM rental_payments WHERE invoice_id = ?
		`, *p.InvoiceID).Scan(&paid); err != nil {
			return fmt.Errorf("sumar pagos de factura: %w", err)
		}
		status := ChargePartiallyPaid
		if paid+0.005 >= total {
			status = ChargePaid
		}
		if err := s.SetChargesInvoiceStatus(ctx, tx, *p.InvoiceID, status); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar pago: %w", err)
	}
	return nil
}

// SumPaid suma los pagos de un contrato.
func (s *Storage) SumPaid(ctx context.Context, leaseID int64) (float64, error) {
	var total float64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM rental_payments WHERE lease_id = ?
	`, leaseID).Scan(&total); err != nil {
		return 0, fmt.Errorf("sumar pagos: %w", err)
	}
	return total, nil
}

// SumPaidInvoice suma los pagos de una factura.
func (s *Storage) SumPaidInvoice(ctx context.Context, invoiceID int64) (float64, error) {
	var total float64
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM rental_payments WHERE invoice_id = ?
	`, invoiceID).Scan(&total); err != nil {
		return 0, fmt.Errorf("sumar pagos de factura: %w", err)
	}
	return total, nil
}

// ------------------------------------------------------------------ recibos

const receiptColumns = `
	id, payment_id, series, number, year, status, issue_date, amount, currency,
	pdf_path, pdf_sha256, pdf_size_bytes, sent_at, created_at
`

func scanReceipt(row interface{ Scan(...any) error }) (*Receipt, error) {
	var r Receipt
	var sentAt sql.NullTime
	var pdfPath string
	err := row.Scan(&r.ID, &r.PaymentID, &r.Series, &r.Number, &r.Year, &r.Status,
		&r.IssueDate, &r.Amount, &r.Currency, &pdfPath, &r.PDFSHA256, &r.PDFSize, &sentAt, &r.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear recibo: %w", err)
	}
	if sentAt.Valid {
		r.SentAt = &sentAt.Time
	}
	return &r, nil
}

// GetReceiptByPayment devuelve el recibo de un pago.
func (s *Storage) GetReceiptByPayment(ctx context.Context, paymentID int64) (*Receipt, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+receiptColumns+` FROM rental_receipts WHERE payment_id = ?`, paymentID)
	return scanReceipt(row)
}

// CreateReceiptTx emite un recibo con numeración atómica.
func (s *Storage) CreateReceiptTx(ctx context.Context, r *Receipt) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar recibo: %w", err)
	}
	defer tx.Rollback()

	number, err := s.nextSequence(ctx, tx, "receipt", r.Series, r.Year)
	if err != nil {
		return err
	}
	r.Number = number
	res, err := tx.ExecContext(ctx, `
		INSERT INTO rental_receipts (payment_id, series, number, year, status, issue_date, amount, currency)
		VALUES (?, ?, ?, ?, 'issued', ?, ?, ?)
	`, r.PaymentID, r.Series, number, r.Year, r.IssueDate, r.Amount, r.Currency)
	if err != nil {
		return fmt.Errorf("insertar recibo: %w", err)
	}
	r.ID, _ = res.LastInsertId()
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar recibo: %w", err)
	}
	return nil
}

// SetReceiptPDF guarda el PDF del recibo.
func (s *Storage) SetReceiptPDF(ctx context.Context, id int64, path, hash string, size int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rental_receipts SET pdf_path = ?, pdf_sha256 = ?, pdf_size_bytes = ? WHERE id = ?
	`, path, hash, size, id)
	if err != nil {
		return fmt.Errorf("guardar PDF de recibo: %w", err)
	}
	return nil
}

// GetReceipt devuelve un recibo con su pago.
func (s *Storage) GetReceipt(ctx context.Context, id int64) (*Receipt, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+receiptColumns+` FROM rental_receipts WHERE id = ?`, id)
	return scanReceipt(row)
}

// --------------------------------------------------------------- comprobantes

const proofColumns = `
	id, payment_id, original_name, content_type, size_bytes, sha256, storage_path, created_at
`

func scanProof(row interface{ Scan(...any) error }) (*Proof, error) {
	var p Proof
	var path string
	err := row.Scan(&p.ID, &p.PaymentID, &p.OriginalName, &p.ContentType,
		&p.SizeBytes, &p.SHA256, &path, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("escanear comprobante: %w", err)
	}
	return &p, nil
}

// ListProofs lista comprobantes de un pago.
func (s *Storage) ListProofs(ctx context.Context, paymentID int64) ([]Proof, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+proofColumns+` FROM rental_payment_proofs WHERE payment_id = ? ORDER BY id
	`, paymentID)
	if err != nil {
		return nil, fmt.Errorf("listar comprobantes: %w", err)
	}
	defer rows.Close()
	var out []Proof
	for rows.Next() {
		p, err := scanProof(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// CountProofs cuenta comprobantes de un pago.
func (s *Storage) CountProofs(ctx context.Context, paymentID int64) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rental_payment_proofs WHERE payment_id = ?`, paymentID).Scan(&n); err != nil {
		return 0, fmt.Errorf("contar comprobantes: %w", err)
	}
	return n, nil
}

// CreateProof registra un comprobante.
func (s *Storage) CreateProof(ctx context.Context, p *Proof, path string) (*Proof, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO rental_payment_proofs (payment_id, original_name, content_type, size_bytes, sha256, storage_path)
		VALUES (?, ?, ?, ?, ?, ?)
	`, p.PaymentID, p.OriginalName, p.ContentType, p.SizeBytes, p.SHA256, path)
	if err != nil {
		return nil, fmt.Errorf("insertar comprobante: %w", err)
	}
	p.ID, _ = res.LastInsertId()
	return p, nil
}

// GetProofPath devuelve la ruta y metadatos de un comprobante.
func (s *Storage) GetProofPath(ctx context.Context, paymentID, proofID int64) (*Proof, string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+proofColumns+` FROM rental_payment_proofs WHERE id = ? AND payment_id = ?
	`, proofID, paymentID)
	var p Proof
	var path string
	err := row.Scan(&p.ID, &p.PaymentID, &p.OriginalName, &p.ContentType,
		&p.SizeBytes, &p.SHA256, &path, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrProofNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("obtener comprobante: %w", err)
	}
	return &p, path, nil
}

// DeleteProof elimina un comprobante.
func (s *Storage) DeleteProof(ctx context.Context, paymentID, proofID int64) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM rental_payment_proofs WHERE id = ? AND payment_id = ?
	`, proofID, paymentID)
	if err != nil {
		return fmt.Errorf("eliminar comprobante: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrProofNotFound
	}
	return nil
}

// ------------------------------------------------------- documentos del contrato

const documentColumns = `
	id, lease_id, doc_type, original_name, content_type, size_bytes, sha256, storage_path, created_at
`

func scanDocument(row interface{ Scan(...any) error }) (*LeaseDocument, string, error) {
	var d LeaseDocument
	var path string
	err := row.Scan(&d.ID, &d.LeaseID, &d.DocType, &d.OriginalName, &d.ContentType,
		&d.SizeBytes, &d.SHA256, &path, &d.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("escanear documento: %w", err)
	}
	return &d, path, nil
}

// ListDocuments lista los documentos de un contrato.
func (s *Storage) ListDocuments(ctx context.Context, leaseID int64) ([]LeaseDocument, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+documentColumns+` FROM rental_lease_documents WHERE lease_id = ? ORDER BY id DESC
	`, leaseID)
	if err != nil {
		return nil, fmt.Errorf("listar documentos: %w", err)
	}
	defer rows.Close()
	var out []LeaseDocument
	for rows.Next() {
		d, _, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

// CountDocuments cuenta documentos de un contrato.
func (s *Storage) CountDocuments(ctx context.Context, leaseID int64) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rental_lease_documents WHERE lease_id = ?`, leaseID).Scan(&n); err != nil {
		return 0, fmt.Errorf("contar documentos: %w", err)
	}
	return n, nil
}

// CreateDocument registra un documento.
func (s *Storage) CreateDocument(ctx context.Context, d *LeaseDocument, path string) (*LeaseDocument, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO rental_lease_documents (lease_id, doc_type, original_name, content_type, size_bytes, sha256, storage_path)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, d.LeaseID, d.DocType, d.OriginalName, d.ContentType, d.SizeBytes, d.SHA256, path)
	if err != nil {
		return nil, fmt.Errorf("insertar documento: %w", err)
	}
	d.ID, _ = res.LastInsertId()
	return d, nil
}

// GetDocument devuelve un documento con su ruta.
func (s *Storage) GetDocument(ctx context.Context, leaseID, docID int64) (*LeaseDocument, string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+documentColumns+` FROM rental_lease_documents WHERE id = ? AND lease_id = ?
	`, docID, leaseID)
	d, path, err := scanDocument(row)
	if err != nil {
		return nil, "", err
	}
	if d == nil {
		return nil, "", ErrDocumentNotFound
	}
	return d, path, nil
}

// DeleteDocument elimina un documento.
func (s *Storage) DeleteDocument(ctx context.Context, leaseID, docID int64) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM rental_lease_documents WHERE id = ? AND lease_id = ?
	`, docID, leaseID)
	if err != nil {
		return fmt.Errorf("eliminar documento: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrDocumentNotFound
	}
	return nil
}

// SetInvoiceSnapshot actualiza el snapshot congelado (por ejemplo, al anular).
func (s *Storage) SetInvoiceSnapshot(ctx context.Context, id int64, snapshot string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE rental_invoices SET snapshot_json = ? WHERE id = ?
	`, snapshot, id)
	if err != nil {
		return fmt.Errorf("guardar snapshot de factura: %w", err)
	}
	return nil
}

// OldestUnpaidInvoice devuelve la factura emitida con saldo pendiente más antigua.
func (s *Storage) OldestUnpaidInvoice(ctx context.Context, leaseID int64) (*Invoice, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+invoiceColumns+`
		FROM rental_invoices i
		WHERE i.lease_id = ? AND i.status = 'issued'
			AND i.total > (
				SELECT COALESCE(SUM(p.amount), 0)
				FROM rental_payments p
				WHERE p.invoice_id = i.id
			)
		ORDER BY i.issue_date ASC, i.year ASC, i.number ASC
		LIMIT 1
	`, leaseID)
	return scanInvoice(row)
}
