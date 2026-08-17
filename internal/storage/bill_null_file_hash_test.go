package storage

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// newBillNullFileHashTestDB crea una DB de prueba con bills cuyo file_hash es
// NULL, vacío o un hash real (escenario de la SPEC-042: registros legacy en iHost).
func newBillNullFileHashTestDB(t *testing.T) *BillStorage {
	t.Helper()
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

	mustExec("INSERT INTO homes (name) VALUES ('Casa Central')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Internet', 'Claro', 1, 'monthly', 100, 1, 'internet', 'fixed', 1)`)

	// file_hash NULL (manual, legacy), vacío y con hash real (importada por PDF).
	mustExec("INSERT INTO bills (service_id, year, month, amount, status) VALUES (1, 2026, 8, 1500, 'pending')")
	mustExec("INSERT INTO bills (service_id, year, month, amount, status, file_hash) VALUES (1, 2026, 7, 1490, 'paid', '')")
	mustExec("INSERT INTO bills (service_id, year, month, amount, status, file_hash) VALUES (1, 2026, 6, 1480, 'pending', 'd41d8cd98f00b204e9800998ecf8427e')")

	return NewBillStorage(database)
}

func TestListByServiceHandlesNullFileHash(t *testing.T) {
	store := newBillNullFileHashTestDB(t)
	ctx := context.Background()

	bills, err := store.ListByService(ctx, 1)
	if err != nil {
		t.Fatalf("ListByService con file_hash NULL: %v", err)
	}
	if len(bills) != 3 {
		t.Fatalf("se esperaban 3 facturas, got %d", len(bills))
	}

	byMonth := map[int]models.Bill{}
	for _, b := range bills {
		byMonth[b.Month] = b
	}

	if b := byMonth[8]; b.FileHash != "" {
		t.Errorf("file_hash NULL debería escanearse como vacío, got %q", b.FileHash)
	}
	if b := byMonth[8]; b.InvoiceNumber != "" {
		t.Errorf("invoice_number NULL debería escanearse como vacío, got %q", b.InvoiceNumber)
	}
	if b := byMonth[8]; b.DriveURL != "" {
		t.Errorf("drive_url NULL debería escanearse como vacío, got %q", b.DriveURL)
	}
	if b := byMonth[7]; b.FileHash != "" {
		t.Errorf("file_hash vacío debería escanearse como vacío, got %q", b.FileHash)
	}
	if b := byMonth[6]; b.FileHash != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Errorf("file_hash con hash real debería escanearse correctamente, got %q", b.FileHash)
	}
}

func TestGetByIDHandlesNullFileHash(t *testing.T) {
	store := newBillNullFileHashTestDB(t)
	ctx := context.Background()

	bill, err := store.GetByID(ctx, 1)
	if err != nil {
		t.Fatalf("GetByID con file_hash NULL: %v", err)
	}
	if bill == nil {
		t.Fatalf("GetByID no devolvió la factura")
	}
	if bill.FileHash != "" {
		t.Errorf("file_hash NULL debería escanearse como vacío, got %q", bill.FileHash)
	}
	if bill.InvoiceNumber != "" {
		t.Errorf("invoice_number NULL debería escanearse como vacío, got %q", bill.InvoiceNumber)
	}
	if bill.DriveURL != "" {
		t.Errorf("drive_url NULL debería escanearse como vacío, got %q", bill.DriveURL)
	}
}
