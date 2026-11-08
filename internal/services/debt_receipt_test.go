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

func TestDebtReceiptPayStoresPDF(t *testing.T) {
	svc, ctx, debtID, dataDir := newDebtReceiptTestService(t)

	bills, err := svc.ListBills(ctx, debtID)
	if err != nil || len(bills) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(bills), err)
	}
	if bills[0].HasReceipt {
		t.Fatal("una cuota nueva no debería tener comprobante")
	}

	paid, err := svc.PayBillWithReceipt(
		ctx,
		bills[0].ID,
		time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
		"demo-reference-001",
		&ReceiptUpload{FileName: "comprobante.pdf", Reader: bytes.NewReader([]byte(testPDF))},
	)
	if err != nil {
		t.Fatalf("PayBillWithReceipt: %v", err)
	}
	if paid.Status != "paid" || !paid.HasReceipt {
		t.Fatalf("cuota pagada con comprobante: %+v", paid)
	}

	path := filepath.Join(dataDir, "documents", fmt.Sprintf("%d", debtID), "payment_1.pdf")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer comprobante: %v", err)
	}
	if !bytes.Equal(data, []byte(testPDF)) {
		t.Errorf("contenido del comprobante inesperado: %q", data)
	}

	gotPath, gotName, err := svc.ReceiptPath(ctx, bills[0].ID)
	if err != nil {
		t.Fatalf("ReceiptPath: %v", err)
	}
	if gotPath != path || gotName != "payment_1.pdf" {
		t.Errorf("ReceiptPath = %q, %q", gotPath, gotName)
	}

	// El listado debe calcular has_receipt desde el disco.
	reloaded, err := svc.ListBills(ctx, debtID)
	if err != nil || len(reloaded) != 1 {
		t.Fatalf("ListBills recargado: got %d, err %v", len(reloaded), err)
	}
	if !reloaded[0].HasReceipt {
		t.Error("ListBills debería devolver has_receipt=true")
	}

	debts, err := svc.List(ctx)
	if err != nil || len(debts) != 1 {
		t.Fatalf("List: got %d, err %v", len(debts), err)
	}
	if !debts[0].HasReceipts {
		t.Error("List debería devolver has_receipts=true")
	}
	detail, err := svc.GetByID(ctx, debtID)
	if err != nil || detail == nil {
		t.Fatalf("GetByID: %v, %v", detail, err)
	}
	if !detail.HasReceipts {
		t.Error("GetByID debería devolver has_receipts=true")
	}
}

func TestDebtReceiptRejectsInvalidFiles(t *testing.T) {
	svc, ctx, debtID, dataDir := newDebtReceiptTestService(t)

	bills, err := svc.ListBills(ctx, debtID)
	if err != nil || len(bills) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(bills), err)
	}

	cases := []struct {
		name string
		file ReceiptUpload
	}{
		{
			name: "extensión distinta de PDF",
			file: ReceiptUpload{FileName: "comprobante.txt", Reader: bytes.NewReader([]byte(testPDF))},
		},
		{
			name: "contenido sin firma PDF",
			file: ReceiptUpload{FileName: "comprobante.pdf", Reader: bytes.NewReader([]byte("no es un pdf"))},
		},
		{
			name: "archivo vacío",
			file: ReceiptUpload{FileName: "comprobante.pdf", Reader: bytes.NewReader(nil)},
		},
		{
			name: "archivo mayor a 10 MB",
			file: ReceiptUpload{
				FileName: "comprobante.pdf",
				Reader:   bytes.NewReader(append([]byte("%PDF-"), bytes.Repeat([]byte("a"), int(MaxDebtReceiptBytes))...)),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.PayBillWithReceipt(
				ctx,
				bills[0].ID,
				time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
				"",
				&tc.file,
			); err == nil {
				t.Fatal("se esperaba error para un comprobante inválido")
			}

			current, err := svc.ListBills(ctx, debtID)
			if err != nil || len(current) != 1 {
				t.Fatalf("ListBills: got %d, err %v", len(current), err)
			}
			if current[0].Status != "pending" {
				t.Errorf("la cuota no debería haberse pagado, status=%q", current[0].Status)
			}
			if current[0].HasReceipt {
				t.Error("no debería existir comprobante para una cuota impaga")
			}
		})
	}

	path := filepath.Join(dataDir, "documents", fmt.Sprintf("%d", debtID), "payment_1.pdf")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("no debería existir el archivo final, err=%v", err)
	}
}

