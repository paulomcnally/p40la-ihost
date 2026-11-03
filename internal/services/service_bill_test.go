package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func newServiceBillTestEnv(t *testing.T) (*BillService, func(string, ...any)) {
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
			t.Fatalf("ejecutar SQL de prueba (%q): %v", query, err)
		}
	}

	mustExec("INSERT INTO homes (name) VALUES ('Casa Demo')")
	mustExec("INSERT INTO homes (name) VALUES ('Casa Demo 2')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Servicio Demo Mensual', 'Proveedor Demo', 1, 'monthly', 100, 1, 'internet', 'fixed', 1)`)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Servicio Demo Con Factura', 'Proveedor Demo', 1, 'monthly', 200, 1, 'water', 'fixed', 1)`)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Servicio Demo Anual', 'Proveedor Demo', 1, 'yearly', 300, 1, 'insurance_car', 'fixed', 1)`)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Servicio Demo Inactivo', 'Proveedor Demo', 1, 'monthly', 999, 0, 'other', 'fixed', 1)`)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring, start_date)
		VALUES (1, 'Servicio Demo Futuro', 'Proveedor Demo', 2, 'monthly', 50, 1, 'other', 'fixed', 1, '2027-01-01')`)

	billStorage := storage.NewBillStorage(database)
	return NewBillService(billStorage, storage.NewServiceStorage(database)), mustExec
}

func TestApplyServiceBillPeriod(t *testing.T) {
	due := "2026-11-15"
	billingDay := 31
	noBillingDay := (*int)(nil)

	tests := []struct {
		name          string
		bill          models.ServiceBillDetail
		wantYear      int
		wantMonth     int
		wantDate      string
		wantEstimated bool
	}{
		{
			name:          "due_date tiene prioridad",
			bill:          models.ServiceBillDetail{Year: 2026, Month: 8, DueDate: &due},
			wantYear:      2026,
			wantMonth:     11,
			wantDate:      "2026-11-15",
			wantEstimated: false,
		},
		{
			name:          "mensual usa year/month y billing_day",
			bill:          models.ServiceBillDetail{Year: 2026, Month: 8, BillingDay: &billingDay},
			wantYear:      2026,
			wantMonth:     8,
			wantDate:      "2026-08-31",
			wantEstimated: true,
		},
		{
			name:          "billing_day se ajusta al ultimo dia del mes",
			bill:          models.ServiceBillDetail{Year: 2026, Month: 2, BillingDay: &billingDay},
			wantYear:      2026,
			wantMonth:     2,
			wantDate:      "2026-02-28",
			wantEstimated: true,
		},
		{
			name:          "sin billing_day usa dia 1",
			bill:          models.ServiceBillDetail{Year: 2026, Month: 5, BillingDay: noBillingDay},
			wantYear:      2026,
			wantMonth:     5,
			wantDate:      "2026-05-01",
			wantEstimated: true,
		},
		{
			name:          "anual sin due_date cae en enero",
			bill:          models.ServiceBillDetail{Year: 2026, Month: 0, BillingDay: noBillingDay},
			wantYear:      2026,
			wantMonth:     1,
			wantDate:      "2026-01-01",
			wantEstimated: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			applyServiceBillPeriod(&tc.bill)
			if tc.bill.PeriodYear != tc.wantYear || tc.bill.PeriodMonth != tc.wantMonth {
				t.Fatalf("período = %d-%02d, quiero %d-%02d", tc.bill.PeriodYear, tc.bill.PeriodMonth, tc.wantYear, tc.wantMonth)
			}
			if tc.bill.PeriodDate != tc.wantDate {
				t.Fatalf("period_date = %q, quiero %q", tc.bill.PeriodDate, tc.wantDate)
			}
			if tc.bill.DateEstimated != tc.wantEstimated {
				t.Fatalf("date_estimated = %v, quiero %v", tc.bill.DateEstimated, tc.wantEstimated)
			}
		})
	}
}

func TestListServiceBillsByPeriodAndSummary(t *testing.T) {
	service, mustExec := newServiceBillTestEnv(t)
	ctx := context.Background()

	mustExec(`INSERT INTO bills (service_id, year, month, amount, status) VALUES (1, 2026, 11, 100, 'pending')`)
	mustExec(`INSERT INTO bills (service_id, year, month, amount, status, due_date) VALUES (1, 2026, 12, 150, 'paid', '2026-12-10')`)
	mustExec(`INSERT INTO bills (service_id, year, month, amount, status) VALUES (3, 2026, 0, 300, 'pending')`)
	mustExec(`INSERT INTO bills (service_id, year, month, amount, status) VALUES (3, 2025, 0, 200, 'paid')`)

	november, err := service.ListServiceBillsByPeriod(ctx, 2026, 11, nil)
	if err != nil {
		t.Fatalf("listar noviembre: %v", err)
	}
	if len(november) != 1 {
		t.Fatalf("noviembre: quiero 1 factura, obtuve %d", len(november))
	}
	if november[0].PeriodDate != "2026-11-01" || !november[0].DateEstimated {
		t.Fatalf("noviembre: período inesperado %+v", november[0])
	}

	december, err := service.ListServiceBillsByPeriod(ctx, 2026, 12, nil)
	if err != nil {
		t.Fatalf("listar diciembre: %v", err)
	}
	if len(december) != 1 || december[0].PeriodDate != "2026-12-10" || december[0].DateEstimated {
		t.Fatalf("diciembre: quiero la factura con due_date, obtuve %+v", december)
	}

	january, err := service.ListServiceBillsByPeriod(ctx, 2026, 1, nil)
	if err != nil {
		t.Fatalf("listar enero: %v", err)
	}
	if len(january) != 1 || january[0].Amount != 300 {
		t.Fatalf("enero: quiero la factura anual, obtuve %+v", january)
	}

	summary, err := service.ServiceBillsSummary(ctx, "2026-01", "2026-12", nil)
	if err != nil {
		t.Fatalf("resumen: %v", err)
	}
	byPeriod := make(map[string]models.ServiceBillSummaryRow)
	for _, row := range summary.Periods {
		byPeriod[row.Period] = row
	}
	if got := byPeriod["2026-01"]; got.Total != 300 || got.Pending != 300 {
		t.Fatalf("resumen enero = %+v", got)
	}
	if got := byPeriod["2026-11"]; got.Total != 100 || got.Pending != 100 {
		t.Fatalf("resumen noviembre = %+v", got)
	}
	if got := byPeriod["2026-12"]; got.Total != 150 || got.Paid != 150 || got.Pending != 0 {
		t.Fatalf("resumen diciembre = %+v", got)
	}

	homeOne := int64(1)
	filtered, err := service.ListServiceBillsByPeriod(ctx, 2026, 11, &homeOne)
	if err != nil {
		t.Fatalf("listar con home: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("filtro por casa: quiero 1, obtuve %d", len(filtered))
	}
}

func TestServiceBillsProjection(t *testing.T) {
	service, mustExec := newServiceBillTestEnv(t)
	ctx := context.Background()

	mustExec(`INSERT INTO bills (service_id, year, month, amount, status) VALUES (2, 2026, 11, 250, 'pending')`)

	projection, err := service.ServiceBillsProjection(ctx, 2026, 11, 3, nil)
	if err != nil {
		t.Fatalf("proyección: %v", err)
	}

	byPeriod := make(map[string]models.ServiceProjectionPeriod)
	for _, period := range projection.Periods {
		byPeriod[period.Period+"|"+period.CurrencyCode] = period
	}

	nov := byPeriod["2026-11|NIO"]
	if nov.Confirmed != 250 || nov.Estimated != 100 || nov.Total != 350 {
		t.Fatalf("noviembre = %+v, quiero confirmed 250 / estimated 100 / total 350", nov)
	}
	if len(nov.Services) != 2 {
		t.Fatalf("noviembre: quiero 2 servicios, obtuve %d", len(nov.Services))
	}
	for _, item := range nov.Services {
		if item.ServiceID == 2 && item.Source != "bill" {
			t.Fatalf("servicio con factura debe ser bill: %+v", item)
		}
	}

	dec := byPeriod["2026-12|NIO"]
	if dec.Confirmed != 0 || dec.Estimated != 300 || dec.Total != 300 {
		t.Fatalf("diciembre = %+v, quiero estimated 300 (100 + 200 mensuales)", dec)
	}

	jan := byPeriod["2027-01|NIO"]
	if jan.Confirmed != 0 || jan.Estimated != 600 || jan.Total != 600 {
		t.Fatalf("enero NIO = %+v, quiero estimated 600 (100 + 200 mensuales + 300 anual)", jan)
	}
	janUSD := byPeriod["2027-01|USD"]
	if janUSD.Confirmed != 0 || janUSD.Estimated != 50 || janUSD.Total != 50 {
		t.Fatalf("enero USD = %+v, quiero estimated 50 del servicio futuro", janUSD)
	}
}
