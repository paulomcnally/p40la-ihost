package services

import (
	"strings"
	"testing"
	"time"

	appmodels "github.com/paulomcnally/p40la-ihost/internal/models"
)

func TestFormatInsuranceAlerts(t *testing.T) {
	t.Run("sin alertas", func(t *testing.T) {
		got := formatInsuranceAlerts(nil)
		if len(got) != 1 || !strings.Contains(got[0], "No hay autos en condición de alerta") {
			t.Errorf("esperado mensaje vacío, got %q", got)
		}
	})

	t.Run("autos sin seguro y con seguro vencido", func(t *testing.T) {
		alerts := []appmodels.AutoAlert{
			{AutoID: 1, Brand: "Toyota", Model: "Corolla", Year: 2020, Placa: "ABC123", AlertType: appmodels.AlertTypeNoInsurance},
			{AutoID: 2, Brand: "Honda", Model: "Civic", Year: 2019, Placa: "XYZ789", AlertType: appmodels.AlertTypeExpired, EndDate: "2026-09-01"},
		}
		got := formatInsuranceAlerts(alerts)
		if len(got) != 1 {
			t.Fatalf("esperado 1 mensaje, got %d", len(got))
		}
		msg := got[0]
		if !strings.Contains(msg, "Toyota Corolla 2020") || !strings.Contains(msg, "ABC123") || !strings.Contains(msg, "Sin seguro asociado") {
			t.Errorf("auto sin seguro mal formateado:\n%s", msg)
		}
		if !strings.Contains(msg, "Honda Civic 2019") || !strings.Contains(msg, "XYZ789") || !strings.Contains(msg, "Seguro vencido el 01/09/2026") {
			t.Errorf("auto con seguro vencido mal formateado:\n%s", msg)
		}
	})
}

func TestFormatBillCreatedAlert(t *testing.T) {
	one := formatBillCreatedAlert(1)
	if len(one) != 1 || !strings.Contains(one[0], "1 factura automática") {
		t.Errorf("singular incorrecto: %q", one)
	}
	many := formatBillCreatedAlert(3)
	if len(many) != 1 || !strings.Contains(many[0], "3 facturas automáticas") {
		t.Errorf("plural incorrecto: %q", many)
	}
}

func TestFormatDebtsDueToday(t *testing.T) {
	format := DefaultCurrencyFormat()
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

	t.Run("sin cuotas que vencen hoy", func(t *testing.T) {
		pending := []appmodels.PendingDebtDetail{
			{DebtID: 1, DebtDescription: "Deuda A", InstitutionName: "Inst", DueDate: "2026-09-20", Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatDebtsDueToday(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		if len(got) != 1 || !strings.Contains(got[0], "No hay cuotas que vencen hoy") {
			t.Errorf("esperado mensaje vacío, got %q", got)
		}
	})

	t.Run("solo cuotas que vencen hoy", func(t *testing.T) {
		pending := []appmodels.PendingDebtDetail{
			{DebtID: 1, DebtDescription: "Deuda A", InstitutionName: "Inst", DueDate: "2026-09-19", Amount: 100, CurrencySymbol: "C$", CurrencyCode: "NIO"},
			{DebtID: 2, DebtDescription: "Deuda B", InstitutionName: "Inst2", DueDate: "2026-09-20", Amount: 500, CurrencySymbol: "C$", CurrencyCode: "NIO"},
		}
		got := formatDebtsDueToday(pending, format, now, DefaultTelegramBotSeparatorLength, DefaultTelegramBotShowMonths)
		if len(got) != 2 {
			t.Fatalf("esperados 2 mensajes (1 deuda + totales), got %d: %q", len(got), got)
		}
		if !strings.Contains(got[0], "Deuda A") || strings.Contains(got[0], "Deuda B") {
			t.Errorf("solo la cuota de hoy debe aparecer:\n%s", got[0])
		}
		if !strings.Contains(got[0], "Vence hoy") {
			t.Errorf("la cuota de hoy debe marcar semáforo rojo:\n%s", got[0])
		}
	})
}

func TestFormatPensionRecordsCreated(t *testing.T) {
	format := DefaultCurrencyFormat()
	salaries := []appmodels.SalaryPayment{
		{Employer: "Empleador A", Amount: 1000, Currency: "NIO"},
	}
	records := []appmodels.SupportRecord{
		{ChildName: "Hijo 1", CategoryName: "Alimentos", Amount: 200, Currency: "NIO", Month: 9, Year: 2026},
	}
	got := formatPensionRecordsCreated(salaries, records, 2026, 9, format)
	if len(got) != 1 {
		t.Fatalf("esperado 1 mensaje, got %d", len(got))
	}
	msg := got[0]
	if !strings.Contains(msg, "2 registro(s) creado(s)") || !strings.Contains(msg, "Septiembre 2026") {
		t.Errorf("encabezado incorrecto:\n%s", msg)
	}
	if !strings.Contains(msg, "Empleador A") || !strings.Contains(msg, "Hijo 1") || !strings.Contains(msg, "Alimentos") {
		t.Errorf("detalle incorrecto:\n%s", msg)
	}
}

func TestFormatPensionRecordPaid(t *testing.T) {
	format := DefaultCurrencyFormat()
	method := "efectivo"
	reference := "REF-1"
	record := &appmodels.SupportRecord{
		ChildName: "Hijo 1", CategoryName: "Alimentos", Amount: 200, Currency: "NIO",
		Month: 9, Year: 2026, PaymentMethod: &method, PaymentReference: &reference,
	}
	got := formatPensionRecordPaid(record, format)
	if len(got) != 1 {
		t.Fatalf("esperado 1 mensaje, got %d", len(got))
	}
	if !strings.Contains(got[0], "Pago registrado") || !strings.Contains(got[0], "REF-1") {
		t.Errorf("aviso de pago incorrecto:\n%s", got[0])
	}
}

func TestFormatPensionMonthClosing(t *testing.T) {
	format := DefaultCurrencyFormat()
	records := []appmodels.SupportRecord{
		{ChildName: "Hijo 1", CategoryName: "Alimentos", Amount: 100, Currency: "NIO", Status: "paid", Month: 9, Year: 2026},
		{ChildName: "Hijo 2", CategoryName: "Alimentos", Amount: 300, Currency: "NIO", Status: "pending", Month: 9, Year: 2026},
	}
	got := formatPensionMonthClosing(records, nil, 2026, 9, format)
	if len(got) != 1 {
		t.Fatalf("esperado 1 mensaje, got %d", len(got))
	}
	msg := got[0]
	if !strings.Contains(msg, "Cierre de mes") || !strings.Contains(msg, "Pagados") || !strings.Contains(msg, "Pendientes") {
		t.Errorf("resumen de cierre incorrecto:\n%s", msg)
	}
}