func TestDebtReceiptSaveAfterPayment(t *testing.T) {
	svc, ctx, debtID, _ := newDebtReceiptTestService(t)

	bills, err := svc.ListBills(ctx, debtID)
	if err != nil || len(bills) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(bills), err)
	}

	if _, err := svc.PayBill(ctx, bills[0].ID, time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), ""); err != nil {
		t.Fatalf("PayBill: %v", err)
	}

	updated, err := svc.SaveReceipt(ctx, bills[0].ID, ReceiptUpload{
		FileName: "comprobante.pdf",
		Reader:   bytes.NewReader([]byte(testPDF)),
	})
	if err != nil {
		t.Fatalf("SaveReceipt: %v", err)
	}
	if updated == nil || !updated.HasReceipt {
		t.Fatalf("SaveReceipt no devolvió has_receipt: %+v", updated)
	}

	// Reemplazar el comprobante debe sobrescribir el archivo.
	replaced := "%PDF-1.7\n%%EOF\n"
	if _, err := svc.SaveReceipt(ctx, bills[0].ID, ReceiptUpload{
		FileName: "comprobante.pdf",
		Reader:   bytes.NewReader([]byte(replaced)),
	}); err != nil {
		t.Fatalf("SaveReceipt (reemplazo): %v", err)
	}
	path, _, err := svc.ReceiptPath(ctx, bills[0].ID)
	if err != nil {
		t.Fatalf("ReceiptPath: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer comprobante reemplazado: %v", err)
	}
	if string(data) != replaced {
		t.Errorf("el comprobante no fue reemplazado: %q", data)
	}
}

func TestDebtReceiptSaveRequiresPaidBill(t *testing.T) {
	svc, ctx, debtID, _ := newDebtReceiptTestService(t)

	bills, err := svc.ListBills(ctx, debtID)
	if err != nil || len(bills) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(bills), err)
	}

	if _, err := svc.SaveReceipt(ctx, bills[0].ID, ReceiptUpload{
		FileName: "comprobante.pdf",
		Reader:   bytes.NewReader([]byte(testPDF)),
	}); err == nil {
		t.Fatal("se esperaba error al adjuntar comprobante a una cuota impaga")
	}
}

func TestDebtReceiptMissingFile(t *testing.T) {
	svc, ctx, debtID, dataDir := newDebtReceiptTestService(t)

	bills, err := svc.ListBills(ctx, debtID)
	if err != nil || len(bills) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(bills), err)
	}
	if _, err := svc.PayBillWithReceipt(
		ctx,
		bills[0].ID,
		time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
		"",
		&ReceiptUpload{FileName: "comprobante.pdf", Reader: bytes.NewReader([]byte(testPDF))},
	); err != nil {
		t.Fatalf("PayBillWithReceipt: %v", err)
	}

	path := filepath.Join(dataDir, "documents", fmt.Sprintf("%d", debtID), "payment_1.pdf")
	if err := os.Remove(path); err != nil {
		t.Fatalf("borrar comprobante: %v", err)
	}

	reloaded, err := svc.ListBills(ctx, debtID)
	if err != nil || len(reloaded) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(reloaded), err)
	}
	if reloaded[0].HasReceipt {
		t.Error("has_receipt debería ser false si el archivo no existe")
	}
	if _, _, err := svc.ReceiptPath(ctx, bills[0].ID); !errors.Is(err, ErrReceiptNotFound) {
		t.Errorf("ReceiptPath: got %v, want ErrReceiptNotFound", err)
	}
}
