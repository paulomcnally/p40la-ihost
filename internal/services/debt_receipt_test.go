package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

const testPDF = "%PDF-1.4\n1 0 obj\n<<>>\nendobj\n%%EOF\n"

func newDebtReceiptTestService(t *testing.T) (*DebtService, context.Context, int64, string) {
	t.Helper()

	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	if _, err := database.ExecContext(ctx, "INSERT INTO institutions (name) VALUES ('Banco Uno')"); err != nil {
		t.Fatalf("insertar institución: %v", err)
	}

	dataDir := t.TempDir()
	svc := NewDebtService(
		storage.NewDebtStorage(database),
		storage.NewDebtBillStorage(database),
		storage.NewInstitutionStorage(database),
		storage.NewCurrencyStorage(database),
		dataDir,
	)

	debt, err := svc.Create(ctx, &models.Debt{
		InstitutionID:     1,
		Identifier:        "D-122",
		Description:       "Deuda de prueba",
		Total:             100,
		Principal:         100,
		CurrencyID:        1,
		InstallmentsTotal: 1,
		InstallmentAmount: 100,
		PaymentDay:        5,
		StartDate:         "2026-09-01",
		Status:            DebtStatusActive,
	})
	if err != nil {
		t.Fatalf("crear deuda: %v", err)
	}

	return svc, ctx, debt.ID, dataDir
}

func firstBill(t *testing.T, svc *DebtService, ctx context.Context, debtID int64) models.DebtBill {
	t.Helper()
	bills, err := svc.ListBills(ctx, debtID)
	if err != nil || len(bills) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(bills), err)
	}
	return bills[0]
}

func documentPathFor(dataDir string, debtID int64, name string) string {
	return filepath.Join(dataDir, "documents", fmt.Sprintf("%d", debtID), name)
}

func TestDebtDocumentsPayStoresBothPDFs(t *testing.T) {
	svc, ctx, debtID, dataDir := newDebtReceiptTestService(t)
	bill := firstBill(t, svc, ctx, debtID)

	if bill.HasInvoice || bill.HasReceipt {
		t.Fatal("una cuota nueva no debería tener documentos")
	}

	paidAt := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	paid, err := svc.PayBillWithDocuments(
		ctx,
		bill.ID,
		paidAt,
		"demo-reference-001",
		&PDFUpload{FileName: "factura.pdf", Reader: bytes.NewReader([]byte(testPDF))},
		&PDFUpload{FileName: "comprobante.pdf", Reader: bytes.NewReader([]byte(testPDF))},
	)
	if err != nil {
		t.Fatalf("PayBillWithDocuments: %v", err)
	}
	if paid.Status != "paid" || !paid.HasInvoice || !paid.HasReceipt {
		t.Fatalf("cuota pagada con documentos inesperada: %+v", paid)
	}

	for _, name := range []string{"invoice_1.pdf", "receipt_1.pdf"} {
		data, err := os.ReadFile(documentPathFor(dataDir, debtID, name))
		if err != nil {
			t.Fatalf("leer %s: %v", name, err)
		}
		if !bytes.Equal(data, []byte(testPDF)) {
			t.Errorf("contenido de %s inesperado: %q", name, data)
		}
	}

	gotInvoicePath, gotInvoiceName, err := svc.DocumentPath(ctx, bill.ID, DebtDocumentInvoice)
	if err != nil {
		t.Fatalf("DocumentPath invoice: %v", err)
	}
	if gotInvoiceName != "invoice_1.pdf" {
		t.Errorf("nombre de factura = %q", gotInvoiceName)
	}
	gotReceiptPath, gotReceiptName, err := svc.DocumentPath(ctx, bill.ID, DebtDocumentReceipt)
	if err != nil {
		t.Fatalf("DocumentPath receipt: %v", err)
	}
	if gotReceiptName != "receipt_1.pdf" {
		t.Errorf("nombre de comprobante = %q", gotReceiptName)
	}
	if gotInvoicePath == gotReceiptPath {
		t.Error("factura y comprobante no deben compartir ruta")
	}

	reloaded, err := svc.ListBills(ctx, debtID)
	if err != nil || len(reloaded) != 1 {
		t.Fatalf("ListBills recargado: got %d, err %v", len(reloaded), err)
	}
	if !reloaded[0].HasInvoice || !reloaded[0].HasReceipt {
		t.Errorf("ListBills debería devolver ambos flags: %+v", reloaded[0])
	}

	debts, err := svc.List(ctx)
	if err != nil || len(debts) != 1 {
		t.Fatalf("List: got %d, err %v", len(debts), err)
	}
	if !debts[0].HasDocuments {
		t.Error("List debería devolver has_documents=true")
	}
	detail, err := svc.GetByID(ctx, debtID)
	if err != nil || detail == nil {
		t.Fatalf("GetByID: %v, %v", detail, err)
	}
	if !detail.HasDocuments {
		t.Error("GetByID debería devolver has_documents=true")
	}
}

