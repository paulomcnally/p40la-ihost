package services

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

func TestDebtContractLifecycle(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	if _, err := database.ExecContext(ctx, "INSERT INTO institutions (name) VALUES ('Proveedor Demo')"); err != nil {
		t.Fatalf("insertar institución: %v", err)
	}

	svc := NewDebtService(
		storage.NewDebtStorage(database),
		storage.NewDebtBillStorage(database),
		storage.NewInstitutionStorage(database),
		storage.NewCurrencyStorage(database),
	)
	store := NewContractStore(t.TempDir())
	svc.SetContractStore(store)

	debt, err := svc.Create(ctx, &models.Debt{
		InstitutionID:     1,
		Identifier:        "demo-account-001",
		Description:       "Deuda Demo",
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
	if debt.HasContract {
		t.Fatal("HasContract debería ser false al crear")
	}

	if err := svc.SaveContract(ctx, debt.ID, "contrato.pdf", bytes.NewReader(testPDF("deuda"))); err != nil {
		t.Fatalf("SaveContract: %v", err)
	}

	reloaded, err := svc.GetByID(ctx, debt.ID)
	if err != nil || reloaded == nil {
		t.Fatalf("GetByID: %v, %v", reloaded, err)
	}
	if !reloaded.HasContract {
		t.Error("HasContract debería ser true después de guardar")
	}

	path, err := svc.ContractPath(ctx, debt.ID)
	if err != nil {
		t.Fatalf("ContractPath: %v", err)
	}
	if !strings.Contains(path, "contracts/debts") {
		t.Errorf("ruta inesperada: %q", path)
	}

	list, err := svc.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v, %v", list, err)
	}
	if !list[0].HasContract {
		t.Error("List debería completar HasContract")
	}

	if err := svc.DeleteContract(ctx, debt.ID); err != nil {
		t.Fatalf("DeleteContract: %v", err)
	}
	reloaded, err = svc.GetByID(ctx, debt.ID)
	if err != nil || reloaded == nil {
		t.Fatalf("GetByID después de borrar: %v, %v", reloaded, err)
	}
	if reloaded.HasContract {
		t.Error("HasContract debería ser false después de borrar")
	}

	if err := svc.SaveContract(ctx, 99999, "contrato.pdf", bytes.NewReader(testPDF("x"))); !errors.Is(err, ErrContractEntityNotFound) {
		t.Errorf("SaveContract de deuda inexistente = %v, want ErrContractEntityNotFound", err)
	}
	if _, err := svc.ContractPath(ctx, 99999); !errors.Is(err, ErrContractEntityNotFound) {
		t.Errorf("ContractPath de deuda inexistente = %v, want ErrContractEntityNotFound", err)
	}
	if err := svc.DeleteContract(ctx, 99999); !errors.Is(err, ErrContractEntityNotFound) {
		t.Errorf("DeleteContract de deuda inexistente = %v, want ErrContractEntityNotFound", err)
	}
}

func TestServiceContractLifecycle(t *testing.T) {
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	ctx := context.Background()
	if _, err := database.ExecContext(ctx, "INSERT INTO homes (name) VALUES ('Casa Demo')"); err != nil {
		t.Fatalf("insertar casa: %v", err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO services
		(home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Servicio Demo', 'Proveedor Demo', 1, 'monthly', 100, 1, 'internet', 'fixed', 1)`); err != nil {
		t.Fatalf("insertar servicio: %v", err)
	}

	svc := NewServiceService(
		storage.NewServiceStorage(database),
		storage.NewHomeStorage(database),
		storage.NewCurrencyStorage(database),
		storage.NewBillStorage(database),
	)
	store := NewContractStore(t.TempDir())
	svc.SetContractStore(store)

	list, err := svc.List(ctx, nil)
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v, %v", list, err)
	}
	if list[0].HasContract {
		t.Fatal("HasContract debería ser false sin contrato")
	}

	if err := svc.SaveContract(ctx, list[0].ID, "contrato.pdf", bytes.NewReader(testPDF("servicio"))); err != nil {
		t.Fatalf("SaveContract: %v", err)
	}

	reloaded, err := svc.GetByID(ctx, list[0].ID)
	if err != nil || reloaded == nil {
		t.Fatalf("GetByID: %v, %v", reloaded, err)
	}
	if !reloaded.HasContract {
		t.Error("HasContract debería ser true después de guardar")
	}

	path, err := svc.ContractPath(ctx, list[0].ID)
	if err != nil {
		t.Fatalf("ContractPath: %v", err)
	}
	if !strings.Contains(path, "contracts/services") {
		t.Errorf("ruta inesperada: %q", path)
	}

	if err := svc.DeleteContract(ctx, list[0].ID); err != nil {
		t.Fatalf("DeleteContract: %v", err)
	}
	reloaded, err = svc.GetByID(ctx, list[0].ID)
	if err != nil || reloaded == nil {
		t.Fatalf("GetByID después de borrar: %v, %v", reloaded, err)
	}
	if reloaded.HasContract {
		t.Error("HasContract debería ser false después de borrar")
	}

	if err := svc.SaveContract(ctx, 99999, "contrato.pdf", bytes.NewReader(testPDF("x"))); !errors.Is(err, ErrContractEntityNotFound) {
		t.Errorf("SaveContract de servicio inexistente = %v, want ErrContractEntityNotFound", err)
	}
}
