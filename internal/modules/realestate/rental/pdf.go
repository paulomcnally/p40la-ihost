package rental

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// PDFIssuer son los datos del emisor configurables (nunca hardcodeados).
type PDFIssuer struct {
	Name    string `json:"name"`
	TaxID   string `json:"tax_id"`
	Address string `json:"address"`
	Email   string `json:"email"`
}

// PDFLine es una línea impresa.
type PDFLine struct {
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

// InvoiceSnapshot congela los datos de una factura al emitirse.
type InvoiceSnapshot struct {
	Issuer              PDFIssuer `json:"issuer"`
	NumberLabel         string    `json:"number_label"`
	IssueDate           string    `json:"issue_date"`
	DueDate             string    `json:"due_date"`
	Status              string    `json:"status"`
	Currency            string    `json:"currency"`
	HomeName            string    `json:"home_name"`
	HomeAddress         string    `json:"home_address"`
	LeaseStart          string    `json:"lease_start"`
	LeaseEnd            string    `json:"lease_end"`
	TenantNames         []string  `json:"tenant_names"`
	Lines               []PDFLine `json:"lines"`
	Subtotal            float64   `json:"subtotal"`
	Total               float64   `json:"total"`
	PaymentInstructions string    `json:"payment_instructions"`
	Notes               string    `json:"notes"`
	VoidReason          string    `json:"void_reason,omitempty"`
}

// ReceiptSnapshot congela los datos de un recibo.
type ReceiptSnapshot struct {
	Issuer      PDFIssuer `json:"issuer"`
	NumberLabel string    `json:"number_label"`
	IssueDate   string    `json:"issue_date"`
	Currency    string    `json:"currency"`
	Amount      float64   `json:"amount"`
	Method      string    `json:"method"`
	Reference   string    `json:"reference"`
	HomeName    string    `json:"home_name"`
	TenantNames []string  `json:"tenant_names"`
	Notes       string    `json:"notes"`
}

// ReservationSnapshot congela los datos del acuerdo de reserva.
type ReservationSnapshot struct {
	Issuer             PDFIssuer `json:"issuer"`
	HomeName           string    `json:"home_name"`
	HomeAddress        string    `json:"home_address"`
	ApplicantName      string    `json:"applicant_name"`
	ApplicantTaxID     string    `json:"applicant_tax_id"`
	AgentName          string    `json:"agent_name"`
	AgencyName         string    `json:"agency_name"`
	Amount             float64   `json:"amount"`
	Currency           string    `json:"currency"`
	Reference          string    `json:"reference"`
	AppliedToDeposit   bool      `json:"applied_to_deposit"`
	StartsAt           string    `json:"starts_at"`
	ExpiresAt          string    `json:"expires_at"`
	ContractDate       string    `json:"contract_date"`
	MoveInDate         string    `json:"move_in_date"`
	RentAmount         float64   `json:"rent_amount"`
	MaintenanceAmount  float64   `json:"maintenance_amount"`
	OwnerSplitPercent  float64   `json:"owner_split_percent"`
	AgencySplitPercent float64   `json:"agency_split_percent"`
	RefundPolicy       string    `json:"refund_policy"`
	RequiredDocuments  string    `json:"required_documents"`
	Notes              string    `json:"notes"`
	GeneratedAt        string    `json:"generated_at"`
}

// RenderInvoice genera la factura interna de cobro en PDF.
func RenderInvoice(s PDFIssuer, snap InvoiceSnapshot) ([]byte, error) {
	pdf := newRentalPDF()
	pdf.SetTitle("Factura interna de cobro", true)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	writePDFHeader(pdf, tr, "FACTURA INTERNA DE COBRO", snap.NumberLabel, snap.Status)

	pdf.SetFont("Helvetica", "", 10)
	writeKeyValue(pdf, tr, "Emisión", snap.IssueDate)
	writeKeyValue(pdf, tr, "Vencimiento", snap.DueDate)
	writeKeyValue(pdf, tr, "Moneda", snap.Currency)
	pdf.Ln(3)

	writeSectionTitle(pdf, tr, "Vivienda")
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 5, tr(snap.HomeName+" — "+snap.HomeAddress), "", "L", false)
	writeKeyValue(pdf, tr, "Contrato", snap.LeaseStart+" al "+snap.LeaseEnd)
	writeKeyValue(pdf, tr, "Cobrar a", strings.Join(snap.TenantNames, ", "))
	pdf.Ln(4)

	writeSectionTitle(pdf, tr, "Detalle")
	pdf.SetFillColor(240, 240, 240)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(130, 7, tr("Concepto"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 7, tr("Importe"), "1", 1, "R", true, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	for _, line := range snap.Lines {
		pdf.CellFormat(130, 6, tr(line.Description), "1", 0, "L", false, 0, "")
		pdf.CellFormat(40, 6, tr(formatMoney(line.Amount, snap.Currency)), "1", 1, "R", false, 0, "")
	}
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(130, 8, tr("TOTAL"), "1", 0, "L", false, 0, "")
	pdf.CellFormat(40, 8, tr(formatMoney(snap.Total, snap.Currency)), "1", 1, "R", false, 0, "")
	pdf.Ln(6)

	if snap.PaymentInstructions != "" {
		writeSectionTitle(pdf, tr, "Forma de pago")
		pdf.SetFont("Helvetica", "", 9)
		pdf.MultiCell(0, 5, tr(snap.PaymentInstructions), "", "L", false)
		pdf.Ln(3)
	}
	if snap.Notes != "" {
		writeSectionTitle(pdf, tr, "Notas")
		pdf.SetFont("Helvetica", "", 9)
		pdf.MultiCell(0, 5, tr(snap.Notes), "", "L", false)
	}
	if snap.Status == "void" {
		pdf.SetTextColor(180, 0, 0)
		pdf.SetFont("Helvetica", "B", 24)
		pdf.SetXY(35, 120)
		pdf.CellFormat(140, 20, tr("ANULADA"), "", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetY(200)
		pdf.SetFont("Helvetica", "", 9)
		pdf.MultiCell(0, 5, tr("Motivo de anulación: "+snap.VoidReason), "", "L", false)
	}

	writePDFFooter(pdf, tr, "Documento interno de cobro — no constituye factura fiscal.")
	return pdfBytes(pdf)
}

// RenderReceipt genera el recibo de pago en PDF.
func RenderReceipt(snap ReceiptSnapshot) ([]byte, error) {
	pdf := newRentalPDF()
	pdf.SetTitle("Recibo de pago", true)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	writePDFHeader(pdf, tr, "RECIBO DE PAGO", snap.NumberLabel, "issued")
	pdf.SetFont("Helvetica", "", 10)
	writeKeyValue(pdf, tr, "Fecha", snap.IssueDate)
	writeKeyValue(pdf, tr, "Importe", formatMoney(snap.Amount, snap.Currency))
	writeKeyValue(pdf, tr, "Método", snap.Method)
	if snap.Reference != "" {
		writeKeyValue(pdf, tr, "Referencia", snap.Reference)
	}
	writeKeyValue(pdf, tr, "Vivienda", snap.HomeName)
	writeKeyValue(pdf, tr, "Recibido de", strings.Join(snap.TenantNames, ", "))
	pdf.Ln(6)
	writeSectionTitle(pdf, tr, "Concepto")
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 5, tr(snap.Notes), "", "L", false)

	writePDFFooter(pdf, tr, "Comprobante interno de pago.")
	return pdfBytes(pdf)
}

// RenderReservation genera el acuerdo de reserva en PDF.
func RenderReservation(snap ReservationSnapshot) ([]byte, error) {
	pdf := newRentalPDF()
	pdf.SetTitle("Acuerdo de reserva", true)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	writePDFHeader(pdf, tr, "ACUERDO DE RESERVA", snap.GeneratedAt, "issued")
	pdf.SetFont("Helvetica", "", 10)

	writeSectionTitle(pdf, tr, "Propiedad")
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 5, tr(snap.HomeName+" — "+snap.HomeAddress), "", "L", false)
	pdf.Ln(2)

	writeSectionTitle(pdf, tr, "Partes")
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 5, tr("Solicitante: "+snap.ApplicantName+"  "+snap.ApplicantTaxID), "", "L", false)
	if snap.AgentName != "" || snap.AgencyName != "" {
		pdf.MultiCell(0, 5, tr("Agente: "+snap.AgentName+"  "+snap.AgencyName), "", "L", false)
	}
	pdf.Ln(2)

	writeSectionTitle(pdf, tr, "Condiciones")
	pdf.SetFont("Helvetica", "", 10)
	rows := [][2]string{
		{"Monto de reserva", formatMoney(snap.Amount, snap.Currency)},
		{"Referencia", snap.Reference},
		{"Vigencia", snap.StartsAt + " al " + snap.ExpiresAt},
		{"Firma de contrato", snap.ContractDate},
		{"Mudanza", snap.MoveInDate},
		{"Renta mensual", formatMoney(snap.RentAmount, snap.Currency)},
		{"Mantenimiento", formatMoney(snap.MaintenanceAmount, snap.Currency)},
		{"Política de devolución", snap.RefundPolicy},
	}
	for _, row := range rows {
		if row[1] == "" {
			continue
		}
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(55, 6, tr(row[0]+":"), "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.MultiCell(0, 6, tr(row[1]), "", "L", false)
	}
	pdf.Ln(2)

	pdf.SetFont("Helvetica", "", 9)
	if snap.AppliedToDeposit {
		pdf.MultiCell(0, 5, tr("La reserva se imputa al depósito de garantía del contrato de alquiler."), "", "L", false)
	}
	if snap.RefundPolicy == RefundForfeitable {
		pdf.MultiCell(0, 5, tr(fmt.Sprintf(
			"Si el solicitante desiste, la reserva se retiene como compensación: %.2f%% propietario y %.2f%% agencia.",
			snap.OwnerSplitPercent, snap.AgencySplitPercent)), "", "L", false)
	}
	if snap.RefundPolicy == RefundRefundable {
		pdf.MultiCell(0, 5, tr("Si el solicitante desiste, la reserva se devuelve en su totalidad."), "", "L", false)
	}
	if snap.RequiredDocuments != "" {
		pdf.Ln(2)
		pdf.MultiCell(0, 5, tr("Documentos exigidos al firmar: "+snap.RequiredDocuments), "", "L", false)
	}
	if snap.Notes != "" {
		pdf.Ln(2)
		pdf.MultiCell(0, 5, tr("Notas: "+snap.Notes), "", "L", false)
	}

	pdf.Ln(12)
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(80, 15, tr("Firma del propietario / agencia"), "T", 0, "C", false, 0, "")
	pdf.CellFormat(10, 15, "", "", 0, "C", false, 0, "")
	pdf.CellFormat(80, 15, tr("Firma del solicitante"), "T", 1, "C", false, 0, "")

	writePDFFooter(pdf, tr, "Documento interno de reserva. No constituye contrato de alquiler.")
	return pdfBytes(pdf)
}

// --------------------------------------------------------------- helpers PDF

func newRentalPDF() *fpdf.Fpdf {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 18, 20)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()
	pdf.SetCreator("p40la-ihost", true)
	return pdf
}

