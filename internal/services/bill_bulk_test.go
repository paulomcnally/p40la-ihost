package services

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newBillBulkTestEnv(t *testing.T) (*BillService, *sql.DB, func(string, ...any)) {
	t.Helper()
	database, err := db.OpenDB(filepath.Join(t.TempDir(), "test.db"), "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("ejecutar SQL de prueba (%q): %v", query, err)
		}
	}

	mustExec("INSERT INTO homes (name) VALUES ('Casa Demo')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Servicio Demo Mensual', 'Proveedor Demo', 1, 'monthly', 100, 1, 'internet', 'fixed', 1)`)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Servicio Demo Sin Monto', 'Proveedor Demo', 1, 'monthly', 0, 1, 'water', 'fixed', 1)`)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Servicio Demo Inactivo', 'Proveedor Demo', 1, 'monthly', 500, 0, 'other', 'fixed', 1)`)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Servicio Demo Anual', 'Proveedor Demo', 1, 'yearly', 300, 1, 'insurance_car', 'fixed', 1)`)

	billStorage := storage.NewBillStorage(database)
	service := NewBillService(billStorage, storage.NewServiceStorage(database))
	service.SetBillHistoryStorage(storage.NewBillHistoryStorage(database))
	service.SetCurrencyStorage(storage.NewCurrencyStorage(database))
	return service, database, mustExec
}

func fixedBulkNow() time.Time {
	return time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
}

func requireBulkError(t *testing.T, err error, code string) {
	t.Helper()
	var bulkErr *BulkError
	if !errors.As(err, &bulkErr) {
		t.Fatalf("quiero BulkError %q, obtuve %v", code, err)
	}
	if bulkErr.Code != code {
		t.Fatalf("código = %q, quiero %q", bulkErr.Code, code)
	}
}

