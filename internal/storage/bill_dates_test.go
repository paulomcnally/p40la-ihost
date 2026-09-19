package storage

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
)

func TestBillStorageDates(t *testing.T) {
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

	store := NewBillStorage(database)

	// Create sin fechas: NULL por defecto (no requeridas).
	issue, due := "2026-09-10", "2026-10-05"
	bill, err := store.Create(ctx, &models.Bill{
		ServiceID: 1, Year: 2026, Month: 9, Amount: 1250, Status: "pending",
	})
	if err != nil {
		t.Fatalf("Create sin fechas: %v", err)
	}
	if bill.IssueDate != nil || bill.DueDate != nil {
		t.Errorf("sin fechas se esperaba nil/nil, got %v/%v", bill.IssueDate, bill.DueDate)
	}

	// Create con fechas.
	bill2, err := store.Create(ctx, &models.Bill{
		ServiceID: 1, Year: 2026, Month: 10, Amount: 1250, Status: "pending",
		IssueDate: &issue, DueDate: &due,
	})
	if err != nil {
		t.Fatalf("Create con fechas: %v", err)
	}
	if bill2.IssueDate == nil || *bill2.IssueDate != issue {
		t.Errorf("issue_date esperado %q, got %v", issue, bill2.IssueDate)
	}
	if bill2.DueDate == nil || *bill2.DueDate != due {
		t.Errorf("due_date esperado %q, got %v", due, bill2.DueDate)
	}

	// UpdateWebhookFields sin fechas: conserva los valores previos (COALESCE).
	if err := store.UpdateWebhookFields(ctx, bill2.ID, 1300, "INV", "", nil, nil); err != nil {
		t.Fatalf("UpdateWebhookFields sin fechas: %v", err)
	}
	got, err := store.GetByID(ctx, bill2.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.IssueDate == nil || *got.IssueDate != issue {
		t.Errorf("issue_date previo debe conservarse, got %v", got.IssueDate)
	}
	if got.DueDate == nil || *got.DueDate != due {
		t.Errorf("due_date previo debe conservarse, got %v", got.DueDate)
	}

	// UpdateWebhookFields con fechas nuevas: sobrescriben.
	newIssue, newDue := "2026-09-11", "2026-10-06"
	if err := store.UpdateWebhookFields(ctx, bill2.ID, 1300, "INV", "", &newIssue, &newDue); err != nil {
		t.Fatalf("UpdateWebhookFields con fechas: %v", err)
	}
	got, err = store.GetByID(ctx, bill2.ID)
	if err != nil {
		t.Fatalf("GetByID tras update: %v", err)
	}
	if got.IssueDate == nil || *got.IssueDate != newIssue {
		t.Errorf("issue_date nuevo esperado %q, got %v", newIssue, got.IssueDate)
	}
	if got.DueDate == nil || *got.DueDate != newDue {
		t.Errorf("due_date nuevo esperado %q, got %v", newDue, got.DueDate)
	}
}

func TestDebtBillStorageIssueDate(t *testing.T) {
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
	mustExec(`INSERT INTO institutions (name) VALUES ('Banco')`)
	mustExec(`INSERT INTO debts (institution_id, identifier, description, total, principal, currency_id, installments_total, installment_amount, interest_rate, payment_day, start_date, status)
		VALUES (1, 'D-001', 'Deuda prueba', 1000, 1000, 1, 3, 333.33, 0, 5, '2026-09-01', 'activa')`)

	store := NewDebtBillStorage(database)
	issue := "2026-09-10"

	// Sin issue_date: NULL.
	bill, err := store.Create(ctx, &models.DebtBill{
		DebtID: 1, InstallmentNumber: 1, DueDate: "2026-10-05", Amount: 333.33, Status: "pending",
	})
	if err != nil {
		t.Fatalf("Create cuota sin issue_date: %v", err)
	}
	if bill.IssueDate != nil {
		t.Errorf("sin issue_date se esperaba nil, got %v", bill.IssueDate)
	}

	// Con issue_date.
	bill2, err := store.Create(ctx, &models.DebtBill{
		DebtID: 1, InstallmentNumber: 2, DueDate: "2026-11-05", IssueDate: &issue, Amount: 333.33, Status: "pending",
	})
	if err != nil {
		t.Fatalf("Create cuota con issue_date: %v", err)
	}
	if bill2.IssueDate == nil || *bill2.IssueDate != issue {
		t.Errorf("issue_date esperado %q, got %v", issue, bill2.IssueDate)
	}
	if bill2.DueDate != "2026-11-05" {
		t.Errorf("due_date esperado 2026-11-05, got %q", bill2.DueDate)
	}
}
