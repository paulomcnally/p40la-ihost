package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// newTestBudget arma el BudgetService + helpers sobre una DB en memoria.
func newTestBudget(t *testing.T) (*BudgetService, *BudgetTransactionService, *CategoryGroupService, *CategoryService, *AccountService, int64) {
	t.Helper()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cgStorage := storage.NewCategoryGroupStorage(database)
	catStorage := storage.NewCategoryStorage(database)
	monthStorage := storage.NewBudgetMonthStorage(database)
	assignStorage := storage.NewAssignmentStorage(database)
	ruleStorage := storage.NewRecurringRuleStorage(database)
	txStorage := storage.NewTransactionStorage(database)
	accountStorage := storage.NewAccountStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)

	currencies, err := currencyStorage.List(context.Background())
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}

	budgetSvc := NewBudgetService(cgStorage, catStorage, monthStorage, assignStorage, ruleStorage, txStorage, currencyStorage)
	txSvc := NewBudgetTransactionService(txStorage, accountStorage, catStorage, currencyStorage)
	groupSvc := NewCategoryGroupService(cgStorage, catStorage)
	catSvc := NewCategoryService(catStorage, cgStorage)
	accountSvc := NewAccountService(accountStorage, currencyStorage)

	return budgetSvc, txSvc, groupSvc, catSvc, accountSvc, currencies[0].ID
}

func TestBudgetAssignAvailable(t *testing.T) {
	budgetSvc, txSvc, groupSvc, catSvc, accountSvc, currencyID := newTestBudget(t)
	ctx := context.Background()

	group, err := groupSvc.Create(ctx, "Necesidades", "home")
	if err != nil {
		t.Fatalf("crear grupo: %v", err)
	}
	cat, err := catSvc.Create(ctx, &models.Category{
		CategoryGroupID: group.ID,
		Name:            "Alquiler",
		Icon:            "home",
	})
	if err != nil {
		t.Fatalf("crear categoría: %v", err)
	}
	account, err := accountSvc.Create(ctx, &models.Account{
		Name:       "Efectivo",
		Type:       "cash",
		CurrencyID: currencyID,
	})
	if err != nil {
		t.Fatalf("crear cuenta: %v", err)
	}

	// Asignar 300 en el mes 2026-09.
	if _, err := budgetSvc.Assign(ctx, 2026, 9, cat.ID, currencyID, 300, false); err != nil {
		t.Fatalf("asignar: %v", err)
	}

	view, err := budgetSvc.MonthView(ctx, 2026, 9)
	if err != nil {
		t.Fatalf("vista mensual: %v", err)
	}
	if len(view.Groups) != 1 || len(view.Groups[0].Categories) != 1 {
		t.Fatalf("esperaba 1 grupo con 1 categoría, got %d grupos", len(view.Groups))
	}
	row := view.Groups[0].Categories[0]
	if row.Assigned[currencyID] != 300 {
		t.Errorf("assigned esperado 300, got %v", row.Assigned[currencyID])
	}
	if row.Activity[currencyID] != 0 {
		t.Errorf("activity esperado 0, got %v", row.Activity[currencyID])
	}
	if row.Available[currencyID] != 300 {
		t.Errorf("available esperado 300, got %v", row.Available[currencyID])
	}

	// Registrar una transacción de outflow 100 en la categoría.
	if _, err := txSvc.Create(ctx, &models.Transaction{
		AccountID:  account.ID,
		CategoryID: &cat.ID,
		CurrencyID: currencyID,
		Date:       "2026-09-15",
		Payee:      "Alquiler",
		Outflow:    100,
	}); err != nil {
		t.Fatalf("crear transacción: %v", err)
	}

	// Recalcular.
	view2, err := budgetSvc.MonthView(ctx, 2026, 9)
	if err != nil {
		t.Fatalf("vista mensual 2: %v", err)
	}
	row2 := view2.Groups[0].Categories[0]
	if row2.Activity[currencyID] != 100 {
		t.Errorf("activity esperado 100, got %v", row2.Activity[currencyID])
	}
	if row2.Available[currencyID] != 200 {
		t.Errorf("available esperado 200 (300-100), got %v", row2.Available[currencyID])
	}
}

