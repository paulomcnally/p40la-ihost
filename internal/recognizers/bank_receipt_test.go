package recognizers

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/testpdf"
)

const debitNoteText = `Comprobante
Nota de débito
Fecha: 01/02/2026
Hora 09:15:22 (GMT-0600)
Referencia 902000001
Nombre Cliente Demo
Cuenta bancaria *****0000
Detalle Lote Demo - Cuota 7
Moneda Dólares
Monto 123.45
Banco Demo`

const transferText = `Confirmación de Transferencia
Número de Comprobante 52700001
De Cuenta de...0000
Para Proveedor Demo ****-0000
Fecha y Hora 01/02/2026 - 23:02
Monto debitado $ 123.45
Tipo de Cambio $ 1.00
Monto acreditado $ 123.45
Motivo Lote Demo - Cuota 40`

func TestParseBankReceiptText_DebitNote(t *testing.T) {
	doc, err := parseBankReceiptText(debitNoteText)
	if err != nil {
		t.Fatalf("parseBankReceiptText() error = %v", err)
	}

	if doc.DocumentType != DocumentTypeDebitNote {
		t.Errorf("DocumentType = %q, want %q", doc.DocumentType, DocumentTypeDebitNote)
	}
	if doc.Reference != "902000001" {
		t.Errorf("Reference = %q, want 902000001", doc.Reference)
	}
	if doc.Amount != 123.45 {
		t.Errorf("Amount = %v, want 123.45", doc.Amount)
	}
	if doc.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", doc.Currency)
	}
	if doc.InstallmentNumber != 7 {
		t.Errorf("InstallmentNumber = %d, want 7", doc.InstallmentNumber)
	}
	if doc.Concept != "Lote Demo - Cuota 7" {
		t.Errorf("Concept = %q, want %q", doc.Concept, "Lote Demo - Cuota 7")
	}
	if doc.PaidAt == nil {
		t.Fatal("PaidAt no debería ser nil")
	}
	if got := doc.PaidAt.Format("2006-01-02 15:04:05 -0700"); got != "2026-02-01 09:15:22 -0600" {
		t.Errorf("PaidAt = %q, want 2026-02-01 09:15:22 -0600", got)
	}
	if len(doc.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none", doc.Warnings)
	}
	if doc.Confidence != 1 {
		t.Errorf("Confidence = %v, want 1", doc.Confidence)
	}
}

func TestParseBankReceiptText_Transfer(t *testing.T) {
	doc, err := parseBankReceiptText(transferText)
	if err != nil {
		t.Fatalf("parseBankReceiptText() error = %v", err)
	}

	if doc.DocumentType != DocumentTypeTransfer {
		t.Errorf("DocumentType = %q, want %q", doc.DocumentType, DocumentTypeTransfer)
	}
	if doc.Reference != "52700001" {
		t.Errorf("Reference = %q, want 52700001", doc.Reference)
	}
	if doc.Amount != 123.45 {
		t.Errorf("Amount = %v, want 123.45", doc.Amount)
	}
	if doc.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", doc.Currency)
	}
	if doc.InstallmentNumber != 40 {
		t.Errorf("InstallmentNumber = %d, want 40", doc.InstallmentNumber)
	}
	if doc.Concept != "Lote Demo - Cuota 40" {
		t.Errorf("Concept = %q, want %q", doc.Concept, "Lote Demo - Cuota 40")
	}
	if doc.PaidAt == nil {
		t.Fatal("PaidAt no debería ser nil")
	}
	if got := doc.PaidAt.Format("2006-01-02 15:04"); got != "2026-02-01 23:02" {
		t.Errorf("PaidAt = %q, want 2026-02-01 23:02", got)
	}
}

func TestParseBankReceiptText_AmountWithThousands(t *testing.T) {
	text := `Nota de débito
Referencia 902000002
Fecha: 03/03/2026
Moneda Córdobas
Monto 4,567.89
Detalle Lote Demo - Cuota 3`

	doc, err := parseBankReceiptText(text)
	if err != nil {
		t.Fatalf("parseBankReceiptText() error = %v", err)
	}
	if doc.Amount != 4567.89 {
		t.Errorf("Amount = %v, want 4567.89", doc.Amount)
	}
	if doc.Currency != "NIO" {
		t.Errorf("Currency = %q, want NIO", doc.Currency)
	}
}

func TestParseBankReceiptText_PaymentDateWinsOverEmissionDate(t *testing.T) {
	text := `Comprobante
Hora 09:15:22 (GMT-0600) | Fecha: 01/02/2026
Nota de débito
Fecha 03/02/2026
Referencia 902000010
Moneda Dólares
Monto 10.00
Detalle Lote Demo - Cuota 2`

	doc, err := parseBankReceiptText(text)
	if err != nil {
		t.Fatalf("parseBankReceiptText() error = %v", err)
	}
	if doc.PaidAt == nil {
		t.Fatal("PaidAt no debería ser nil")
	}
	if got := doc.PaidAt.Format("2006-01-02"); got != "2026-02-03" {
		t.Errorf("PaidAt = %q, want 2026-02-03 (fecha de pago, no de emisión)", got)
	}
}

