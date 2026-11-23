package rental

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newTestService(t *testing.T) (*Service, func()) {
	t.Helper()
	dir := t.TempDir()
	database, err := db.OpenDB(filepath.Join(dir, "test.db"), filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatalf("abrir db: %v", err)
	}
	settings := services.NewSystemSettingsService(storage.NewSystemSettingsStorage(database))
	email := services.NewEmailService(settings)
	svc := NewService(NewStorage(database), email, settings, dir)
	return svc, func() { database.Close() }
}

func createHome(t *testing.T, svc *Service, name string) int64 {
	t.Helper()
	res, err := svc.storage.db.Exec(`INSERT INTO homes (name, address) VALUES (?, ?)`, name, "Calle Demo 123")
	if err != nil {
		t.Fatalf("crear casa: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestLeaseLifecycleInvoiceAndReceipt(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	svc.now = func() time.Time { return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) }

	homeID := createHome(t, svc, "Casa Demo")
	person, err := svc.CreatePerson(ctx, &Person{FullName: "Inquilino Demo", Email: "demo@example.com", Phone: "+1 555 0100"})
	if err != nil {
		t.Fatalf("crear persona: %v", err)
	}

	lease, err := svc.CreateLease(ctx, &Lease{
		HomeID: homeID, StartDate: "2026-12-01", TermMonths: 12,
		RentAmount: 500, MaintenanceAmount: 50, Currency: "USD",
		PaymentDay: 5, DepositAmount: 1000,
		Parties: []LeaseParty{{PersonID: person.ID, Role: RoleTenant, NotifyEmail: true}},
	})
	if err != nil {
		t.Fatalf("crear contrato: %v", err)
	}
	if lease.Status != LeaseDraft {
		t.Fatalf("estado inicial = %s", lease.Status)
	}
	if lease.EndDate != "2027-11-30" {
		t.Fatalf("fin de contrato = %s, quería 2027-11-30", lease.EndDate)
	}
	if len(lease.Parties) != 1 {
		t.Fatalf("partes = %d", len(lease.Parties))
	}

	active, err := svc.ActivateLease(ctx, lease.ID)
	if err != nil {
		t.Fatalf("activar: %v", err)
	}
	if active.Status != LeaseActive {
		t.Fatalf("estado = %s", active.Status)
	}

	charges, err := svc.ListCharges(ctx, lease.ID)
	if err != nil {
		t.Fatalf("cargos: %v", err)
	}
	// 12 rentas + 12 mantenimientos + 1 depósito.
	if len(charges) != 25 {
		t.Fatalf("cargos = %d, quería 25", len(charges))
	}
	created, err := svc.GenerateCharges(ctx, lease.ID)
	if err != nil {
		t.Fatalf("regenerar cargos: %v", err)
	}
	if created != 0 {
		t.Fatalf("generación idempotente creó %d cargos", created)
	}

	var ids []int64
	var expectedTotal float64
	for _, c := range charges {
		if c.Status == ChargePending && len(ids) < 2 {
			ids = append(ids, c.ID)
			expectedTotal += c.Amount
		}
	}
	inv, err := svc.IssueInvoice(ctx, IssueInvoiceInput{
		LeaseID: lease.ID, ChargeIDs: ids, DueDate: "2026-12-05",
	})
	if err != nil {
		t.Fatalf("emitir factura: %v", err)
	}
	if inv.Number != 1 || inv.NumberLabel != "ALQ-2026-0001" {
		t.Fatalf("numeración = %s (%d)", inv.NumberLabel, inv.Number)
	}
	if inv.Total != expectedTotal {
		t.Fatalf("total = %.2f, quería %.2f", inv.Total, expectedTotal)
	}
	path, err := svc.EnsureInvoicePDF(ctx, inv.ID)
	if err != nil {
		t.Fatalf("generar PDF: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer PDF: %v", err)
	}
	if len(data) < 5 || string(data[:5]) != "%PDF-" {
		t.Fatalf("PDF inválido")
	}

	payment, err := svc.CreatePayment(ctx, CreatePaymentInput{
		LeaseID: lease.ID, InvoiceID: &inv.ID, Amount: expectedTotal, Currency: "USD",
		Method: "transfer", PaidAt: "2026-12-05", WithReceipt: true,
	})
	if err != nil {
		t.Fatalf("registrar pago: %v", err)
	}
	if payment.ReceiptID == nil {
		t.Fatal("no se emitió recibo")
	}
	receipt, err := svc.storage.GetReceipt(ctx, *payment.ReceiptID)
	if err != nil || receipt == nil {
		t.Fatalf("recibo: %v", err)
	}
	if receipt.Number != 1 {
		t.Fatalf("número de recibo = %d", receipt.Number)
	}
	receiptData, _, err := svc.ReceiptPDF(ctx, receipt.ID)
	if err != nil || len(receiptData) < 5 || string(receiptData[:5]) != "%PDF-" {
		t.Fatalf("PDF de recibo inválido: %v", err)
	}

	chargeRows, _ := svc.ListCharges(ctx, lease.ID)
	for _, c := range chargeRows {
		if c.ID == ids[0] && c.Status != ChargePaid {
			t.Fatalf("cargo %d quedó en %s", c.ID, c.Status)
		}
	}

	bal, err := svc.Balance(ctx, lease.ID)
	if err != nil {
		t.Fatalf("saldo: %v", err)
	}
	if bal.Paid != expectedTotal || bal.Balance != bal.Charged-expectedTotal {
		t.Fatalf("saldo inesperado: %+v", bal)
	}
}

func TestReservationBlocksHomeAndExpires(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	homeID := createHome(t, svc, "Casa Reserva")
	person, err := svc.CreatePerson(ctx, &Person{FullName: "Solicitante Demo", Email: "demo@example.com"})
	if err != nil {
		t.Fatalf("persona: %v", err)
	}

	res, err := svc.CreateReservation(ctx, &Reservation{
		HomeID: homeID, ApplicantPersonID: person.ID, Amount: 300, Currency: "USD",
		Reference: "DEMO-REF-001", AppliedToDeposit: true,
		StartsAt: now.Format(time.RFC3339), ExpiresAt: now.AddDate(0, 0, 15).Format(time.RFC3339),
		RefundPolicy: RefundForfeitable, OwnerSplitPercent: 50, AgencySplitPercent: 50,
	})
	if err != nil {
		t.Fatalf("crear apartado: %v", err)
	}
	if res.Status != ReservationActive {
		t.Fatalf("estado = %s", res.Status)
	}

	status, err := svc.GetHomeStatus(ctx, homeID)
	if err != nil {
		t.Fatalf("estado de casa: %v", err)
	}
	if status.ListingStatus != ListingReserved {
		t.Fatalf("listing = %s", status.ListingStatus)
	}

	if _, err := svc.CreateReservation(ctx, &Reservation{
		HomeID: homeID, ApplicantPersonID: person.ID, StartsAt: now.Format(time.RFC3339),
		ExpiresAt: now.AddDate(0, 0, 5).Format(time.RFC3339), Currency: "USD",
		RefundPolicy: RefundForfeitable, OwnerSplitPercent: 50, AgencySplitPercent: 50,
	}); err == nil {
		t.Fatal("se permitió un segundo apartado activo")
	}

	pdf, name, err := svc.ReservationPDF(ctx, res.ID)
	if err != nil || len(pdf) < 5 || string(pdf[:5]) != "%PDF-" {
		t.Fatalf("PDF de reserva inválido (%s): %v", name, err)
	}

	now = now.AddDate(0, 0, 16)
	svc.now = func() time.Time { return now }
	n, err := svc.ExpireReservations(ctx)
	if err != nil || n != 1 {
		t.Fatalf("expirar: n=%d err=%v", n, err)
	}
	status, _ = svc.GetHomeStatus(ctx, homeID)
	if status.ListingStatus != ListingAvailable {
		t.Fatalf("listing tras vencer = %s", status.ListingStatus)
	}
}

func TestChargesMatchTermFromMidMonth(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	svc.now = func() time.Time { return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) }
	homeID := createHome(t, svc, "Casa Mitad de Mes")
	person, _ := svc.CreatePerson(ctx, &Person{FullName: "Inquilino Demo", Email: "demo@example.com"})
	lease, err := svc.CreateLease(ctx, &Lease{
		HomeID: homeID, StartDate: "2026-11-23", TermMonths: 12,
		RentAmount: 500, MaintenanceAmount: 50, Currency: "USD", PaymentDay: 5, DepositAmount: 500,
		Parties: []LeaseParty{{PersonID: person.ID, Role: RoleTenant, NotifyEmail: true}},
	})
	if err != nil {
		t.Fatalf("crear contrato: %v", err)
	}
	if _, err := svc.ActivateLease(ctx, lease.ID); err != nil {
		t.Fatalf("activar: %v", err)
	}
	charges, _ := svc.ListCharges(ctx, lease.ID)
	if len(charges) != 25 {
		t.Fatalf("cargos = %d, quería 25 (12 rentas + 12 mantenimientos + depósito)", len(charges))
	}
}

func TestLeaseValidationAndImmutability(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	homeID := createHome(t, svc, "Casa Validación")

	if _, err := svc.CreateLease(ctx, &Lease{
		HomeID: homeID, StartDate: "2026-12-01", TermMonths: 0,
		RentAmount: 100, Currency: "USD", PaymentDay: 1,
	}); err == nil {
		t.Fatal("se aceptó un plazo de 0 meses")
	}
	if _, err := svc.CreateLease(ctx, &Lease{
		HomeID: homeID, StartDate: "2026-12-01", TermMonths: 6,
		RentAmount: 0, Currency: "USD", PaymentDay: 1,
	}); err == nil {
		t.Fatal("se aceptó un contrato sin renta ni mantenimiento")
	}
}

func TestVoidInvoiceKeepsNumber(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	svc.now = func() time.Time { return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) }
	homeID := createHome(t, svc, "Casa Anulación")
	person, _ := svc.CreatePerson(ctx, &Person{FullName: "Inquilino Demo", Email: "demo@example.com"})
	lease, err := svc.CreateLease(ctx, &Lease{
		HomeID: homeID, StartDate: "2026-12-01", TermMonths: 6,
		RentAmount: 100, Currency: "USD", PaymentDay: 1,
		Parties: []LeaseParty{{PersonID: person.ID, Role: RoleTenant, NotifyEmail: true}},
	})
	if err != nil {
		t.Fatalf("crear contrato: %v", err)
	}
	if _, err := svc.ActivateLease(ctx, lease.ID); err != nil {
		t.Fatalf("activar: %v", err)
	}
	charges, _ := svc.ListCharges(ctx, lease.ID)
	inv, err := svc.IssueInvoice(ctx, IssueInvoiceInput{LeaseID: lease.ID, ChargeIDs: []int64{charges[0].ID}, DueDate: "2026-12-05"})
	if err != nil {
		t.Fatalf("factura: %v", err)
	}
	voided, err := svc.VoidInvoice(ctx, inv.ID, "Error de ejemplo")
	if err != nil {
		t.Fatalf("anular: %v", err)
	}
	if voided.Number != inv.Number || voided.Status != "void" {
		t.Fatalf("anulación alteró número o estado: %+v", voided)
	}
	second, err := svc.IssueInvoice(ctx, IssueInvoiceInput{LeaseID: lease.ID, ChargeIDs: []int64{charges[0].ID}, DueDate: "2026-12-05"})
	if err != nil {
		t.Fatalf("segunda factura: %v", err)
	}
	if second.Number != inv.Number+1 {
		t.Fatalf("numeración tras anular = %d, quería %d", second.Number, inv.Number+1)
	}
}

