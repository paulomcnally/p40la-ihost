package portfolio

import (
	"database/sql"
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/modules"
)

// Module es el módulo core realestate.portfolio (ARCH-MODULES).
type Module struct {
	service *Service
}

// New construye el módulo con la DB y el proveedor de estados de viviendas.
func New(db *sql.DB, states modules.HomeStateProvider) *Module {
	return &Module{service: NewService(NewStorage(db), states)}
}

// Info devuelve los metadatos canónicos del módulo.
func (m *Module) Info() modules.Info {
	return modules.Info{
		ID:          ModuleID,
		Version:     ModuleVersion,
		Kind:        "core",
		Name:        "Resumen de inmuebles",
		Description: "Ingresos, gastos y pendientes por vivienda y moneda",
		Migrations: []string{
			"0058_homes_usage",
			"0059_portfolio_indexes",
		},
	}
}

// RegisterRoutes monta los endpoints del módulo.
func (m *Module) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	h := &handlers{service: m.service}
	h.register(mux, auth)
}

// Service expone el servicio para tests de integración.
func (m *Module) Service() *Service { return m.service }
