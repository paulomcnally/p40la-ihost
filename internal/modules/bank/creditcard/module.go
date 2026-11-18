package creditcard

import (
	"database/sql"
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/modules"
)

// Module es el módulo core de tarjetas de crédito (bank.creditcard).
type Module struct {
	service *Service
}

// New construye el módulo con su storage y servicio.
func New(db *sql.DB, importers ImporterLookup) *Module {
	storage := NewStorage(db)
	return &Module{service: NewService(storage, importers)}
}

// Info devuelve los metadatos canónicos del módulo.
func (m *Module) Info() modules.Info {
	return modules.Info{
		ID:          ModuleID,
		Version:     ModuleVersion,
		Kind:        "core",
		Name:        "Tarjetas de crédito",
		Description: "Tarjetas, estados de cuenta y movimientos",
		Migrations:  []string{"0048_create_credit_cards"},
	}
}

// RegisterRoutes monta los endpoints del módulo.
func (m *Module) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	h := &handlers{service: m.service}

	mux.Handle("GET /api/banks/{bank_id}/credit-cards", auth(http.HandlerFunc(h.listCards)))
	mux.Handle("POST /api/banks/{bank_id}/credit-cards", auth(http.HandlerFunc(h.createCard)))
	mux.Handle("GET /api/credit-cards/{id}", auth(http.HandlerFunc(h.getCard)))
	mux.Handle("PUT /api/credit-cards/{id}", auth(http.HandlerFunc(h.updateCard)))
	mux.Handle("DELETE /api/credit-cards/{id}", auth(http.HandlerFunc(h.deleteCard)))

	mux.Handle("GET /api/credit-cards/{id}/statements", auth(http.HandlerFunc(h.listStatements)))
	mux.Handle("POST /api/credit-cards/{id}/statements", auth(http.HandlerFunc(h.createStatement)))
	mux.Handle("GET /api/credit-card-statements/{id}", auth(http.HandlerFunc(h.getStatement)))
	mux.Handle("PUT /api/credit-card-statements/{id}", auth(http.HandlerFunc(h.updateStatement)))
	mux.Handle("DELETE /api/credit-card-statements/{id}", auth(http.HandlerFunc(h.deleteStatement)))

	mux.Handle("GET /api/credit-card-statements/{id}/transactions", auth(http.HandlerFunc(h.listTransactions)))
	mux.Handle("POST /api/credit-card-statements/{id}/transactions", auth(http.HandlerFunc(h.createTransaction)))
	mux.Handle("PUT /api/credit-card-transactions/{id}", auth(http.HandlerFunc(h.updateTransaction)))
	mux.Handle("DELETE /api/credit-card-transactions/{id}", auth(http.HandlerFunc(h.deleteTransaction)))

	mux.Handle("POST /api/credit-cards/{id}/statements/import/preview", auth(http.HandlerFunc(h.previewImport)))
	mux.Handle("POST /api/credit-cards/{id}/statements/import", auth(http.HandlerFunc(h.confirmImport)))
}