func TestAgencyMustHaveRealEstateCategory(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	homeID := createHome(t, svc, "Casa Inmobiliaria")

	var insuranceID, realEstateID int64
	if err := svc.storage.db.QueryRow(`SELECT id FROM institution_categories WHERE key = 'insurance'`).Scan(&insuranceID); err != nil {
		t.Fatalf("categoría insurance: %v", err)
	}
	if err := svc.storage.db.QueryRow(`SELECT id FROM institution_categories WHERE key = 'real_estate'`).Scan(&realEstateID); err != nil {
		t.Fatalf("categoría real_estate: %v", err)
	}
	insertAgency := func(name string, categoryID int64) int64 {
		res, err := svc.storage.db.Exec(`INSERT INTO institutions (name, category_id) VALUES (?, ?)`, name, categoryID)
		if err != nil {
			t.Fatalf("crear institución: %v", err)
		}
		id, _ := res.LastInsertId()
		return id
	}
	insurerID := insertAgency("Aseguradora Demo", insuranceID)
	if _, err := svc.CreateLease(ctx, &Lease{
		HomeID: homeID, StartDate: "2026-12-01", TermMonths: 12,
		RentAmount: 500, Currency: "USD", PaymentDay: 1,
		AgencyInstitutionID: &insurerID,
	}); err == nil {
		t.Fatal("se aceptó una aseguradora como inmobiliaria")
	}
	agencyID := insertAgency("Inmobiliaria Demo", realEstateID)
	if _, err := svc.CreateLease(ctx, &Lease{
		HomeID: homeID, StartDate: "2026-12-01", TermMonths: 12,
		RentAmount: 500, Currency: "USD", PaymentDay: 1,
		AgencyInstitutionID: &agencyID,
	}); err != nil {
		t.Fatalf("no se aceptó una inmobiliaria válida: %v", err)
	}
}

