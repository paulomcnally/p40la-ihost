package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/mcp/config"
)

func newBudgetTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := New(config.Config{BaseURL: server.URL, APIKey: "demo-token", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}
	return client
}

func TestBudgetClientListCategoryGroups(t *testing.T) {
	var gotPath string
	accountID := int64(3)
	client := newBudgetTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]CategoryGroup{{
			ID:   2,
			Name: "Necesidades",
			Icon: "cart",
			Categories: []Category{{
				ID: 5, CategoryGroupID: 2, Name: "Comida", Icon: "food", AccountID: &accountID,
			}},
		}})
	})

	groups, err := client.ListCategoryGroups(context.Background())
	if err != nil {
		t.Fatalf("listar grupos: %v", err)
	}
	if gotPath != "/api/budget/category-groups" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(groups) != 1 || groups[0].Name != "Necesidades" || len(groups[0].Categories) != 1 {
		t.Fatalf("grupos inesperados: %+v", groups)
	}
	category := groups[0].Categories[0]
	if category.ID != 5 || category.Name != "Comida" || category.AccountID == nil || *category.AccountID != 3 {
		t.Fatalf("categoría inesperada: %+v", category)
	}
}

func TestBudgetClientListAccounts(t *testing.T) {
	client := newBudgetTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Account{{
			ID: 3, Name: "Cuenta Demo", Type: "checking", CurrencyID: 1,
			CurrencyCode: "NIO", StartingBalance: 1000, Balance: 850,
		}})
	})

	accounts, err := client.ListAccounts(context.Background())
	if err != nil {
		t.Fatalf("listar cuentas: %v", err)
	}
	if len(accounts) != 1 || accounts[0].CurrencyCode != "NIO" || accounts[0].Balance != 850 {
		t.Fatalf("cuentas inesperadas: %+v", accounts)
	}
}

func TestBudgetClientListTransactionsSendsPeriod(t *testing.T) {
	var gotPath, gotQuery string
	client := newBudgetTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Transaction{{
			ID: 10, AccountID: 3, CurrencyID: 1, Date: "2026-03-10", Payee: "Proveedor Demo", Outflow: 1200,
		}})
	})

	transactions, err := client.ListTransactions(context.Background(), 2026, 3)
	if err != nil {
		t.Fatalf("listar transacciones: %v", err)
	}
	if gotPath != "/api/budget/transactions" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotQuery != "month=3&year=2026" {
		t.Fatalf("query = %q", gotQuery)
	}
	if len(transactions) != 1 || transactions[0].Outflow != 1200 {
		t.Fatalf("transacciones inesperadas: %+v", transactions)
	}
}

func TestBudgetClientMonthView(t *testing.T) {
	var gotPath string
	client := newBudgetTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(MonthView{
			Year: 2026, Month: 3,
			CurrencyTotals: []MonthCurrencyTotal{{CurrencyID: 1, Code: "NIO", Symbol: "C$"}},
			Groups: []MonthGroup{{
				ID: 2, Name: "Necesidades",
				Categories: []MonthCategory{{
					Category:  Category{ID: 5, Name: "Comida"},
					Assigned:  map[int64]float64{1: 5000},
					Activity:  map[int64]float64{1: 1200},
					Available: map[int64]float64{1: 3800},
				}},
			}},
		})
	})

	view, err := client.MonthView(context.Background(), 2026, 3)
	if err != nil {
		t.Fatalf("vista mensual: %v", err)
	}
	if gotPath != "/api/budget/months/2026/3" {
		t.Fatalf("path = %q", gotPath)
	}
	if view.Year != 2026 || len(view.Groups) != 1 || len(view.Groups[0].Categories) != 1 {
		t.Fatalf("vista inesperada: %+v", view)
	}
	row := view.Groups[0].Categories[0]
	if row.Assigned[1] != 5000 || row.Activity[1] != 1200 || row.Available[1] != 3800 {
		t.Fatalf("mapas inesperados: assigned=%v activity=%v available=%v", row.Assigned, row.Activity, row.Available)
	}
}

func TestBudgetClientCreateTransactionSendsSource(t *testing.T) {
	var gotSource, gotPath string
	var gotBody TransactionRequest
	client := newBudgetTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotSource = r.Header.Get("X-P40LA-Source")
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decodificar body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Transaction{
			ID: 10, AccountID: 3, AccountName: "Cuenta Demo", CurrencyID: 1, CurrencyCode: "NIO",
			Date: "2026-03-10", Payee: "Proveedor Demo", Outflow: 1200,
		})
	})

	categoryID := int64(5)
	created, err := client.CreateTransaction(context.Background(), TransactionRequest{
		AccountID:  3,
		CategoryID: &categoryID,
		CurrencyID: 1,
		Date:       "2026-03-10",
		Payee:      "Proveedor Demo",
		Outflow:    1200,
	})
	if err != nil {
		t.Fatalf("crear transacción: %v", err)
	}
	if gotSource != "mcp" {
		t.Fatalf("source = %q, quiero mcp", gotSource)
	}
	if gotPath != "/api/budget/transactions" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody.AccountID != 3 || gotBody.CategoryID == nil || *gotBody.CategoryID != 5 ||
		gotBody.CurrencyID != 1 || gotBody.Outflow != 1200 || gotBody.Inflow != 0 {
		t.Fatalf("body inesperado: %+v", gotBody)
	}
	if created.ID != 10 || created.AccountName != "Cuenta Demo" || created.CategoryName != "" {
		t.Fatalf("transacción creada inesperada: %+v", created)
	}
}
