package services

import (
	"context"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// TestCategorySuggestedAssignment verifica la lógica de SPEC-098: la sugerencia
// de asignación de una categoría es la suma de la factura más reciente por cada
// servicio vinculado, agrupada por moneda.
func TestCategorySuggestedAssignment(t *testing.T) {
	ctx := context.Background()
	database, err := db.OpenDB(":memory:", "../../migrations")
	if err != nil {
		t.Fatalf("abrir db de prueba: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	serviceStorage := storage.NewServiceStorage(database)
	categoryStorage := storage.NewCategoryStorage(database)
	accountStorage := storage.NewAccountStorage(database)
	currencyStorage := storage.NewCurrencyStorage(database)
	groupStorage := storage.NewCategoryGroupStorage(database)
	billStorage := storage.NewBillStorage(database)

	currencies, err := currencyStorage.List(ctx)
	if err != nil || len(currencies) == 0 {
		t.Fatalf("obtener monedas: %v", err)
	}
	currencyID := currencies[0].ID

	catSvc := NewCategoryService(categoryStorage, groupStorage)
	catSvc.SetServiceStorage(serviceStorage)
	catSvc.SetBillStorage(billStorage)

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("insertar dato de prueba (%q): %v", query, err)
		}
	}
	mustExec("INSERT INTO homes (name) VALUES ('Casa')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Claro Internet Móvil', 'Claro', ?, 'monthly', 100, 1, 'internet', 'fixed', 1)`, currencyID)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Tigo Internet Móvil', 'Tigo', ?, 'monthly', 100, 1, 'internet', 'fixed', 1)`, currencyID)
	mustExec("INSERT INTO category_groups (name, icon) VALUES ('Servicios', 'home')")

	account, err := accountStorage.Create(ctx, &models.Account{Name: "Efectivo", Type: "cash", CurrencyID: currencyID})
	if err != nil {
		t.Fatalf("crear cuenta: %v", err)
	}

	// Categoría sin servicios → no aplica.
	noServices, err := catSvc.Create(ctx, &models.Category{
		CategoryGroupID: 1,
		Name:            "Sin servicios",
		Icon:            "tag",
	})
	if err != nil {
		t.Fatalf("crear categoría sin servicios: %v", err)
	}
	res, err := catSvc.SuggestedAssignment(ctx, noServices.ID)
	if err != nil {
		t.Fatalf("SuggestedAssignment sin servicios: %v", err)
	}
	if res.Applies {
		t.Error("sin servicios debería applies=false")
	}

	// Categoría con 2 servicios pero sin facturas → no aplica.
	noBills, err := catSvc.Create(ctx, &models.Category{
		CategoryGroupID: 1,
		Name:            "Internet Móvil",
		Icon:            "internet",
		ServiceIDs:      []int64{1, 2},
		AccountID:       &account.ID,
	})
	if err != nil {
		t.Fatalf("crear categoría multi-servicio: %v", err)
	}
	res, err = catSvc.SuggestedAssignment(ctx, noBills.ID)
	if err != nil {
		t.Fatalf("SuggestedAssignment sin facturas: %v", err)
	}
	if res.Applies {
		t.Error("sin facturas debería applies=false")
	}

	// Agregar facturas: la más reciente por servicio.
	// Servicio 1 (Claro): factura de 2026-08 por 1000 y de 2026-09 por 1200 → 1200.
	// Servicio 2 (Tigo): factura de 2026-09 por 800 → 800.
	// Total sugerido = 2000 en la moneda.
	for _, b := range []models.Bill{
		{ServiceID: 1, Year: 2026, Month: 8, Amount: 1000, InvoiceNumber: "C-1", Status: "paid"},
		{ServiceID: 1, Year: 2026, Month: 9, Amount: 1200, InvoiceNumber: "C-2", Status: "paid"},
		{ServiceID: 2, Year: 2026, Month: 9, Amount: 800, InvoiceNumber: "T-1", Status: "pending"},
	} {
		if _, err := billStorage.Create(ctx, &b); err != nil {
			t.Fatalf("crear factura: %v", err)
		}
	}

	res, err = catSvc.SuggestedAssignment(ctx, noBills.ID)
	if err != nil {
		t.Fatalf("SuggestedAssignment con facturas: %v", err)
	}
	if !res.Applies {
		t.Error("con facturas debería applies=true")
	}
	if got := res.Suggested[currencyID]; got != 2000 {
		t.Errorf("sugerencia esperada 2000, got %v", got)
	}
	if len(res.SourceServiceNames) != 2 {
		t.Errorf("esperaba 2 nombres de servicio, got %d", len(res.SourceServiceNames))
	}
}