func TestCommissionPercentOrFixed(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	homeID := createHome(t, svc, "Casa Comisión")

	fixed, err := svc.CreateLease(ctx, &Lease{
		HomeID: homeID, StartDate: "2026-12-01", TermMonths: 12,
		RentAmount: 500, Currency: "USD", PaymentDay: 1,
		CommissionType: "fixed", CommissionAmount: 500,
	})
	if err != nil {
		t.Fatalf("contrato con comisión fija: %v", err)
	}
	if fixed.CommissionType != "fixed" || fixed.CommissionAmount != 500 || fixed.CommissionPercent != 0 {
		t.Fatalf("comisión fija persistida mal: %+v", fixed)
	}

	percent, err := svc.CreateLease(ctx, &Lease{
		HomeID: homeID, StartDate: "2026-12-01", TermMonths: 6,
		RentAmount: 400, Currency: "USD", PaymentDay: 1,
		CommissionType: "percent", CommissionPercent: 8.5, CommissionAmount: 999,
	})
	if err != nil {
		t.Fatalf("contrato con comisión porcentual: %v", err)
	}
	if percent.CommissionType != "percent" || percent.CommissionPercent != 8.5 || percent.CommissionAmount != 0 {
		t.Fatalf("comisión porcentual persistida mal: %+v", percent)
	}

	if _, err := svc.CreateLease(ctx, &Lease{
		HomeID: homeID, StartDate: "2026-12-01", TermMonths: 6,
		RentAmount: 400, Currency: "USD", PaymentDay: 1, CommissionType: "other",
	}); err == nil {
		t.Fatal("se aceptó un tipo de comisión inválido")
	}
}

