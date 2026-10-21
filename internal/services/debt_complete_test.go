package services

import (
	"context"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// SPEC-104: cierre automático al pagar la última cuota, reconciliación y
// coherencia al editar el total de cuotas.
func TestDebtCloseLifecycle(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	if _, err := database.ExecContext(ctx, "INSERT INTO institutions (name) VALUES ('Banco Uno')"); err != nil {
		t.Fatalf("insertar institución: %v", err)
	}

	svc := NewDebtService(
		storage.NewDebtStorage(database),
		storage.NewDebtBillStorage(database),
		storage.NewInstitutionStorage(database),
		storage.NewCurrencyStorage(database),
	)

	debt, err := svc.Create(ctx, &models.Debt{
		InstitutionID:     1,
		Identifier:        "D-1",
		Description:       "Préstamo de prueba",
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
		t.Fatalf("Create: %v", err)
	}

	bills, err := svc.ListBills(ctx, debt.ID)
	if err != nil || len(bills) != 1 {
		t.Fatalf("ListBills: got %d, err %v", len(bills), err)
	}

	paid, err := svc.PayBill(ctx, bills[0].ID, time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), "")
	if err != nil {
		t.Fatalf("PayBill: %v", err)
	}
	if paid.Status != "paid" {
		t.Errorf("cuota pagada: got %q", paid.Status)
	}

	reloaded, err := svc.GetByID(ctx, debt.ID)
	if err != nil || reloaded == nil {
		t.Fatalf("GetByID: %v, %v", reloaded, err)
	}
	if reloaded.Status != DebtStatusFinished {
		t.Errorf("deuda completada: got %q, want %q", reloaded.Status, DebtStatusFinished)
	}
	if reloaded.PendingCount != 0 || reloaded.PaidCount != 1 || reloaded.PaidAmount != 100 {
		t.Errorf("resumen inesperado: %+v", reloaded)
	}

	// REQ-009a: agregar cuotas a una deuda finalizada la reactiva.
	reopened, err := svc.Update(ctx, &models.Debt{
		ID:                debt.ID,
		InstitutionID:     1,
		Identifier:        "D-1",
		Description:       "Préstamo de prueba",
		Total:             300,
		Principal:         300,
		CurrencyID:        1,
		InstallmentsTotal: 3,
		InstallmentAmount: 100,
		PaymentDay:        5,
		StartDate:         "2026-09-01",
		Status:            DebtStatusFinished,
	})
	if err != nil {
		t.Fatalf("Update (ampliar): %v", err)
	}
	if reopened.Status != DebtStatusActive {
		t.Errorf("deuda ampliada: got %q, want %q", reopened.Status, DebtStatusActive)
	}
	bills, err = svc.ListBills(ctx, debt.ID)
	if err != nil || len(bills) != 3 {
		t.Fatalf("cuotas tras ampliar: got %d, err %v", len(bills), err)
	}

	// REQ-009b: reducir cuotas elimina lógicamente las sobrantes.
	reduced, err := svc.Update(ctx, &models.Debt{
		ID:                debt.ID,
		InstitutionID:     1,
		Identifier:        "D-1",
		Description:       "Préstamo de prueba",
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
		t.Fatalf("Update (reducir): %v", err)
	}
	bills, err = svc.ListBills(ctx, debt.ID)
	if err != nil || len(bills) != 1 {
		t.Fatalf("cuotas tras reducir: got %d, err %v", len(bills), err)
	}
	if reduced.PendingCount != 0 {
		t.Errorf("tras reducir, pendientes: got %d, want 0", reduced.PendingCount)
	}

	// REQ-002: el reconciliador cierra la deuda que quedó completa.
	if err := svc.ReconcileDebtBills(ctx); err != nil {
		t.Fatalf("ReconcileDebtBills: %v", err)
	}
	final, err := svc.GetByID(ctx, debt.ID)
	if err != nil || final == nil {
		t.Fatalf("GetByID final: %v, %v", final, err)
	}
	if final.Status != DebtStatusFinished {
		t.Errorf("tras reconciliar: got %q, want %q", final.Status, DebtStatusFinished)
	}
}
