package maintenance

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/db"
)

func newTestService(t *testing.T) (*Service, *sql.DB, int64, int64) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "home-maintenance.db")
	database, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("abrir db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	migrationsDir, err := filepath.Abs("../../../../migrations")
	if err != nil {
		t.Fatalf("ruta de migraciones: %v", err)
	}
	if err := db.Migrate(database, migrationsDir); err != nil {
		t.Fatalf("migrar: %v", err)
	}

	res, err := database.Exec(`INSERT INTO homes (name) VALUES ('Casa Demo')`)
	if err != nil {
		t.Fatalf("insertar vivienda: %v", err)
	}
	homeID, _ := res.LastInsertId()

	res, err = database.Exec(`INSERT INTO institutions (name) VALUES ('Proveedor Demo')`)
	if err != nil {
		t.Fatalf("insertar institución: %v", err)
	}
	institutionID, _ := res.LastInsertId()

	service := NewService(NewStorage(database))
	service.SetClock(func() time.Time { return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) })
	return service, database, homeID, institutionID
}

func currencyID(t *testing.T, database *sql.DB, code string) int64 {
	t.Helper()
	var id int64
	if err := database.QueryRow(`SELECT id FROM currencies WHERE code = ?`, code).Scan(&id); err != nil {
		t.Fatalf("moneda %s: %v", code, err)
	}
	return id
}

func TestCreateMaintenanceWithItemsAndTotals(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, institutionID := newTestService(t)
	usd := currencyID(t, database, "USD")

	record, err := service.Create(ctx, homeID, Input{
		PerformedAt:   "2026-03-15",
		Kind:          KindCorrective,
		Area:          AreaPlumbing,
		InstitutionID: &institutionID,
		Description:   "Cambio de llave de paso",
		CurrencyID:    usd,
	})
	if err != nil {
		t.Fatalf("crear mantenimiento: %v", err)
	}
	if record.InstitutionName != "Proveedor Demo" {
		t.Fatalf("institución = %q", record.InstitutionName)
	}

	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemLabor, Description: "Mano de obra", Amount: 40,
	}); err != nil {
		t.Fatalf("crear mano de obra: %v", err)
	}
	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Pintura blanca", Quantity: 2, Unit: UnitGallon,
		Amount: 25.55, InstitutionID: &institutionID,
	}); err != nil {
		t.Fatalf("crear insumo: %v", err)
	}

	detail, err := service.Get(ctx, homeID, record.ID)
	if err != nil {
		t.Fatalf("obtener detalle: %v", err)
	}
	if len(detail.Items) != 2 {
		t.Fatalf("líneas = %d, quería 2", len(detail.Items))
	}
	if detail.LaborCost != 40 {
		t.Fatalf("mano de obra = %v", detail.LaborCost)
	}
	if detail.SuppliesCost != 25.55 {
		t.Fatalf("insumos = %v", detail.SuppliesCost)
	}
	if detail.TotalCost != 65.55 {
		t.Fatalf("total = %v", detail.TotalCost)
	}

	list, err := service.List(ctx, homeID, Filter{})
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("registros = %d", len(list.Items))
	}
	if len(list.TotalsByCurrency) != 1 || list.TotalsByCurrency[0].Total != 65.55 {
		t.Fatalf("totales = %+v", list.TotalsByCurrency)
	}
	if list.TotalsByCurrency[0].CurrencyCode != "USD" {
		t.Fatalf("moneda = %q", list.TotalsByCurrency[0].CurrencyCode)
	}
}

func TestLaborRejectsInstitution(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, institutionID := newTestService(t)
	record, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-15", Kind: KindCorrective, Area: AreaOther,
		CurrencyID: currencyID(t, database, "USD"),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemLabor, Description: "Mano de obra", Amount: 10,
		InstitutionID: &institutionID,
	})
	if err == nil {
		t.Fatal("se esperaba error al asociar institución a mano de obra")
	}
	var validation *ValidationError
	if !asValidation(err, &validation) || validation.Code != "labor_institution" {
		t.Fatalf("error = %v", err)
	}
}

func TestFutureDateRejected(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, _ := newTestService(t)

	_, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-11-24", Kind: KindCorrective, Area: AreaOther,
		CurrencyID: currencyID(t, database, "USD"),
	})
	if err == nil {
		t.Fatal("se esperaba error con fecha futura")
	}
	var validation *ValidationError
	if !asValidation(err, &validation) || validation.Code != "future_date" {
		t.Fatalf("error = %v", err)
	}
}

