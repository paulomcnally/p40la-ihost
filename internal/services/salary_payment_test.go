package services

import (
	"context"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

type salaryPaymentFixture struct {
	svc      *SalaryPaymentService
	closing  *MonthClosingService
	salaryID int64
}

func newSalaryPaymentFixture(t *testing.T) *salaryPaymentFixture {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	closingStorage := storage.NewMonthClosingStorage(database)
	closingSvc := NewMonthClosingService(closingStorage)
	svc := NewSalaryPaymentService(storage.NewSalaryPaymentStorage(database), closingStorage)

	ctx := context.Background()
	salarySvc := NewSalaryService(storage.NewSalaryStorage(database), storage.NewCurrencyStorage(database))
	salary, err := salarySvc.Create(ctx, "Empresa XYZ", 15000, 1, 15, true, "")
	if err != nil {
		t.Fatalf("crear salario: %v", err)
	}

	payment, err := svc.storage.Create(ctx, &models.SalaryPayment{
		SalaryID: salary.ID,
		Year:     2026,
		Month:    8,
		Amount:   15000,
		Currency: "NIO",
		Status:   "pending",
	})
	if err != nil {
		t.Fatalf("crear pago de salario: %v", err)
	}

	return &salaryPaymentFixture{svc: svc, closing: closingSvc, salaryID: payment.ID}
}

func TestSalaryPaymentService_MarkReceived(t *testing.T) {
	f := newSalaryPaymentFixture(t)
	ctx := context.Background()

	// receivedAt requerido
	if _, err := f.svc.MarkReceived(ctx, f.salaryID, time.Time{}, nil, nil); err == nil {
		t.Fatal("se esperaba error por fecha de recepción vacía")
	}

	receivedAt := parseTime(t, "2026-08-15T12:00:00")
	receivedAmount := 14500.0
	notes := "Depósito bancario"
	payment, err := f.svc.MarkReceived(ctx, f.salaryID, receivedAt, &receivedAmount, &notes)
	if err != nil {
		t.Fatalf("mark received: %v", err)
	}
	if payment.Status != "received" || payment.ReceivedAt == nil || payment.ReceivedAmount == nil || *payment.ReceivedAmount != 14500 {
		t.Fatalf("datos de recepción incorrectos: %+v", payment)
	}

	// MarkPending limpia
	payment, err = f.svc.MarkPending(ctx, f.salaryID)
	if err != nil {
		t.Fatalf("mark pending: %v", err)
	}
	if payment.Status != "pending" || payment.ReceivedAt != nil || payment.ReceivedAmount != nil {
		t.Fatalf("mark pending debería limpiar recepción: %+v", payment)
	}
}

func TestSalaryPaymentService_MonthClosed(t *testing.T) {
	f := newSalaryPaymentFixture(t)
	ctx := context.Background()

	if _, err := f.closing.Close(ctx, 2026, 8); err != nil {
		t.Fatalf("cerrar mes: %v", err)
	}
	receivedAt := parseTime(t, "2026-08-15T12:00:00")
	if _, err := f.svc.MarkReceived(ctx, f.salaryID, receivedAt, nil, nil); err == nil {
		t.Fatal("se esperaba error al marcar recibido en mes cerrado")
	}
	if _, err := f.svc.MarkPending(ctx, f.salaryID); err == nil {
		t.Fatal("se esperaba error al marcar pendiente en mes cerrado")
	}
}

func TestSalaryPaymentService_List(t *testing.T) {
	f := newSalaryPaymentFixture(t)
	ctx := context.Background()

	payments, err := f.svc.List(ctx, 2026, 8, 0)
	if err != nil {
		t.Fatalf("list pagos: %v", err)
	}
	if len(payments) != 1 {
		t.Fatalf("se esperaba 1 pago, count=%d", len(payments))
	}
	if payments[0].Employer == "" {
		t.Fatal("se esperaba employer resuelto por JOIN")
	}

	empty, err := f.svc.List(ctx, 2026, 9, 0)
	if err != nil {
		t.Fatalf("list pagos vacío: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("se esperaba lista vacía, count=%d", len(empty))
	}
}
