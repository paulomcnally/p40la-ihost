package services

import (
	"context"
	"strings"
	"testing"

	"github.com/paulomcnally/p40la-ihost/internal/db"
	"github.com/paulomcnally/p40la-ihost/internal/models"
	"github.com/paulomcnally/p40la-ihost/internal/storage"
)

// newAlertTestDB abre una DB en memoria con migraciones y datos de prueba.
// Datos creados:
//   - home id=1
//   - auto 1: sin seguro → alerta no_insurance
//   - auto 2: seguro vencido (2025-01-01) sin reemplazo → alerta expired
//   - auto 3: seguro vencido PERO con otro activo → NO alerta
//   - auto 4: seguro activo (sin end_date) → NO alerta
func newAlertTestDB(t *testing.T) (autoStorage *storage.AutoStorage) {
	t.Helper()
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

	mustExec("INSERT INTO homes (name) VALUES ('Casa de prueba')")
	mustExec("INSERT INTO autos (year, model, brand, color, icon, placa) VALUES (2020, 'Corolla', 'Toyota', 'Rojo', 'car', 'ABC-123')")
	mustExec("INSERT INTO autos (year, model, brand, color, icon, placa) VALUES (2019, 'Civic', 'Honda', 'Azul', 'car', 'XYZ-789')")
	mustExec("INSERT INTO autos (year, model, brand, color, icon, placa) VALUES (2021, 'Accord', 'Honda', 'Negro', 'car', 'QWE-456')")
	mustExec("INSERT INTO autos (year, model, brand, color, icon, placa) VALUES (2018, 'Focus', 'Ford', 'Blanco', 'car', 'ASD-321')")

	// Servicios: se asocian a home=1 y currency=1 (NIO).
	// id 1: seguro de auto 2, vencido
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, auto_generate, start_date, end_date, is_recurring)
		VALUES (1, 'Seguro Honda Civic', '', 1, 'yearly', 300, 1, 'insurance_car', 'fixed', 0, '2024-01-01', '2025-01-01', 1)`)
	// id 2: seguro de auto 3, vencido
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, auto_generate, start_date, end_date, is_recurring)
		VALUES (1, 'Seguro viejo Accord', '', 1, 'yearly', 300, 1, 'insurance_car', 'fixed', 0, '2024-01-01', '2025-01-01', 1)`)
	// id 3: seguro de auto 3, activo (sin end_date)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, auto_generate, start_date, end_date, is_recurring)
		VALUES (1, 'Seguro nuevo Accord', '', 1, 'yearly', 350, 1, 'insurance_car', 'fixed', 0, '2025-02-01', NULL, 1)`)
	// id 4: seguro de auto 4, activo (sin end_date)
	mustExec(`INSERT INTO services (home_id, name, institution, currency_id, frequency, suggested_amount, active, icon_key, billing_type, auto_generate, start_date, end_date, is_recurring)
		VALUES (1, 'Seguro Ford Focus', '', 1, 'yearly', 250, 1, 'insurance_car', 'fixed', 0, '2025-01-01', NULL, 1)`)

	// auto_services: auto 2→svc1 (vencido), auto 3→svc2 (vencido)+svc3 (activo), auto 4→svc4 (activo)
	mustExec("INSERT INTO auto_services (auto_id, service_id, coverage_type) VALUES (2, 1, 'full_cover')")
	mustExec("INSERT INTO auto_services (auto_id, service_id, coverage_type) VALUES (3, 2, 'full_cover')")
	mustExec("INSERT INTO auto_services (auto_id, service_id, coverage_type) VALUES (3, 3, 'full_cover')")
	mustExec("INSERT INTO auto_services (auto_id, service_id, coverage_type) VALUES (4, 4, 'full_cover')")

	return storage.NewAutoStorage(database)
}

func TestListWithoutInsurance(t *testing.T) {
	autoStorage := newAlertTestDB(t)
	ctx := context.Background()

	alerts, err := autoStorage.ListWithoutInsurance(ctx)
	if err != nil {
		t.Fatalf("ListWithoutInsurance: %v", err)
	}

	if len(alerts) != 1 {
		t.Fatalf("se esperaba 1 auto sin seguro, got %d (%+v)", len(alerts), alerts)
	}
	if alerts[0].AlertType != models.AlertTypeNoInsurance {
		t.Errorf("alert type esperado no_insurance, got %s", alerts[0].AlertType)
	}
	if alerts[0].AutoID != 1 || alerts[0].Placa != "ABC-123" {
		t.Errorf("auto sin seguro incorrecto: %+v", alerts[0])
	}
}

func TestListWithExpiredInsurance(t *testing.T) {
	autoStorage := newAlertTestDB(t)
	ctx := context.Background()

	alerts, err := autoStorage.ListWithExpiredInsurance(ctx)
	if err != nil {
		t.Fatalf("ListWithExpiredInsurance: %v", err)
	}

	// Solo auto 2 debe aparecer (seguro vencido sin reemplazo activo).
	// Auto 3 NO debe aparecer (tiene seguro vencido pero también uno activo).
	if len(alerts) != 1 {
		t.Fatalf("se esperaba 1 auto con seguro vencido, got %d (%+v)", len(alerts), alerts)
	}
	if alerts[0].AutoID != 2 {
		t.Errorf("auto con seguro vencido incorrecto: %+v", alerts[0])
	}
	if alerts[0].AlertType != models.AlertTypeExpired {
		t.Errorf("alert type esperado expired, got %s", alerts[0].AlertType)
	}
	if alerts[0].EndDate != "2025-01-01" {
		t.Errorf("end_date esperado 2025-01-01, got %s", alerts[0].EndDate)
	}
}

func TestRenderAlertsContent(t *testing.T) {
	alerts := []models.AutoAlert{
		{AutoID: 1, Year: 2020, Model: "Corolla", Brand: "Toyota", Placa: "ABC-123", AlertType: models.AlertTypeNoInsurance},
		{AutoID: 2, Year: 2019, Model: "Civic", Brand: "Honda", Placa: "XYZ-789", AlertType: models.AlertTypeExpired, EndDate: "2025-01-01"},
	}

	html := renderAlertsContent(alerts)

	if !strings.Contains(html, "Toyota Corolla 2020") {
		t.Errorf("vehículo 1 no aparece en HTML")
	}
	if !strings.Contains(html, "ABC-123") {
		t.Errorf("placa 1 no aparece en HTML")
	}
	if !strings.Contains(html, "Sin seguro asociado") {
		t.Errorf("motivo sin seguro no aparece")
	}
	if !strings.Contains(html, "Honda Civic 2019") {
		t.Errorf("vehículo 2 no aparece en HTML")
	}
	if !strings.Contains(html, "Seguro vencido el 01/01/2025") {
		t.Errorf("motivo vencido no aparece en HTML")
	}
	if !strings.Contains(html, "<table") || !strings.Contains(html, "</table>") {
		t.Errorf("no se generó tabla HTML")
	}
}

func TestFormatDate(t *testing.T) {
	cases := []struct{ in, want string }{
		{"2025-01-01", "01/01/2025"},
		{"2026-08-16", "16/08/2026"},
		{"2025-1-1", "2025-1-1"}, // formato no estándar, devuelve tal cual
		{"", ""},
	}
	for _, c := range cases {
		if got := formatDate(c.in); got != c.want {
			t.Errorf("formatDate(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