func TestAutoInvoiceDuePeriodAndPaymentMarksPaid(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	svc.now = func() time.Time { return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) }

	homeID := createHome(t, svc, "Casa Automática")
	person, _ := svc.CreatePerson(ctx, &Person{FullName: "Inquilino Demo", Email: "demo@example.com"})
	lease, err := svc.CreateLease(ctx, &Lease{
		HomeID: homeID, StartDate: "2026-11-01", TermMonths: 12,
		RentAmount: 500, MaintenanceAmount: 50, Currency: "USD",
		PaymentDay: 1, DepositAmount: 1000,
		Parties: []LeaseParty{{PersonID: person.ID, Role: RoleTenant, NotifyEmail: true}},
	})
	if err != nil {
		t.Fatalf("crear contrato: %v", err)
	}
	if _, err := svc.ActivateLease(ctx, lease.ID); err != nil {
		t.Fatalf("activar: %v", err)
	}

	created, err := svc.AutoInvoiceDuePeriods(ctx)
	if err != nil {
		t.Fatalf("facturación automática: %v", err)
	}
	if created != 1 {
		t.Fatalf("facturas creadas = %d, quería 1 (solo el período de noviembre)", created)
	}
	again, err := svc.AutoInvoiceDuePeriods(ctx)
	if err != nil || again != 0 {
		t.Fatalf("segunda pasada creó %d facturas (err=%v)", again, err)
	}
	invoices, _ := svc.ListInvoices(ctx, lease.ID, "")
	if len(invoices) != 1 {
		t.Fatalf("facturas = %d", len(invoices))
	}
	if invoices[0].Total != 1550 {
		t.Fatalf("total = %.2f, quería 1550 (renta + mantenimiento + depósito)", invoices[0].Total)
	}

	payment, err := svc.CreatePayment(ctx, CreatePaymentInput{
		LeaseID: lease.ID, Amount: 1550, Currency: "USD",
		Method: "transfer", PaidAt: "2026-11-15",
	})
	if err != nil {
		t.Fatalf("pago: %v", err)
	}
	if payment.InvoiceID == nil || *payment.InvoiceID != invoices[0].ID {
		t.Fatalf("el pago no se imputó a la factura más antigua: %+v", payment)
	}
	charges, _ := svc.ListCharges(ctx, lease.ID)
	paid := 0
	for _, c := range charges {
		if c.Status == ChargePaid {
			paid++
		}
	}
	if paid != 3 {
		t.Fatalf("cargos pagados = %d, quería 3 (renta, mantenimiento y depósito)", paid)
	}
}

