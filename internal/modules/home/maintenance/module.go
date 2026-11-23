package maintenance

import (
	"database/sql"
	"net/http"

	"github.com/paulomcnally/p40la-ihost/internal/modules"
)

// Module es el módulo core home.maintenance (ARCH-MODULES).
type Module struct {
	service *Service
}

// New construye el módulo con sus dependencias.
func New(db *sql.DB) *Module {
	return &Module{service: NewService(NewStorage(db))}
}

// Info devuelve los metadatos canónicos del módulo.
func (m *Module) Info() modules.Info {
	return modules.Info{
		ID:          ModuleID,
		Version:     ModuleVersion,
		Kind:        "core",
		Name:        "Reparaciones y mantenimientos del hogar",
		Description: "Histórico de reparaciones con insumos, mano de obra y proveedores",
		Migrations:  []string{"0057_home_maintenance"},
	}
}

// RegisterRoutes monta los endpoints del módulo.
func (m *Module) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	h := &handlers{service: m.service}
	h.register(mux, auth)
}

// Service expone el servicio para tests de integración.
func (m *Module) Service() *Service { return m.service }