func TestDeleteMaintenanceCascadesItems(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, _ := newTestService(t)
	record, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-15", Kind: KindPreventive, Area: AreaGarden,
		CurrencyID: currencyID(t, database, "USD"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Tierra", Quantity: 3, Unit: UnitSack, Amount: 15,
	}); err != nil {
		t.Fatal(err)
	}

	if err := service.Delete(ctx, homeID, record.ID); err != nil {
		t.Fatalf("eliminar: %v", err)
	}

	var count int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM home_maintenance_items WHERE maintenance_id = ?`, record.ID,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("líneas restantes = %d", count)
	}
}

func TestDeletingInstitutionKeepsHistory(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, institutionID := newTestService(t)
	record, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-15", Kind: KindCorrective, Area: AreaElectrical,
		InstitutionID: &institutionID,
		CurrencyID:    currencyID(t, database, "USD"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Cable", Quantity: 10, Unit: UnitMeter,
		Amount: 12, InstitutionID: &institutionID,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(`DELETE FROM institutions WHERE id = ?`, institutionID); err != nil {
		t.Fatalf("borrar institución: %v", err)
	}

	detail, err := service.Get(ctx, homeID, record.ID)
	if err != nil {
		t.Fatalf("detalle tras borrar institución: %v", err)
	}
	if detail.InstitutionID != nil {
		t.Fatalf("institution_id = %v, quería NULL", *detail.InstitutionID)
	}
	if detail.TotalCost != 12 {
		t.Fatalf("total = %v", detail.TotalCost)
	}
	if detail.Items[0].InstitutionID != nil {
		t.Fatal("la línea debería quedar sin institución")
	}
}

func TestItemOwnershipScopedToMaintenance(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, _ := newTestService(t)
	usd := currencyID(t, database, "USD")

	first, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-01-10", Kind: KindCorrective, Area: AreaOther, CurrencyID: usd,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-02-10", Kind: KindCorrective, Area: AreaOther, CurrencyID: usd,
	})
	if err != nil {
		t.Fatal(err)
	}
	item, err := service.CreateItem(ctx, homeID, second.ID, ItemInput{
		ItemType: ItemLabor, Description: "Trabajo", Amount: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := service.DeleteItem(ctx, homeID, first.ID, item.ID); err == nil {
		t.Fatal("no debería borrar una línea de otro mantenimiento")
	}
}

func TestQuantityMustBePositive(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, _ := newTestService(t)
	record, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-15", Kind: KindCorrective, Area: AreaOther,
		CurrencyID: currencyID(t, database, "USD"),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Cemento", Quantity: -1, Unit: UnitSack, Amount: 10,
	})
	if err == nil {
		t.Fatal("se esperaba error con cantidad negativa")
	}
}

func TestListFiltersByYearAndKind(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, _ := newTestService(t)
	usd := currencyID(t, database, "USD")

	first, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2025-05-01", Kind: KindCorrective, Area: AreaOther, CurrencyID: usd,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, first.ID, ItemInput{
		ItemType: ItemLabor, Description: "Trabajo", Amount: 30,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-05-01", Kind: KindPreventive, Area: AreaGarden, CurrencyID: usd,
	}); err != nil {
		t.Fatal(err)
	}

	byYear, err := service.List(ctx, homeID, Filter{Year: 2025})
	if err != nil {
		t.Fatal(err)
	}
	if len(byYear.Items) != 1 || byYear.Items[0].ID != first.ID {
		t.Fatalf("filtro año devolvió %d registros", len(byYear.Items))
	}
	if len(byYear.TotalsByCurrency) != 1 || byYear.TotalsByCurrency[0].Total != 30 {
		t.Fatalf("total filtrado = %+v", byYear.TotalsByCurrency)
	}

	byKind, err := service.List(ctx, homeID, Filter{Kind: KindPreventive})
	if err != nil {
		t.Fatal(err)
	}
	if len(byKind.Items) != 1 || byKind.Items[0].Kind != KindPreventive {
		t.Fatalf("filtro tipo devolvió %d registros", len(byKind.Items))
	}

	if _, err := service.List(ctx, homeID, Filter{Kind: "unknown"}); err == nil {
		t.Fatal("se esperaba error con tipo desconocido")
	}
}

func asValidation(err error, target **ValidationError) bool {
	return errors.As(err, target)
}