func TestSanitizeConcept_RedactsPII(t *testing.T) {
	value := SanitizeConcept("Cliente Demo 1234567890 demo@example.com Cuota 7")
	if strings.Contains(value, "1234567890") {
		t.Errorf("SanitizeConcept no redactó los dígitos largos: %q", value)
	}
	if strings.Contains(value, "demo@example.com") {
		t.Errorf("SanitizeConcept no redactó el email: %q", value)
	}
	if !strings.Contains(value, "Cuota 7") {
		t.Errorf("SanitizeConcept perdió el concepto útil: %q", value)
	}
}

func TestParseBankReceiptText_MissingReference(t *testing.T) {
	text := `Nota de débito
Fecha: 03/03/2026
Monto 123.45
Detalle Lote Demo - Cuota 3`

	doc, err := parseBankReceiptText(text)
	if err != nil {
		t.Fatalf("parseBankReceiptText() error = %v", err)
	}
	if doc.Reference != "" {
		t.Errorf("Reference = %q, want empty", doc.Reference)
	}
	if !slices.Contains(doc.Warnings, "sin_referencia") {
		t.Errorf("Warnings = %v, want sin_referencia", doc.Warnings)
	}
	if doc.Confidence >= 0.6 {
		t.Errorf("Confidence = %v, should be low without reference", doc.Confidence)
	}
}

func TestParseBankReceiptText_Unsupported(t *testing.T) {
	_, err := parseBankReceiptText("Factura de otra empresa, S.A.")
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("error = %v, want ErrUnsupportedFormat", err)
	}
}

func TestBankReceiptRecognizer_ExtractsTextFromPDF(t *testing.T) {
	pdfBytes := testpdf.Build([]string{
		"Comprobante",
		"Nota de debito",
		"Fecha: 01/02/2026",
		"Hora 09:15:22 (GMT-0600)",
		"Referencia 902000001",
		"Detalle Lote Demo - Cuota 7",
		"Moneda Dolares",
		"Monto 123.45",
	})

	doc, err := Recognize(Source{
		ReaderAt: bytes.NewReader(pdfBytes),
		Size:     int64(len(pdfBytes)),
		MimeType: "application/pdf",
	})
	if err != nil {
		t.Fatalf("Recognize() error = %v", err)
	}
	if doc.Reference != "902000001" {
		t.Errorf("Reference = %q, want 902000001", doc.Reference)
	}
	if doc.Amount != 123.45 {
		t.Errorf("Amount = %v, want 123.45", doc.Amount)
	}
	if doc.InstallmentNumber != 7 {
		t.Errorf("InstallmentNumber = %d, want 7", doc.InstallmentNumber)
	}
	if doc.RecognizerID != "bank_receipt_v1" {
		t.Errorf("RecognizerID = %q, want bank_receipt_v1", doc.RecognizerID)
	}
}

func TestBankReceiptRecognizer_NeedsOCR(t *testing.T) {
	pdfBytes := testpdf.Build(nil)

	_, err := Recognize(Source{
		ReaderAt: bytes.NewReader(pdfBytes),
		Size:     int64(len(pdfBytes)),
		MimeType: "application/pdf",
	})
	if !errors.Is(err, ErrNeedsOCR) {
		t.Fatalf("error = %v, want ErrNeedsOCR", err)
	}
}

func TestBankReceiptRecognizer_InvalidPDF(t *testing.T) {
	content := []byte("esto no es un pdf")
	_, err := Recognize(Source{
		ReaderAt: bytes.NewReader(content),
		Size:     int64(len(content)),
		MimeType: "application/pdf",
	})
	if !errors.Is(err, ErrInvalidPDF) {
		t.Fatalf("error = %v, want ErrInvalidPDF", err)
	}
}

func TestBankReceiptRecognizer_Info(t *testing.T) {
	recognizer := &BankReceiptRecognizer{}
	info := recognizer.Info()
	if info.ID != "bank_receipt_v1" {
		t.Errorf("ID = %q, want bank_receipt_v1", info.ID)
	}
	if info.Name == "" {
		t.Error("Name no debería estar vacío")
	}
}

func TestParseBankReceiptText_TimeZone(t *testing.T) {
	text := `Nota de débito
Referencia 902000003
Fecha: 31/12/2025
Hora 23:59:59 (GMT+0600)
Monto 10.00`

	doc, err := parseBankReceiptText(text)
	if err != nil {
		t.Fatalf("parseBankReceiptText() error = %v", err)
	}
	if doc.PaidAt == nil {
		t.Fatal("PaidAt no debería ser nil")
	}
	_, offset := doc.PaidAt.Zone()
	if offset != 6*3600 {
		t.Errorf("offset = %d, want %d", offset, 6*3600)
	}
	expected := time.Date(2025, 12, 31, 23, 59, 59, 0, doc.PaidAt.Location())
	if !doc.PaidAt.Equal(expected) {
		t.Errorf("PaidAt = %v, want %v", doc.PaidAt, expected)
	}
}