func TestPaymentCreatesInvoiceForDueChargesWhenNoneExists(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	svc.now = func() time.Time { return time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC) }
	homeID := createHome(t, svc, "Casa Pago Directo")
	person, _ := svc.CreatePerson(ctx, &Person{FullName: "Inquilino Demo", Email: "demo@example.com"})
	lease, err := svc.CreateLease(ctx, &Lease{
		HomeID: homeID, StartDate: "2026-11-01", TermMonths: 6,
		RentAmount: 500, Currency: "USD", PaymentDay: 1,
		Parties: []LeaseParty{{PersonID: person.ID, Role: RoleTenant, NotifyEmail: true}},
	})
	if err != nil {
		t.Fatalf("crear contrato: %v", err)
	}
	if _, err := svc.ActivateLease(ctx, lease.ID); err != nil {
		t.Fatalf("activar: %v", err)
	}
	payment, err := svc.CreatePayment(ctx, CreatePaymentInput{
		LeaseID: lease.ID, Amount: 500, Currency: "USD", Method: "cash", PaidAt: "2026-11-15",
	})
	if err != nil {
		t.Fatalf("pago: %v", err)
	}
	if payment.InvoiceID == nil {
		t.Fatal("el pago no generó ni imputó una factura")
	}
	invoices, _ := svc.ListInvoices(ctx, lease.ID, "")
	if len(invoices) != 1 || invoices[0].Total != 500 {
		t.Fatalf("factura inesperada: %+v", invoices)
	}
}