func TestDebtDocumentsPayWithOnlyInvoice(t *testing.T) {
	svc, ctx, debtID, dataDir := newDebtReceiptTestService(t)
	bill := firstBill(t, svc, ctx, debtID)

	paid, err := svc.PayBillWithDocuments(
		ctx,
		bill.ID,
		time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
		"",
		&PDFUpload{FileName: "factura.pdf", Reader: bytes.NewReader([]byte(testPDF))},
		nil,
	)
	if err != nil {
		t.Fatalf("PayBillWithDocuments: %v", err)
	}
	if !paid.HasInvoice || paid.HasReceipt {
		t.Fatalf("flags inesperados: %+v", paid)
	}
	if _, err := os.Stat(documentPathFor(dataDir, debtID, "invoice_1.pdf")); err != nil {
		t.Errorf("falta invoice_1.pdf: %v", err)
	}
	if _, err := os.Stat(documentPathFor(dataDir, debtID, "receipt_1.pdf")); !os.IsNotExist(err) {
		t.Errorf("no debería existir receipt_1.pdf: %v", err)
	}
	if _, _, err := svc.DocumentPath(ctx, bill.ID, DebtDocumentReceipt); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("DocumentPath receipt: got %v, want ErrDocumentNotFound", err)
	}
}

func TestDebtDocumentsRejectInvalidFiles(t *testing.T) {
	cases := []struct {
		name    string
		invoice *PDFUpload
		receipt *PDFUpload
	}{
		{
			name:    "factura con extensión distinta de PDF",
			invoice: &PDFUpload{FileName: "factura.txt", Reader: bytes.NewReader([]byte(testPDF))},
		},
		{
			name:    "comprobante sin firma PDF",
			receipt: &PDFUpload{FileName: "comprobante.pdf", Reader: bytes.NewReader([]byte("no es un pdf"))},
		},
		{
			name:    "archivo vacío",
			invoice: &PDFUpload{FileName: "factura.pdf", Reader: bytes.NewReader(nil)},
		},
		{
			name: "documento mayor a 10 MB",
			invoice: &PDFUpload{
				FileName: "factura.pdf",
				Reader:   bytes.NewReader(append([]byte("%PDF-"), bytes.Repeat([]byte("a"), int(MaxDebtDocumentBytes))...)),
			},
		},
		{
			name:    "factura válida con comprobante inválido",
			invoice: &PDFUpload{FileName: "factura.pdf", Reader: bytes.NewReader([]byte(testPDF))},
			receipt: &PDFUpload{FileName: "comprobante.pdf", Reader: bytes.NewReader([]byte("no es un pdf"))},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, ctx, debtID, dataDir := newDebtReceiptTestService(t)
			bill := firstBill(t, svc, ctx, debtID)

			if _, err := svc.PayBillWithDocuments(
				ctx,
				bill.ID,
				time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
				"",
				tc.invoice,
				tc.receipt,
			); err == nil {
				t.Fatal("se esperaba error para un documento inválido")
			}

			current, err := svc.ListBills(ctx, debtID)
			if err != nil || len(current) != 1 {
				t.Fatalf("ListBills: got %d, err %v", len(current), err)
			}
			if current[0].Status != "pending" {
				t.Errorf("la cuota no debería haberse pagado, status=%q", current[0].Status)
			}
			if current[0].HasInvoice || current[0].HasReceipt {
				t.Errorf("no debería existir documento para una cuota impaga: %+v", current[0])
			}
			for _, name := range []string{"invoice_1.pdf", "receipt_1.pdf"} {
				if _, err := os.Stat(documentPathFor(dataDir, debtID, name)); !os.IsNotExist(err) {
					t.Errorf("no debería existir %s, err=%v", name, err)
				}
			}
		})
	}
}

func TestDebtDocumentSaveAfterPayment(t *testing.T) {
	svc, ctx, debtID, _ := newDebtReceiptTestService(t)
	bill := firstBill(t, svc, ctx, debtID)

	if _, err := svc.PayBill(ctx, bill.ID, time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), ""); err != nil {
		t.Fatalf("PayBill: %v", err)
	}

	invoice, err := svc.SaveDocument(ctx, bill.ID, DebtDocumentInvoice, PDFUpload{
		FileName: "factura.pdf",
		Reader:   bytes.NewReader([]byte(testPDF)),
	})
	if err != nil {
		t.Fatalf("SaveDocument invoice: %v", err)
	}
	if invoice == nil || !invoice.HasInvoice || invoice.HasReceipt {
		t.Fatalf("flags tras guardar factura: %+v", invoice)
	}

	withReceipt, err := svc.SaveDocument(ctx, bill.ID, DebtDocumentReceipt, PDFUpload{
		FileName: "comprobante.pdf",
		Reader:   bytes.NewReader([]byte(testPDF)),
	})
	if err != nil {
		t.Fatalf("SaveDocument receipt: %v", err)
	}
	if withReceipt == nil || !withReceipt.HasInvoice || !withReceipt.HasReceipt {
		t.Fatalf("flags tras guardar comprobante: %+v", withReceipt)
	}

	// Reemplazar la factura debe sobrescribir solo ese archivo.
	replaced := "%PDF-1.7\n%%EOF\n"
	if _, err := svc.SaveDocument(ctx, bill.ID, DebtDocumentInvoice, PDFUpload{
		FileName: "factura.pdf",
		Reader:   bytes.NewReader([]byte(replaced)),
	}); err != nil {
		t.Fatalf("SaveDocument (reemplazo): %v", err)
	}
	path, _, err := svc.DocumentPath(ctx, bill.ID, DebtDocumentInvoice)
	if err != nil {
		t.Fatalf("DocumentPath: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer factura reemplazada: %v", err)
	}
	if string(data) != replaced {
		t.Errorf("la factura no fue reemplazada: %q", data)
	}
}