func pdfBytes(pdf *fpdf.Fpdf) ([]byte, error) {
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generar PDF: %w", err)
	}
	return buf.Bytes(), nil
}

func writePDFHeader(pdf *fpdf.Fpdf, tr func(string) string, title, label, status string) {
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 8, tr(title), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 5, tr(label), "", 1, "L", false, 0, "")
	pdf.Ln(4)
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(5)
}

func writePDFFooter(pdf *fpdf.Fpdf, tr func(string) string, legend string) {
	pdf.SetY(-25)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(120, 120, 120)
	pdf.MultiCell(0, 4, tr(legend), "", "C", false)
	pdf.SetTextColor(0, 0, 0)
}

func writeSectionTitle(pdf *fpdf.Fpdf, tr func(string) string, title string) {
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(0, 6, tr(title), "", 1, "L", false, 0, "")
}

func writeKeyValue(pdf *fpdf.Fpdf, tr func(string) string, key, value string) {
	if value == "" {
		return
	}
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(38, 5, tr(key+":"), "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.MultiCell(0, 5, tr(value), "", "L", false)
}

func formatMoney(amount float64, currency string) string {
	negative := amount < 0
	if negative {
		amount = -amount
	}
	whole := int64(amount)
	cents := int64((amount-float64(whole))*100 + 0.5)
	digits := fmt.Sprintf("%d", whole)
	var parts []string
	for len(digits) > 3 {
		parts = append([]string{digits[len(digits)-3:]}, parts...)
		digits = digits[:len(digits)-3]
	}
	parts = append([]string{digits}, parts...)
	out := strings.Join(parts, ",") + fmt.Sprintf(".%02d", cents)
	if negative {
		out = "-" + out
	}
	return strings.TrimSpace(currency + " " + out)
}

func timestampNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}