func TestBudgetAssignmentNoDuplicate(t *testing.T) {
	budgetSvc, _, groupSvc, catSvc, _, currencyID := newTestBudget(t)
	ctx := context.Background()

	group, _ := groupSvc.Create(ctx, "Gustos", "star")
	cat, _ := catSvc.Create(ctx, &models.Category{CategoryGroupID: group.ID, Name: "Salidas"})

	if _, err := budgetSvc.Assign(ctx, 2026, 9, cat.ID, currencyID, 100, false); err != nil {
		t.Fatalf("asignar 100: %v", err)
	}
	if _, err := budgetSvc.Assign(ctx, 2026, 9, cat.ID, currencyID, 50, false); err != nil {
		t.Fatalf("asignar 50: %v", err)
	}

	view, err := budgetSvc.MonthView(ctx, 2026, 9)
	if err != nil {
		t.Fatalf("vista: %v", err)
	}
	row := view.Groups[0].Categories[0]
	if row.Assigned[currencyID] != 50 {
		t.Errorf("asignar dos veces debería editar (50), got %v", row.Assigned[currencyID])
	}
}

func TestBudgetRecurringRuleMaterializes(t *testing.T) {
	budgetSvc, _, groupSvc, catSvc, _, currencyID := newTestBudget(t)
	ctx := context.Background()

	group, _ := groupSvc.Create(ctx, "Fijos", "credit")
	cat, _ := catSvc.Create(ctx, &models.Category{CategoryGroupID: group.ID, Name: "Suscripción"})

	// Crear regla recurrente desde 2026-09.
	if _, err := budgetSvc.Assign(ctx, 2026, 9, cat.ID, currencyID, 15, true); err != nil {
		t.Fatalf("asignar recurrente: %v", err)
	}

	// El mes siguiente (2026-10) debe materializar 15 automáticamente.
	view, err := budgetSvc.MonthView(ctx, 2026, 10)
	if err != nil {
		t.Fatalf("vista oct: %v", err)
	}
	row := view.Groups[0].Categories[0]
	if row.Assigned[currencyID] != 15 {
		t.Errorf("materialización esperada 15, got %v", row.Assigned[currencyID])
	}
	if row.RecurringRule == nil {
		t.Error("se esperaba recurring_rule en la fila")
	}

	// Pausar la regla: el mes siguiente no debe materializar.
	if row.RecurringRule != nil {
		if _, err := budgetSvc.ToggleRecurring(ctx, row.RecurringRule.ID, false); err != nil {
			t.Fatalf("pausar regla: %v", err)
		}
	}
	view2, err := budgetSvc.MonthView(ctx, 2026, 11)
	if err != nil {
		t.Fatalf("vista nov: %v", err)
	}
	row2 := view2.Groups[0].Categories[0]
	if row2.Assigned[currencyID] != 0 {
		t.Errorf("regla pausada no debe materializar (0), got %v", row2.Assigned[currencyID])
	}
}

func TestBudgetRecurringDoesNotAffectPastMonths(t *testing.T) {
	budgetSvc, _, groupSvc, catSvc, _, currencyID := newTestBudget(t)
	ctx := context.Background()

	group, _ := groupSvc.Create(ctx, "Fijos", "credit")
	cat, _ := catSvc.Create(ctx, &models.Category{CategoryGroupID: group.ID, Name: "Suscripción"})

	// Mes 2026-09 con asignación puntual de 5.
	if _, err := budgetSvc.Assign(ctx, 2026, 9, cat.ID, currencyID, 5, false); err != nil {
		t.Fatalf("asignar sept puntual: %v", err)
	}
	// Crear regla recurrente de 15 desde 2026-10.
	if _, err := budgetSvc.Assign(ctx, 2026, 10, cat.ID, currencyID, 15, true); err != nil {
		t.Fatalf("asignar oct recurrente: %v", err)
	}

	view, err := budgetSvc.MonthView(ctx, 2026, 9)
	if err != nil {
		t.Fatalf("vista sept: %v", err)
	}
	row := view.Groups[0].Categories[0]
	if row.Assigned[currencyID] != 5 {
		t.Errorf("sept debe conservar 5 (sin afectar por recurrente), got %v", row.Assigned[currencyID])
	}
}