func TestDebtDocumentSaveRequiresPaidBill(t *testing.T) {
	svc, ctx, debtID, _ := newDebtReceiptTestService(t)
	bill := firstBill(t, svc, ctx, debtID)

	if _, err := svc.SaveDocument(ctx, bill.ID, DebtDocumentInvoice, PDFUpload{
		FileName: "factura.pdf",
		Reader:   bytes.NewReader([]byte(testPDF)),
	}); err == nil {
		t.Fatal("se esperaba error al adjuntar una factura a una cuota impaga")
	}
}

func TestDebtDocumentDelete(t *testing.T) {
	svc, ctx, debtID, dataDir := newDebtReceiptTestService(t)
	bill := firstBill(t, svc, ctx, debtID)

	if _, err := svc.PayBillWithDocuments(
		ctx,
		bill.ID,
		time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
		"",
		&PDFUpload{FileName: "factura.pdf", Reader: bytes.NewReader([]byte(testPDF))},
		&PDFUpload{FileName: "comprobante.pdf", Reader: bytes.NewReader([]byte(testPDF))},
	); err != nil {
		t.Fatalf("PayBillWithDocuments: %v", err)
	}

	updated, err := svc.DeleteDocument(ctx, bill.ID, DebtDocumentInvoice)
	if err != nil {
		t.Fatalf("DeleteDocument invoice: %v", err)
	}
	if updated.HasInvoice || !updated.HasReceipt {
		t.Fatalf("flags tras remover factura: %+v", updated)
	}
	if _, err := os.Stat(documentPathFor(dataDir, debtID, "invoice_1.pdf")); !os.IsNotExist(err) {
		t.Errorf("invoice_1.pdf todavía existe: %v", err)
	}
	if _, err := os.Stat(documentPathFor(dataDir, debtID, "receipt_1.pdf")); err != nil {
		t.Errorf("receipt_1.pdf no debería haberse removido: %v", err)
	}

	updated, err = svc.DeleteDocument(ctx, bill.ID, DebtDocumentReceipt)
	if err != nil {
		t.Fatalf("DeleteDocument receipt: %v", err)
	}
	if updated.HasInvoice || updated.HasReceipt {
		t.Fatalf("flags tras remover comprobante: %+v", updated)
	}
	if _, err := svc.DeleteDocument(ctx, bill.ID, DebtDocumentInvoice); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("DeleteDocument sin archivo: got %v, want ErrDocumentNotFound", err)
	}
}

func TestDebtDocumentMissingFile(t *testing.T) {
	svc, ctx, debtID, dataDir := newDebtReceiptTestService(t)
	bill := firstBill(t, svc, ctx, debtID)

	if _, err := svc.PayBillWithDocuments(
		ctx,
		bill.ID,
		time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
		"",
		&PDFUpload{FileName: "factura.pdf", Reader: bytes.NewReader([]byte(testPDF))},
		&PDFUpload{FileName: "comprobante.pdf", Reader: bytes.NewReader([]byte(testPDF))},
	); err != nil {
		t.Fatalf("PayBillWithDocuments: %v", err)
	}

	if err := os.Remove(documentPathFor(dataDir, debtID, "receipt_1.pdf")); err != nil {
		t.Fatalf("borrar comprobante: %v", err)
	}

	reloaded, err := svc.ListBills(ctx, debtID)
	if err != nil || len(reloaded) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(reloaded), err)
	}
	if !reloaded[0].HasInvoice {
		t.Error("has_invoice debería seguir en true")
	}
	if reloaded[0].HasReceipt {
		t.Error("has_receipt debería ser false si el archivo no existe")
	}
	if _, _, err := svc.DocumentPath(ctx, bill.ID, DebtDocumentReceipt); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("DocumentPath: got %v, want ErrDocumentNotFound", err)
	}

	// Sin ningún archivo, has_documents debe ser false.
	if err := os.Remove(documentPathFor(dataDir, debtID, "invoice_1.pdf")); err != nil {
		t.Fatalf("borrar factura: %v", err)
	}
	debts, err := svc.List(ctx)
	if err != nil || len(debts) != 1 {
		t.Fatalf("List: got %d, err %v", len(debts), err)
	}
	if debts[0].HasDocuments {
		t.Error("has_documents debería ser false sin archivos")
	}
}