func countBills(t *testing.T, database *sql.DB, serviceID int64) int {
	t.Helper()
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM bills WHERE service_id = ? AND deleted_at IS NULL`, serviceID).Scan(&count); err != nil {
		t.Fatalf("contar facturas: %v", err)
	}
	return count
}

func TestGenerateForServiceRangeAndIdempotency(t *testing.T) {
	service, database, _ := newBillBulkTestEnv(t)
	ctx := context.Background()

	result, err := service.GenerateForService(ctx, 1, GenerateBillsOptions{
		From:   "2026-01",
		To:     "2026-11",
		Source: models.BillSourceMCP,
		Now:    fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("generar facturas: %v", err)
	}
	if result.Created != 11 || result.Skipped != 0 || result.Failed != 0 {
		msg := ""
		if len(result.Items) > 0 {
			msg = result.Items[0].Message
		}
		t.Fatalf("resumen = created:%d skipped:%d failed:%d, quiero 11/0/0 (%s)", result.Created, result.Skipped, result.Failed, msg)
	}
	if len(result.Items) != 11 {
		t.Fatalf("items = %d, quiero 11", len(result.Items))
	}
	if result.Items[0].Period != "2026-01" || result.Items[10].Period != "2026-11" {
		t.Fatalf("períodos = %s..%s, quiero 2026-01..2026-11", result.Items[0].Period, result.Items[10].Period)
	}
	if result.Amount != 100 {
		t.Fatalf("monto = %v, quiero 100 (suggested_amount)", result.Amount)
	}
	if result.CurrencyCode != "NIO" {
		t.Fatalf("moneda = %q, quiero NIO", result.CurrencyCode)
	}
	if got := countBills(t, database, 1); got != 11 {
		t.Fatalf("facturas en DB = %d, quiero 11", got)
	}

	var source string
	if err := database.QueryRow(`SELECT source FROM bill_history WHERE bill_id = ? AND action = ?`,
		result.Items[0].BillID, models.BillActionCreated).Scan(&source); err != nil {
		t.Fatalf("leer historial: %v", err)
	}
	if source != models.BillSourceMCP {
		t.Fatalf("origen en historial = %q, quiero %q", source, models.BillSourceMCP)
	}

	again, err := service.GenerateForService(ctx, 1, GenerateBillsOptions{
		From: "2026-01", To: "2026-11", Now: fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("regenerar facturas: %v", err)
	}
	if again.Created != 0 || again.Skipped != 11 {
		t.Fatalf("segunda pasada = created:%d skipped:%d, quiero 0/11", again.Created, again.Skipped)
	}
	for _, item := range again.Items {
		if item.Reason != bulkReasonExistsPending {
			t.Fatalf("motivo = %q, quiero %q", item.Reason, bulkReasonExistsPending)
		}
	}
}

func TestGenerateForServiceDryRunDoesNotWrite(t *testing.T) {
	service, database, _ := newBillBulkTestEnv(t)
	ctx := context.Background()

	result, err := service.GenerateForService(ctx, 1, GenerateBillsOptions{
		From: "2026-01", To: "2026-03", DryRun: true, Now: fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if result.Created != 0 || len(result.Items) != 3 {
		t.Fatalf("dry run = created:%d items:%d, quiero 0/3", result.Created, len(result.Items))
	}
	for _, item := range result.Items {
		if item.Status != bulkStatusPlanned {
			t.Fatalf("estado = %q, quiero %q", item.Status, bulkStatusPlanned)
		}
	}
	if got := countBills(t, database, 1); got != 0 {
		t.Fatalf("facturas en DB = %d, quiero 0", got)
	}
}

func TestGenerateForServiceSoftDeletedSlot(t *testing.T) {
	service, database, mustExec := newBillBulkTestEnv(t)
	ctx := context.Background()
	mustExec(`INSERT INTO bills (service_id, year, month, amount, status, deleted_at)
		VALUES (1, 2026, 2, 100, 'pending', CURRENT_TIMESTAMP)`)

	result, err := service.GenerateForService(ctx, 1, GenerateBillsOptions{
		From: "2026-01", To: "2026-03", Now: fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("generar con slot borrado: %v", err)
	}
	if result.Created != 2 || result.Skipped != 1 {
		t.Fatalf("resumen = created:%d skipped:%d, quiero 2/1", result.Created, result.Skipped)
	}
	february := result.Items[1]
	if february.Reason != bulkReasonDeletedSlot {
		t.Fatalf("motivo febrero = %q, quiero %q", february.Reason, bulkReasonDeletedSlot)
	}
	if february.BillID == 0 {
		t.Fatal("quiero el bill_id del slot borrado")
	}
	if got := countBills(t, database, 1); got != 2 {
		t.Fatalf("facturas activas = %d, quiero 2", got)
	}
}

func TestGenerateForServiceYearly(t *testing.T) {
	service, _, _ := newBillBulkTestEnv(t)
	ctx := context.Background()

	result, err := service.GenerateForService(ctx, 4, GenerateBillsOptions{
		From: "2026-01", To: "2026-11", Now: fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("generar anual: %v", err)
	}
	if result.Created != 1 || len(result.Items) != 1 {
		t.Fatalf("resumen = created:%d items:%d, quiero 1/1", result.Created, len(result.Items))
	}
	if result.Items[0].Month != 0 || result.Items[0].Period != "2026-01" {
		t.Fatalf("item anual = month:%d period:%q, quiero 0/2026-01", result.Items[0].Month, result.Items[0].Period)
	}

	empty, err := service.GenerateForService(ctx, 4, GenerateBillsOptions{
		From: "2026-02", To: "2026-11", Now: fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("rango sin enero: %v", err)
	}
	if empty.Created != 0 || len(empty.Items) != 0 {
		t.Fatalf("rango sin enero = created:%d items:%d, quiero 0/0", empty.Created, len(empty.Items))
	}
}

func TestGenerateForServiceValidation(t *testing.T) {
	service, _, mustExec := newBillBulkTestEnv(t)
	ctx := context.Background()

	cases := []struct {
		name    string
		service int64
		opts    GenerateBillsOptions
		code    string
	}{
		{
			name: "rango futuro", service: 1,
			opts: GenerateBillsOptions{From: "2026-11", To: "2026-12", Now: fixedBulkNow()},
			code: BulkErrorInvalidRange,
		},
		{
			name: "rango invertido", service: 1,
			opts: GenerateBillsOptions{From: "2026-06", To: "2026-01", Now: fixedBulkNow()},
			code: BulkErrorInvalidRange,
		},
		{
			name: "más de 36 meses", service: 1,
			opts: GenerateBillsOptions{From: "2023-01", To: "2026-11", Now: fixedBulkNow()},
			code: BulkErrorInvalidRange,
		},
		{
			name: "servicio inexistente", service: 999,
			opts: GenerateBillsOptions{From: "2026-01", To: "2026-01", Now: fixedBulkNow()},
			code: BulkErrorServiceNotFound,
		},
		{
			name: "servicio inactivo", service: 3,
			opts: GenerateBillsOptions{From: "2026-01", To: "2026-01", Now: fixedBulkNow()},
			code: BulkErrorServiceInactive,
		},
		{
			name: "monto no resuelto", service: 2,
			opts: GenerateBillsOptions{From: "2026-01", To: "2026-01", Now: fixedBulkNow()},
			code: BulkErrorAmountUnresolved,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.GenerateForService(ctx, tc.service, tc.opts)
			requireBulkError(t, err, tc.code)
		})
	}

	result, err := service.GenerateForService(ctx, 3, GenerateBillsOptions{
		From: "2026-01", To: "2026-01", AllowInactive: true, DryRun: true, Now: fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("servicio inactivo con allow_inactive: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("items = %d, quiero 1", len(result.Items))
	}

	amount := 0.0
	if _, err := service.GenerateForService(ctx, 2, GenerateBillsOptions{
		From: "2026-01", To: "2026-01", Amount: &amount, DryRun: true, Now: fixedBulkNow(),
	}); err != nil {
		t.Fatalf("monto explícito 0 debe permitirse: %v", err)
	}

	mustExec(`INSERT INTO bills (service_id, year, month, amount, status) VALUES (1, 2026, 1, 55, 'pending')`)
	if _, err := service.GenerateForService(ctx, 1, GenerateBillsOptions{
		From: "2026-02", To: "2026-02", DryRun: true, Now: fixedBulkNow(),
	}); err != nil {
		t.Fatalf("generar con monto de última factura: %v", err)
	}
}

func TestPayForServiceSelectors(t *testing.T) {
	service, database, mustExec := newBillBulkTestEnv(t)
	ctx := context.Background()
	if _, err := service.GenerateForService(ctx, 1, GenerateBillsOptions{
		From: "2026-01", To: "2026-11", Now: fixedBulkNow(),
	}); err != nil {
		t.Fatalf("preparar facturas: %v", err)
	}
	mustExec(`INSERT INTO bills (service_id, year, month, amount, status) VALUES (1, 2026, 12, 100, 'pending')`)

	january, err := service.PayForService(ctx, 1, PayBillsOptions{
		Month: "2026-01", PaidAt: "2026-11-23", Source: models.BillSourceMCP, Now: fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("pagar enero: %v", err)
	}
	if january.Paid != 1 || january.Skipped != 0 || january.TotalAmount != 100 {
		t.Fatalf("enero = paid:%d skipped:%d total:%v, quiero 1/0/100", january.Paid, january.Skipped, january.TotalAmount)
	}

	rangePay, err := service.PayForService(ctx, 1, PayBillsOptions{
		From: "2026-01", To: "2026-03", PaidAt: "2026-11-23", Now: fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("pagar rango: %v", err)
	}
	if rangePay.Paid != 2 || rangePay.Skipped != 1 || rangePay.TotalAmount != 200 {
		t.Fatalf("rango = paid:%d skipped:%d total:%v, quiero 2/1/200", rangePay.Paid, rangePay.Skipped, rangePay.TotalAmount)
	}
	if rangePay.Items[0].Reason != bulkReasonAlreadyPaid {
		t.Fatalf("motivo enero = %q, quiero %q", rangePay.Items[0].Reason, bulkReasonAlreadyPaid)
	}

	all, err := service.PayForService(ctx, 1, PayBillsOptions{
		AllPendingUpTo: "2026-12", PaidAt: "2026-11-23", Now: fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("pagar todo: %v", err)
	}
	if all.Paid != 8 || all.Skipped != 4 {
		t.Fatalf("todo = paid:%d skipped:%d, quiero 8/4", all.Paid, all.Skipped)
	}
	futureSkipped := 0
	for _, item := range all.Items {
		if item.Reason == bulkReasonFuturePeriod {
			futureSkipped++
		}
	}
	if futureSkipped != 1 {
		t.Fatalf("omitidas por futuro = %d, quiero 1", futureSkipped)
	}

	var source string
	if err := database.QueryRow(`SELECT source FROM bill_history WHERE bill_id = ? AND action = ?`,
		january.Items[0].BillID, models.BillActionPaid).Scan(&source); err != nil {
		t.Fatalf("leer historial de pago: %v", err)
	}
	if source != models.BillSourceMCP {
		t.Fatalf("origen de pago = %q, quiero %q", source, models.BillSourceMCP)
	}

	already, err := service.PayForService(ctx, 1, PayBillsOptions{
		BillIDs: []int64{january.Items[0].BillID}, PaidAt: "2026-11-23", Now: fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("pagar por ID ya pagada: %v", err)
	}
	if already.Paid != 0 || already.Skipped != 1 || already.Items[0].Reason != bulkReasonAlreadyPaid {
		t.Fatalf("ya pagada = paid:%d skipped:%d, quiero 0/1 already_paid", already.Paid, already.Skipped)
	}
}

func TestPayForServiceDryRunAndValidation(t *testing.T) {
	service, database, _ := newBillBulkTestEnv(t)
	ctx := context.Background()
	if _, err := service.GenerateForService(ctx, 1, GenerateBillsOptions{
		From: "2026-01", To: "2026-11", Now: fixedBulkNow(),
	}); err != nil {
		t.Fatalf("preparar facturas: %v", err)
	}

	dry, err := service.PayForService(ctx, 1, PayBillsOptions{
		AllPendingUpTo: "2026-11", PaidAt: "2026-11-23", DryRun: true, Now: fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("dry run de pago: %v", err)
	}
	if dry.Paid != 0 || len(dry.Items) != 11 || dry.TotalAmount != 1100 {
		t.Fatalf("dry run = paid:%d items:%d total:%v, quiero 0/11/1100", dry.Paid, len(dry.Items), dry.TotalAmount)
	}
	var paid int
	if err := database.QueryRow(`SELECT COUNT(*) FROM bills WHERE service_id = 1 AND status = 'paid'`).Scan(&paid); err != nil {
		t.Fatalf("contar pagadas: %v", err)
	}
	if paid != 0 {
		t.Fatalf("pagadas = %d, quiero 0", paid)
	}

	cases := []struct {
		name string
		opts PayBillsOptions
		code string
	}{
		{
			name: "sin selector",
			opts: PayBillsOptions{PaidAt: "2026-11-23", Now: fixedBulkNow()},
			code: BulkErrorInvalidSelector,
		},
		{
			name: "doble selector",
			opts: PayBillsOptions{BillIDs: []int64{1}, Month: "2026-01", PaidAt: "2026-11-23", Now: fixedBulkNow()},
			code: BulkErrorInvalidSelector,
		},
		{
			name: "rango incompleto",
			opts: PayBillsOptions{From: "2026-01", PaidAt: "2026-11-23", Now: fixedBulkNow()},
			code: BulkErrorInvalidSelector,
		},
		{
			name: "fecha futura",
			opts: PayBillsOptions{Month: "2026-01", PaidAt: "2026-12-01", Now: fixedBulkNow()},
			code: BulkErrorInvalidPaidAt,
		},
		{
			name: "factura de otro servicio",
			opts: PayBillsOptions{BillIDs: []int64{9999}, PaidAt: "2026-11-23", Now: fixedBulkNow()},
			code: BulkErrorInvalidBillID,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.PayForService(ctx, 1, tc.opts)
			requireBulkError(t, err, tc.code)
		})
	}
}

func TestPayForServiceYearlyMonthSelector(t *testing.T) {
	service, _, _ := newBillBulkTestEnv(t)
	ctx := context.Background()
	if _, err := service.GenerateForService(ctx, 4, GenerateBillsOptions{
		From: "2026-01", To: "2026-01", Now: fixedBulkNow(),
	}); err != nil {
		t.Fatalf("generar anual: %v", err)
	}
	result, err := service.PayForService(ctx, 4, PayBillsOptions{
		Month: "2026-01", PaidAt: "2026-11-23", Now: fixedBulkNow(),
	})
	if err != nil {
		t.Fatalf("pagar anual por mes: %v", err)
	}
	if result.Paid != 1 {
		t.Fatalf("pagadas = %d, quiero 1", result.Paid)
	}
}
