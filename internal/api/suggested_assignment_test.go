package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/services"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// TestSuggestedAssignmentEndpoint verifica el endpoint de SPEC-098:
// GET /api/budget/categories/{id}/suggested-assignment devuelve la suma de la
// factura más reciente por cada servicio vinculado, agrupada por moneda.
func TestSuggestedAssignmentEndpoint(t *testing.T) {
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

	catSvc := services.NewCategoryService(categoryStorage, groupStorage)
	catSvc.SetServiceStorage(serviceStorage)
	catSvc.SetBillStorage(billStorage)

	// BudgetHandlers solo necesita el CategoryService para este endpoint.
	handlers := NewBudgetHandlers(nil, nil, catSvc, nil, nil, nil)

	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("insertar dato de prueba (%q): %v", query, err)
		}
	}
	mustExec("INSERT INTO homes (name) VALUES ('Casa')")
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, is_recurring)
		VALUES (1, 'Internet', 'Claro', ?, 'monthly', 100, 1, 'internet', 'fixed', 1)`, currencyID)
	mustExec("INSERT INTO category_groups (name, icon) VALUES ('Servicios', 'home')")

	account, err := accountStorage.Create(ctx, &models.Account{Name: "Efectivo", Type: "cash", CurrencyID: currencyID})
	if err != nil {
		t.Fatalf("crear cuenta: %v", err)
	}

	cat, err := catSvc.Create(ctx, &models.Category{
		CategoryGroupID: 1,
		Name:            "Internet",
		Icon:            "wifi",
		ServiceIDs:      []int64{1},
		AccountID:       &account.ID,
	})
	if err != nil {
		t.Fatalf("crear categoría: %v", err)
	}

	// Sin facturas → no aplica.
	req := httptest.NewRequest(http.MethodGet, "/api/budget/categories/1/suggested-assignment", nil)
	req.SetPathValue("id", strconv.FormatInt(cat.ID, 10))
	rr := httptest.NewRecorder()
	handlers.SuggestedAssignment(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("sin facturas esperaba 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Crear dos facturas del servicio; la sugerencia toma la más reciente.
	billStorage.Create(ctx, &models.Bill{ServiceID: 1, Year: 2026, Month: 8, Amount: 1000, InvoiceNumber: "INV-1", Status: "pending"})
	billStorage.Create(ctx, &models.Bill{ServiceID: 1, Year: 2026, Month: 9, Amount: 1200, InvoiceNumber: "INV-2", Status: "pending"})

	req = httptest.NewRequest(http.MethodGet, "/api/budget/categories/1/suggested-assignment", nil)
	req.SetPathValue("id", strconv.FormatInt(cat.ID, 10))
	rr = httptest.NewRecorder()
	handlers.SuggestedAssignment(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("con facturas esperaba 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var res struct {
		Suggested          map[int64]float64 `json:"suggested"`
		Applies            bool              `json:"applies"`
		SourceServiceNames []string          `json:"source_service_names"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("decodificar respuesta: %v", err)
	}
	if !res.Applies {
		t.Error("applies esperado true")
	}
	if res.Suggested[currencyID] != 1200 {
		t.Errorf("sugerencia esperada 1200, got %v", res.Suggested[currencyID])
	}
	if len(res.SourceServiceNames) != 1 || res.SourceServiceNames[0] != "Internet" {
		t.Errorf("source_service_names esperado [Internet], got %v", res.SourceServiceNames)
	}
}
