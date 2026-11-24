package rental

import (
	"context"
	"database/sql"
	"net/http"
	"sync"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/modules"
	"github.com/paulomcnally/p40la-ihost/internal/services"
)

// Module es el módulo core realestate.rental (ARCH-MODULES).
type Module struct {
	service   *Service
	scheduler *RentalScheduler
	startOnce sync.Once
	stopOnce  sync.Once
}

// New construye el módulo con sus dependencias.
func New(db *sql.DB, email *services.EmailService, settings *services.SystemSettingsService, dataDir string) *Module {
	service := NewService(NewStorage(db), email, settings, dataDir)
	return &Module{service: service, scheduler: NewRentalScheduler(service, 15*time.Minute)}
}

// Info devuelve los metadatos canónicos del módulo.
func (m *Module) Info() modules.Info {
	return modules.Info{
		ID:          ModuleID,
		Version:     ModuleVersion,
		Kind:        "core",
		Name:        "Alquiler de viviendas",
		Description: "Apartados, contratos, cobros en PDF, comprobantes y recordatorios",
		Migrations: []string{
			"0051_rental_leases",
			"0052_rental_reservations",
			"0053_rental_billing",
			"0054_rental_reminders",
			"0055_real_estate_category",
			"0056_rental_commission_type",
			"0058_homes_usage",
		},
	}
}

// RegisterRoutes monta los endpoints del módulo.
func (m *Module) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	h := &handlers{service: m.service}
	h.register(mux, auth)
}

// Start arranca el scheduler (idempotente).
func (m *Module) Start() {
	m.startOnce.Do(func() { m.scheduler.Start() })
}

// Stop detiene el scheduler (idempotente).
func (m *Module) Stop() {
	m.stopOnce.Do(func() { m.scheduler.Stop() })
}

// Service expone el servicio para tests de integración.
func (m *Module) Service() *Service { return m.service }

// HomeStates implementa modules.HomeStateProvider (SPEC-143).
func (m *Module) HomeStates(ctx context.Context) ([]modules.HomeState, error) {
	return m.service.ListHomeStates(ctx)
}

// HasActiveRental expone si la vivienda tiene alquiler activo (SPEC-143).
func (m *Module) HasActiveRental(ctx context.Context, homeID int64) (bool, error) {
	return m.service.HasActiveRental(ctx, homeID)
}
