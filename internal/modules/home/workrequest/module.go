package workrequest

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/modules"
	homemaintenance "github.com/paulomcnally/p40la-ihost/internal/modules/home/maintenance"
)

// Module es el módulo core home.workrequest (ARCH-MODULES).
type Module struct {
	service *Service
}

// New construye el módulo con la DB, el contrato de reparaciones, el
// directorio de datos para la evidencia y la configuración de links de carga.
func New(db *sql.DB, maintenance *homemaintenance.Service, dataDir string, upload UploadConfig) *Module {
	service := NewService(NewStorage(db), maintenance, dataDir)
	service.SetUploadConfig(upload)
	return &Module{service: service}
}

// Info devuelve los metadatos canónicos del módulo.
func (m *Module) Info() modules.Info {
	return modules.Info{
		ID:          ModuleID,
		Version:     ModuleVersion,
		Kind:        "core",
		Name:        "Reportes de viviendas",
		Description: "Problemas y trabajos deseados con evidencia y conversión a reparación",
		Migrations:  []string{"0062_home_work_requests", "0063_work_request_evidence_idempotency"},
	}
}

// RegisterRoutes monta los endpoints del módulo.
func (m *Module) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	h := &handlers{service: m.service, limiter: newUploadRateLimiter(30, time.Minute)}
	h.register(mux, auth)
}

// Service expone el servicio para tests de integración.
func (m *Module) Service() *Service { return m.service }
