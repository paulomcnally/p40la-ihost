package rental

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// IssueInvoiceInput agrupa los cargos a facturar.
type IssueInvoiceInput struct {
	LeaseID   int64   `json:"lease_id"`
	ChargeIDs []int64 `json:"charge_ids"`
	DueDate   string  `json:"due_date"`
	Notes     string  `json:"notes"`
}

// CreatePaymentInput son los datos de un pago.
type CreatePaymentInput struct {
	LeaseID     int64   `json:"lease_id"`
	InvoiceID   *int64  `json:"invoice_id,omitempty"`
	PaidAt      string  `json:"paid_at"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Method      string  `json:"method"`
	Reference   string  `json:"reference"`
	Notes       string  `json:"notes"`
	WithReceipt bool    `json:"with_receipt"`
}

// FileUpload representa un archivo recibido por multipart.
type FileUpload struct {
	FileName string
	Reader   io.Reader
}

// ---------------------------------------------------------------- cargos

// ListCharges devuelve los cargos de un contrato.
func (s *Service) ListCharges(ctx context.Context, leaseID int64) ([]Charge, error) {
	if _, err := s.GetLease(ctx, leaseID); err != nil {
		return nil, err
	}
	return s.storage.ListCharges(ctx, leaseID)
}

// GenerateCharges genera los cargos de renta y mantenimiento del plazo completo.
// Es idempotente por (contrato, período, tipo).
func (s *Service) GenerateCharges(ctx context.Context, leaseID int64) (int, error) {
	l, err := s.storage.GetLease(ctx, leaseID)
	if err != nil {
		return 0, err
	}
	if l == nil {
		return 0, ErrLeaseNotFound
	}
	start, err := time.Parse("2006-01-02", l.StartDate)
	if err != nil {
		return 0, invalid("invalid_date", "Fecha de inicio inválida")
	}
	end, err := time.Parse("2006-01-02", l.EndDate)
	if err != nil {
		return 0, invalid("invalid_date", "Fecha de fin inválida")
	}
	created := 0
	firstMonth := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < l.TermMonths; i++ {
		m := firstMonth.AddDate(0, i, 0)
		period := m.Format("2006-01")
		due := time.Date(m.Year(), m.Month(), l.PaymentDay, 0, 0, 0, 0, time.UTC)
		if due.Before(start) {
			due = start
		}
		if due.After(end) {
			due = end
		}
		if l.RentAmount > 0 {
			ok, err := s.storage.CreateCharge(ctx, &Charge{
				LeaseID: leaseID, Kind: ChargeRent, Period: period,
				Description: "Renta " + period, Amount: l.RentAmount,
				Currency: l.Currency, DueDate: due.Format("2006-01-02"), Status: ChargePending,
			})
			if err != nil {
				return created, err
			}
			if ok {
				created++
			}
		}
		if l.MaintenanceAmount > 0 {
			ok, err := s.storage.CreateCharge(ctx, &Charge{
				LeaseID: leaseID, Kind: ChargeMaintenance, Period: period,
				Description: "Mantenimiento " + period, Amount: l.MaintenanceAmount,
				Currency: l.Currency, DueDate: due.Format("2006-01-02"), Status: ChargePending,
			})
			if err != nil {
				return created, err
			}
			if ok {
				created++
			}
		}
	}
	if l.DepositAmount > 0 {
		ok, err := s.storage.CreateCharge(ctx, &Charge{
			LeaseID: leaseID, Kind: ChargeDeposit, Period: firstMonth.Format("2006-01"),
			Description: "Depósito de garantía", Amount: l.DepositAmount,
			Currency: l.Currency, DueDate: l.StartDate, Status: ChargePending,
		})
		if err != nil {
			return created, err
		}
		if ok {
			created++
		}
	}
	return created, nil
}

// Balance devuelve cargos, pagos y saldo de un contrato.
func (s *Service) Balance(ctx context.Context, leaseID int64) (*Balance, error) {
	l, err := s.storage.GetLease(ctx, leaseID)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrLeaseNotFound
	}
	charged, err := s.storage.SumCharged(ctx, leaseID)
	if err != nil {
		return nil, err
	}
	paid, err := s.storage.SumPaid(ctx, leaseID)
	if err != nil {
		return nil, err
	}
	return &Balance{
		Charged: round2(charged), Paid: round2(paid),
		Balance: round2(charged - paid), Currency: l.Currency,
	}, nil
}

// ---------------------------------------------------------------- facturas

// ListInvoices lista facturas.
func (s *Service) ListInvoices(ctx context.Context, leaseID int64, status string) ([]Invoice, error) {
	return s.storage.ListInvoices(ctx, leaseID, status)
}

// GetInvoice devuelve una factura.
func (s *Service) GetInvoice(ctx context.Context, id int64) (*Invoice, error) {
	inv, err := s.storage.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, ErrInvoiceNotFound
	}
	return inv, nil
}

// IssueInvoice emite una factura con numeración atómica y PDF.
func (s *Service) IssueInvoice(ctx context.Context, in IssueInvoiceInput) (*Invoice, error) {
	lease, err := s.storage.GetLease(ctx, in.LeaseID)
	if err != nil {
		return nil, err
	}
	if lease == nil {
		return nil, ErrLeaseNotFound
	}
	if lease.Status != LeaseActive {
		return nil, invalid("lease_not_active", "Solo se puede facturar un contrato activo")
	}
	if len(in.ChargeIDs) == 0 {
		return nil, invalid("charges_required", "Seleccioná al menos un cargo")
	}
	dueDate := strings.TrimSpace(in.DueDate)
	if dueDate == "" {
		dueDate = s.now().UTC().Format("2006-01-02")
	}
	if !validDate(dueDate) {
		return nil, invalid("invalid_date", "La fecha de vencimiento es inválida")
	}
	var lines []InvoiceLine
	var chargeIDs []int64
	var total float64
	for _, id := range in.ChargeIDs {
		c, err := s.storage.GetCharge(ctx, id)
		if err != nil {
			return nil, err
		}
		if c == nil || c.LeaseID != lease.ID {
			return nil, ErrChargeNotFound
		}
		if c.Status != ChargePending {
			return nil, invalid("charge_not_pending", "Uno de los cargos ya está facturado o pagado")
		}
		if c.Currency != lease.Currency {
			return nil, invalid("currency_mismatch", "Los cargos deben estar en la moneda del contrato")
		}
		lines = append(lines, InvoiceLine{ChargeID: &c.ID, Description: c.Description, Amount: c.Amount})
		chargeIDs = append(chargeIDs, c.ID)
		total += c.Amount
	}
	total = round2(total)
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	tenantNames := s.tenantNames(ctx, lease.ID)
	home := ""
	if lease.HomeName != "" {
		home = lease.HomeName
		if lease.HomeAddress != "" {
			home += " — " + lease.HomeAddress
		}
	}
	snap := InvoiceSnapshot{
		Issuer:              s.issuer(settings),
		IssueDate:           s.now().UTC().Format("2006-01-02"),
		DueDate:             dueDate,
		Status:              "issued",
		Currency:            lease.Currency,
		HomeName:            lease.HomeName,
		HomeAddress:         lease.HomeAddress,
		LeaseStart:          lease.StartDate,
		LeaseEnd:            lease.EndDate,
		TenantNames:         tenantNames,
		Subtotal:            total,
		Total:               total,
		PaymentInstructions: lease.OwnerBankAccount,
		Notes:               strings.TrimSpace(in.Notes),
	}
	for _, line := range lines {
		snap.Lines = append(snap.Lines, PDFLine{Description: line.Description, Amount: line.Amount})
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("serializar snapshot: %w", err)
	}
	inv := &Invoice{
		LeaseID: lease.ID, Series: settings.InvoiceSeries,
		Year: s.now().UTC().Year(), Status: "issued",
		IssueDate: snap.IssueDate, DueDate: dueDate, Currency: lease.Currency,
		Subtotal: total, Total: total, Snapshot: string(raw),
	}
	if err := s.storage.CreateInvoiceTx(ctx, inv, lines, chargeIDs); err != nil {
		return nil, err
	}
	// El número se conoce recién después de la transacción.
	snap.NumberLabel = inv.NumberLabel
	raw, _ = json.Marshal(snap)
	inv.Snapshot = string(raw)
	if err := s.storage.SetInvoiceSnapshot(ctx, inv.ID, string(raw)); err != nil {
		return nil, err
	}
	if err := s.regenerateInvoicePDF(ctx, inv.ID); err != nil {
		return nil, err
	}
	return s.GetInvoice(ctx, inv.ID)
}

// EnsureInvoicePDF genera el PDF si falta y devuelve la ruta.
func (s *Service) EnsureInvoicePDF(ctx context.Context, id int64) (string, error) {
	inv, err := s.storage.GetInvoice(ctx, id)
	if err != nil {
		return "", err
	}
	if inv == nil {
		return "", ErrInvoiceNotFound
	}
	if inv.PDFPath != "" {
		if _, err := os.Stat(inv.PDFPath); err == nil {
			return inv.PDFPath, nil
		}
	}
	if err := s.regenerateInvoicePDF(ctx, id); err != nil {
		return "", err
	}
	inv, err = s.storage.GetInvoice(ctx, id)
	if err != nil || inv == nil {
		return "", ErrInvoiceNotFound
	}
	return inv.PDFPath, nil
}

func (s *Service) regenerateInvoicePDF(ctx context.Context, id int64) error {
	inv, err := s.storage.GetInvoice(ctx, id)
	if err != nil {
		return err
	}
	if inv == nil {
		return ErrInvoiceNotFound
	}
	var snap InvoiceSnapshot
	if err := json.Unmarshal([]byte(inv.Snapshot), &snap); err != nil {
		return fmt.Errorf("leer snapshot de factura: %w", err)
	}
	snap.Status = inv.Status
	snap.VoidReason = inv.VoidReason
	if snap.NumberLabel == "" {
		snap.NumberLabel = inv.NumberLabel
	}
	data, err := RenderInvoice(snap.Issuer, snap)
	if err != nil {
		return err
	}
	dir := filepath.Join(s.dataDir, "rental", "leases", fmt.Sprintf("%d", inv.LeaseID), "invoices")
	path, hash, err := writeRentalFile(dir, fmt.Sprintf("%s.pdf", inv.NumberLabel), data)
	if err != nil {
		return err
	}
	return s.storage.SetInvoicePDF(ctx, id, path, hash, int64(len(data)))
}

// InvoicePDF devuelve el contenido del PDF de una factura.
func (s *Service) InvoicePDF(ctx context.Context, id int64) ([]byte, string, error) {
	path, err := s.EnsureInvoicePDF(ctx, id)
	if err != nil {
		return nil, "", err
	}
	inv, err := s.storage.GetInvoice(ctx, id)
	if err != nil || inv == nil {
		return nil, "", ErrInvoiceNotFound
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("leer PDF de factura: %w", err)
	}
	return data, inv.NumberLabel + ".pdf", nil
}

// SendInvoice envía la factura con su PDF a las partes notificables.
func (s *Service) SendInvoice(ctx context.Context, id int64) (int, error) {
	inv, err := s.storage.GetInvoice(ctx, id)
	if err != nil {
		return 0, err
	}
	if inv == nil {
		return 0, ErrInvoiceNotFound
	}
	if inv.Status == "void" {
		return 0, invalid("invoice_void", "No se puede enviar una factura anulada")
	}
	date := s.now().UTC().Format("2006-01-02")
	occurrence := fmt.Sprintf("%d-%s", id, date)
	exists, err := s.storage.EmailLogExists(ctx, "invoice", id, occurrence)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, invalid("already_sent", "La factura ya se envió hoy")
	}
	path, err := s.EnsureInvoicePDF(ctx, id)
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("leer PDF de factura: %w", err)
	}
	lease, err := s.storage.GetLease(ctx, inv.LeaseID)
	if err != nil || lease == nil {
		return 0, ErrLeaseNotFound
	}
	people, err := s.storage.NotifyParties(ctx, inv.LeaseID, []string{RoleTenant, RoleGuarantor})
	if err != nil {
		return 0, err
	}
	var to []string
	for _, p := range people {
		to = append(to, p.Email)
	}
	if len(to) == 0 {
		return 0, invalid("no_recipients", "No hay partes con email para notificar")
	}
	title := "Factura de alquiler " + inv.NumberLabel
	content := fmt.Sprintf(
		`<p>Adjuntamos la factura <strong>%s</strong> por <strong>%s</strong>.</p>
		 <p>Vivienda: %s<br>Vencimiento: %s</p>
		 <p>Este es un documento interno de cobro y no constituye factura fiscal.</p>`,
		html.EscapeString(inv.NumberLabel), html.EscapeString(formatMoney(inv.Total, inv.Currency)),
		html.EscapeString(lease.HomeName), html.EscapeString(inv.DueDate))
	body := s.email.RenderTemplate(ctx, title, content)
	err = s.email.SendWithAttachments(ctx, to, "P40LA — "+title, body, []services.EmailAttachment{{
		Name: inv.NumberLabel + ".pdf", ContentType: "application/pdf", Data: data,
	}})
	if err != nil {
		_ = s.storage.InsertEmailLog(ctx, "invoice", id, occurrence, len(to), "tenant,guarantor", "failed", "smtp_error")
		return 0, err
	}
	if err := s.storage.InsertEmailLog(ctx, "invoice", id, occurrence, len(to), "tenant,guarantor", "sent", ""); err != nil {
		return 0, err
	}
	if err := s.storage.MarkInvoiceSent(ctx, id); err != nil {
		return 0, err
	}
	return len(to), nil
}

// VoidInvoice anula una factura conservando el número.
func (s *Service) VoidInvoice(ctx context.Context, id int64, reason string) (*Invoice, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, invalid("reason_required", "El motivo de anulación es obligatorio")
	}
	inv, err := s.storage.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, ErrInvoiceNotFound
	}
	if inv.Status == "void" {
		return inv, nil
	}
	if err := s.storage.VoidInvoiceTx(ctx, id, reason); err != nil {
		return nil, err
	}
	if err := s.regenerateInvoicePDF(ctx, id); err != nil {
		return nil, err
	}
	return s.GetInvoice(ctx, id)
}

// ---------------------------------------------------------------- pagos

// ListPayments lista pagos de un contrato o factura.
func (s *Service) ListPayments(ctx context.Context, leaseID, invoiceID int64) ([]Payment, error) {
	payments, err := s.storage.ListPayments(ctx, leaseID, invoiceID)
	if err != nil {
		return nil, err
	}
	for i := range payments {
		proofs, err := s.storage.ListProofs(ctx, payments[i].ID)
		if err == nil {
			payments[i].Proofs = proofs
		}
		receipt, err := s.storage.GetReceiptByPayment(ctx, payments[i].ID)
		if err == nil && receipt != nil {
			payments[i].ReceiptID = &receipt.ID
		}
	}
	return payments, nil
}

// CreatePayment registra un pago y, opcionalmente, emite su recibo.
func (s *Service) CreatePayment(ctx context.Context, in CreatePaymentInput) (*Payment, error) {
	lease, err := s.storage.GetLease(ctx, in.LeaseID)
	if err != nil {
		return nil, err
	}
	if lease == nil {
		return nil, ErrLeaseNotFound
	}
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.Currency == "" {
		in.Currency = lease.Currency
	}
	if in.Currency != lease.Currency {
		return nil, invalid("currency_mismatch", "El pago debe estar en la moneda del contrato")
	}
	if in.Amount <= 0 {
		return nil, invalid("invalid_amount", "El monto del pago debe ser mayor que cero")
	}
	paidAt := strings.TrimSpace(in.PaidAt)
	if paidAt == "" {
		paidAt = s.now().UTC().Format("2006-01-02")
	}
	if !validDate(paidAt) {
		return nil, invalid("invalid_date", "La fecha de pago es inválida")
	}
	if in.Method == "" {
		in.Method = "transfer"
	}
	if !contains(PaymentMethods, in.Method) {
		return nil, invalid("invalid_method", "Método de pago inválido")
	}
	if in.InvoiceID != nil {
		inv, err := s.storage.GetInvoice(ctx, *in.InvoiceID)
		if err != nil {
			return nil, err
		}
		if inv == nil || inv.LeaseID != lease.ID {
			return nil, ErrInvoiceNotFound
		}
		if inv.Status == "void" {
			return nil, invalid("invoice_void", "No se puede pagar una factura anulada")
		}
	}
	payment := &Payment{
		LeaseID: lease.ID, InvoiceID: in.InvoiceID, PaidAt: paidAt,
		Amount: round2(in.Amount), Currency: in.Currency, Method: in.Method,
		Reference: strings.TrimSpace(in.Reference), Notes: strings.TrimSpace(in.Notes),
	}
	if err := s.storage.CreatePaymentTx(ctx, payment); err != nil {
		return nil, err
	}
	if in.WithReceipt {
		receipt, err := s.CreateReceipt(ctx, payment.ID)
		if err != nil {
			return nil, err
		}
		payment.ReceiptID = &receipt.ID
	}
	return payment, nil
}

// CreateReceipt emite el recibo de un pago.
func (s *Service) CreateReceipt(ctx context.Context, paymentID int64) (*Receipt, error) {
	existing, err := s.storage.GetReceiptByPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	payment, err := s.storage.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if payment == nil {
		return nil, ErrPaymentNotFound
	}
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	receipt := &Receipt{
		PaymentID: paymentID, Series: settings.ReceiptSeries,
		Year: s.now().UTC().Year(), Status: "issued",
		IssueDate: s.now().UTC().Format("2006-01-02"),
		Amount:    payment.Amount, Currency: payment.Currency,
	}
	if err := s.storage.CreateReceiptTx(ctx, receipt); err != nil {
		return nil, err
	}
	if err := s.regenerateReceiptPDF(ctx, receipt.ID); err != nil {
		return nil, err
	}
	return s.storage.GetReceipt(ctx, receipt.ID)
}

// ReceiptPDF devuelve el PDF del recibo.
func (s *Service) ReceiptPDF(ctx context.Context, id int64) ([]byte, string, error) {
	receipt, err := s.storage.GetReceipt(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if receipt == nil {
		return nil, "", ErrReceiptNotFound
	}
	path := filepath.Join(s.dataDir, "rental", "receipts", fmt.Sprintf("%d.pdf", receipt.ID))
	if _, err := os.Stat(path); err != nil {
		if err := s.regenerateReceiptPDF(ctx, id); err != nil {
			return nil, "", err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("leer PDF de recibo: %w", err)
	}
	label := fmt.Sprintf("%s-%d-%04d", receipt.Series, receipt.Year, receipt.Number)
	return data, label + ".pdf", nil
}

func (s *Service) regenerateReceiptPDF(ctx context.Context, id int64) error {
	receipt, err := s.storage.GetReceipt(ctx, id)
	if err != nil {
		return err
	}
	if receipt == nil {
		return ErrReceiptNotFound
	}
	payment, err := s.storage.GetPayment(ctx, receipt.PaymentID)
	if err != nil || payment == nil {
		return ErrPaymentNotFound
	}
	lease, _ := s.storage.GetLease(ctx, payment.LeaseID)
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}
	snap := ReceiptSnapshot{
		Issuer:      s.issuer(settings),
		NumberLabel: fmt.Sprintf("%s-%d-%04d", receipt.Series, receipt.Year, receipt.Number),
		IssueDate:   receipt.IssueDate,
		Currency:    receipt.Currency,
		Amount:      receipt.Amount,
		Method:      payment.Method,
		Reference:   payment.Reference,
		TenantNames: s.tenantNames(ctx, payment.LeaseID),
		Notes:       payment.Notes,
	}
	if lease != nil {
		snap.HomeName = lease.HomeName
	}
	data, err := RenderReceipt(snap)
	if err != nil {
		return err
	}
	path, hash, err := writeRentalFile(filepath.Join(s.dataDir, "rental", "receipts"),
		fmt.Sprintf("%d.pdf", receipt.ID), data)
	if err != nil {
		return err
	}
	return s.storage.SetReceiptPDF(ctx, receipt.ID, path, hash, int64(len(data)))
}

// UploadProof guarda un comprobante de pago.
func (s *Service) UploadProof(ctx context.Context, paymentID int64, upload FileUpload) (*Proof, error) {
	payment, err := s.storage.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if payment == nil {
		return nil, ErrPaymentNotFound
	}
	count, err := s.storage.CountProofs(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if count >= MaxPaymentProofs {
		return nil, ErrProofLimit
	}
	saved, err := s.storeRentalFile(ctx, upload, []string{"pdf", "png", "jpeg"}, MaxRentalAttachmentBytes)
	if err != nil {
		return nil, err
	}
	proof := &Proof{
		PaymentID: paymentID, OriginalName: saved.OriginalName,
		ContentType: saved.ContentType, SizeBytes: saved.Size,
		SHA256: saved.SHA256,
	}
	proof, err = s.storage.CreateProof(ctx, proof, saved.Path)
	if err != nil {
		_ = os.Remove(saved.Path)
		return nil, err
	}
	return proof, nil
}

// ProofPath devuelve la ruta de un comprobante.
func (s *Service) ProofPath(ctx context.Context, paymentID, proofID int64) (*Proof, string, error) {
	proof, path, err := s.storage.GetProofPath(ctx, paymentID, proofID)
	if err != nil {
		return nil, "", err
	}
	if _, err := os.Stat(path); err != nil {
		return nil, "", ErrProofNotFound
	}
	return proof, path, nil
}

// DeleteProof elimina un comprobante y su archivo.
func (s *Service) DeleteProof(ctx context.Context, paymentID, proofID int64) error {
	proof, path, err := s.storage.GetProofPath(ctx, paymentID, proofID)
	if err != nil {
		return err
	}
	if proof == nil {
		return ErrProofNotFound
	}
	if err := s.storage.DeleteProof(ctx, paymentID, proofID); err != nil {
		return err
	}
	_ = os.Remove(path)
	return nil
}

// ------------------------------------------------------ documentos del contrato

// ListDocuments lista documentos legales.
func (s *Service) ListDocuments(ctx context.Context, leaseID int64) ([]LeaseDocument, error) {
	return s.storage.ListDocuments(ctx, leaseID)
}

// UploadDocument adjunta un documento legal PDF al contrato.
func (s *Service) UploadDocument(ctx context.Context, leaseID int64, docType string, upload FileUpload) (*LeaseDocument, error) {
	l, err := s.storage.GetLease(ctx, leaseID)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrLeaseNotFound
	}
	if !contains(DocumentTypes, docType) {
		return nil, invalid("invalid_doc_type", "Tipo de documento inválido")
	}
	count, err := s.storage.CountDocuments(ctx, leaseID)
	if err != nil {
		return nil, err
	}
	if count >= MaxLeaseDocuments {
		return nil, ErrDocumentLimit
	}
	saved, err := s.storeRentalFileInDir(upload, filepath.Join(s.dataDir, "rental", "leases", fmt.Sprintf("%d", leaseID), "documents"), []string{"pdf"}, MaxRentalAttachmentBytes)
	if err != nil {
		return nil, err
	}
	doc := &LeaseDocument{
		LeaseID: leaseID, DocType: docType, OriginalName: saved.OriginalName,
		ContentType: saved.ContentType, SizeBytes: saved.Size, SHA256: saved.SHA256,
	}
	doc, err = s.storage.CreateDocument(ctx, doc, saved.Path)
	if err != nil {
		_ = os.Remove(saved.Path)
		return nil, err
	}
	return doc, nil
}

// DocumentPath devuelve la ruta de un documento.
func (s *Service) DocumentPath(ctx context.Context, leaseID, docID int64) (*LeaseDocument, string, error) {
	doc, path, err := s.storage.GetDocument(ctx, leaseID, docID)
	if err != nil {
		return nil, "", err
	}
	if _, err := os.Stat(path); err != nil {
		return nil, "", ErrDocumentNotFound
	}
	return doc, path, nil
}

// DeleteDocument elimina un documento y su archivo.
func (s *Service) DeleteDocument(ctx context.Context, leaseID, docID int64) error {
	doc, path, err := s.storage.GetDocument(ctx, leaseID, docID)
	if err != nil {
		return err
	}
	if doc == nil {
		return ErrDocumentNotFound
	}
	if err := s.storage.DeleteDocument(ctx, leaseID, docID); err != nil {
		return err
	}
	_ = os.Remove(path)
	return nil
}

// ------------------------------------------------------- acuerdo de reserva PDF

// ReservationPDF genera el acuerdo de reserva en PDF.
func (s *Service) ReservationPDF(ctx context.Context, id int64) ([]byte, string, error) {
	r, err := s.GetReservation(ctx, id)
	if err != nil {
		return nil, "", err
	}
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return nil, "", err
	}
	snap := ReservationSnapshot{
		Issuer: s.issuer(settings), HomeName: r.HomeName, HomeAddress: r.HomeAddress,
		Amount: r.Amount, Currency: r.Currency, Reference: r.Reference,
		AppliedToDeposit: r.AppliedToDeposit, StartsAt: r.StartsAt, ExpiresAt: r.ExpiresAt,
		ContractDate: r.ContractDate, MoveInDate: r.MoveInDate,
		RentAmount: r.RentAmount, MaintenanceAmount: r.MaintenanceAmount,
		OwnerSplitPercent: r.OwnerSplitPercent, AgencySplitPercent: r.AgencySplitPercent,
		RefundPolicy: r.RefundPolicy, RequiredDocuments: r.RequiredDocuments,
		Notes: r.Notes, GeneratedAt: s.now().UTC().Format("2006-01-02"),
	}
	if r.Applicant != nil {
		snap.ApplicantName = r.Applicant.FullName
		snap.ApplicantTaxID = r.Applicant.TaxID
	}
	snap.AgentName = r.AgentName
	if r.AgentPersonID != nil {
		if agent, err := s.storage.GetPerson(ctx, *r.AgentPersonID); err == nil && agent != nil {
			snap.AgentName = agent.FullName
		}
	}
	data, err := RenderReservation(snap)
	if err != nil {
		return nil, "", err
	}
	name := fmt.Sprintf("acuerdo-reserva-%d.pdf", r.ID)
	return data, name, nil
}

// ---------------------------------------------------------------- helpers

type savedFile struct {
	OriginalName string
	ContentType  string
	Size         int64
	SHA256       string
	Path         string
}

func (s *Service) storeRentalFile(ctx context.Context, upload FileUpload, allowed []string, maxBytes int64) (*savedFile, error) {
	paymentDir := filepath.Join(s.dataDir, "rental", "uploads")
	return s.storeRentalFileInDir(upload, paymentDir, allowed, maxBytes)
}

func (s *Service) storeRentalFileInDir(upload FileUpload, dir string, allowed []string, maxBytes int64) (*savedFile, error) {
	original := sanitizeRentalName(upload.FileName)
	data, err := io.ReadAll(io.LimitReader(upload.Reader, maxBytes+1))
	if err != nil {
		return nil, invalid("invalid_file", "No se pudo leer el archivo")
	}
	if len(data) == 0 {
		return nil, invalid("invalid_file", "El archivo está vacío")
	}
	if int64(len(data)) > maxBytes {
		return nil, invalid("invalid_file", "El archivo supera el máximo de 10 MB")
	}
	kind := detectFileKind(data)
	if kind == "" || !contains(allowed, kind) {
		return nil, invalid("invalid_file", "Tipo de archivo no permitido")
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	ext := map[string]string{"pdf": ".pdf", "png": ".png", "jpeg": ".jpg"}[kind]
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("crear directorio de archivos: %w", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), hash[:12], ext))
	if err := writeAtomic(path, data); err != nil {
		return nil, err
	}
	contentType := map[string]string{
		"pdf": "application/pdf", "png": "image/png", "jpeg": "image/jpeg",
	}[kind]
	return &savedFile{
		OriginalName: original, ContentType: contentType,
		Size: int64(len(data)), SHA256: hash, Path: path,
	}, nil
}

func writeRentalFile(dir, name string, data []byte) (string, string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("crear directorio: %w", err)
	}
	path := filepath.Join(dir, name)
	if err := writeAtomic(path, data); err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(data)
	return path, hex.EncodeToString(sum[:]), nil
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("escribir archivo: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("mover archivo: %w", err)
	}
	return nil
}

func detectFileKind(data []byte) string {
	switch {
	case len(data) >= 5 && string(data[:5]) == "%PDF-":
		return "pdf"
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "png"
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "jpeg"
	default:
		return ""
	}
}

func sanitizeRentalName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." {
		name = "documento"
	}
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}

func (s *Service) issuer(settings *RentalSettings) PDFIssuer {
	return PDFIssuer{
		Name: settings.IssuerName, TaxID: settings.IssuerTaxID,
		Address: settings.IssuerAddress, Email: settings.IssuerEmail,
	}
}

func (s *Service) tenantNames(ctx context.Context, leaseID int64) []string {
	parties, err := s.storage.ListParties(ctx, leaseID)
	if err != nil {
		return nil
	}
	var names []string
	for _, p := range parties {
		if p.Role == RoleTenant && p.Person != nil {
			names = append(names, p.Person.FullName)
		}
	}
	return names
}