func TestBudgetCategoryDeleteArchivesWithHistory(t *testing.T) {
	budgetSvc, txSvc, groupSvc, catSvc, accountSvc, currencyID := newTestBudget(t)
	ctx := context.Background()

	group, _ := groupSvc.Create(ctx, "Grupo", "star")
	cat, _ := catSvc.Create(ctx, &models.Category{CategoryGroupID: group.ID, Name: "Categoría"})

	// Crear cuenta para poder transaccionar.
	account, err := accountSvc.Create(ctx, &models.Account{
		Name:       "Efectivo",
		Type:       "cash",
		CurrencyID: currencyID,
	})
	if err != nil {
		t.Fatalf("crear cuenta: %v", err)
	}

	if _, err := txSvc.Create(ctx, &models.Transaction{
		AccountID:  account.ID,
		CategoryID: &cat.ID,
		CurrencyID: currencyID,
		Date:       "2026-09-10",
		Payee:      "Pago",
		Outflow:    20,
	}); err != nil {
		t.Fatalf("crear transacción: %v", err)
	}

	// Eliminar la categoría con historial → debe archivarse.
	if err := catSvc.Delete(ctx, cat.ID); err != nil {
		t.Fatalf("eliminar categoría con historial: %v", err)
	}

	// La vista mensual no debe mostrarla.
	view, err := budgetSvc.MonthView(ctx, 2026, 9)
	if err != nil {
		t.Fatalf("vista: %v", err)
	}
	if len(view.Groups) != 1 || len(view.Groups[0].Categories) != 0 {
		t.Errorf("categoría archivada no debe aparecer en la vista, got %d", len(view.Groups[0].Categories))
	}

	// La transacción sigue consultable por categoría (historial conservado).
	history, err := txSvc.ListByCategoryMonth(ctx, cat.ID, 2026, 9)
	if err != nil {
		t.Fatalf("historial: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("historial conservado esperado 1, got %d", len(history))
	}
}

func TestBudgetUncategorizedTransactionKeepsTotals(t *testing.T) {
	budgetSvc, txSvc, groupSvc, catSvc, accountSvc, currencyID := newTestBudget(t)
	ctx := context.Background()

	group, _ := groupSvc.Create(ctx, "Grupo", "star")
	cat, _ := catSvc.Create(ctx, &models.Category{CategoryGroupID: group.ID, Name: "Comida"})
	if _, err := budgetSvc.Assign(ctx, 2026, 9, cat.ID, currencyID, 100, false); err != nil {
		t.Fatalf("asignar: %v", err)
	}

	account, err := accountSvc.Create(ctx, &models.Account{
		Name:       "Banco",
		Type:       "checking",
		CurrencyID: currencyID,
	})
	if err != nil {
		t.Fatalf("crear cuenta: %v", err)
	}

	// Transacción SIN categoría (nil).
	if _, err := txSvc.Create(ctx, &models.Transaction{
		AccountID:  account.ID,
		CategoryID: nil,
		CurrencyID: currencyID,
		Date:       "2026-09-12",
		Payee:      "Desconocido",
		Outflow:    30,
	}); err != nil {
		t.Fatalf("crear transacción sin categoría: %v", err)
	}

	view, err := budgetSvc.MonthView(ctx, 2026, 9)
	if err != nil {
		t.Fatalf("vista: %v", err)
	}
	// El total de ingresos del mes no debe romperse; assigned se mantiene.
	if len(view.CurrencyTotals) == 0 {
		t.Fatal("se esperaban totales por moneda")
	}
	row := view.Groups[0].Categories[0]
	if row.Assigned[currencyID] != 100 {
		t.Errorf("assigned esperado 100, got %v", row.Assigned[currencyID])
	}
	if row.Activity[currencyID] != 0 {
		t.Errorf("actividad de categoría no debe incluir la transacción sin categoría (0), got %v", row.Activity[currencyID])
	}
}