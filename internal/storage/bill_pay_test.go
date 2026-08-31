package storage

import (
	"context"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
)

func TestBillStoragePay(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("insertar dato de prueba (%q): %v", query, err)
		}
	}

	mustExec("INSERT INTO homes (name) VALUES ('Casa')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Internet', 'Claro', 1, 'monthly', 100, 1, 'internet', 'fixed', 1)`)
	mustExec("INSERT INTO bills (service_id, year, month, amount, status) VALUES (1, 2026, 8, 1500, 'pending')")

	store := NewBillStorage(database)
	paidAt := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)

	bill, err := store.Pay(ctx, 1, paidAt, "https://drive.google.com/file/d/abc/view", "TRX-001")
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if bill.Status != "paid" {
		t.Errorf("estado esperado paid, got %s", bill.Status)
	}
	if bill.PaidAt == nil || !bill.PaidAt.Equal(paidAt) {
		t.Errorf("paid_at esperado %v, got %v", paidAt, bill.PaidAt)
	}
	if bill.DriveURL != "https://drive.google.com/file/d/abc/view" {
		t.Errorf("drive_url esperado, got %q", bill.DriveURL)
	}
	if bill.PaymentReference != "TRX-001" {
		t.Errorf("payment_reference esperado TRX-001, got %q", bill.PaymentReference)
	}

	pending, err := store.ListPendingWithDetails(ctx)
	if err != nil {
		t.Fatalf("ListPendingWithDetails: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("la factura pagada no debe aparecer en pendientes, got %d", len(pending))
	}
}

func TestBillStoragePayKeepsExistingFields(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("insertar dato de prueba (%q): %v", query, err)
		}
	}

	mustExec("INSERT INTO homes (name) VALUES ('Casa')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Internet', 'Claro', 1, 'monthly', 100, 1, 'internet', 'fixed', 1)`)
	mustExec("INSERT INTO bills (service_id, year, month, amount, status, invoice_number, file_hash) VALUES (1, 2026, 8, 1500, 'pending', 'INV-123', 'abc')")

	store := NewBillStorage(database)
	paidAt := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)

	bill, err := store.Pay(ctx, 1, paidAt, "", "")
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}
	if bill.InvoiceNumber != "INV-123" {
		t.Errorf("invoice_number se perdió, got %q", bill.InvoiceNumber)
	}
	if bill.FileHash != "abc" {
		t.Errorf("file_hash se perdió, got %q", bill.FileHash)
	}
}
