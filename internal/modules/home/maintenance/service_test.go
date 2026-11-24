package maintenance

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
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

func totalFor(totals []CurrencyTotal, code string) (CurrencyTotal, bool) {
	for _, total := range totals {
		if total.CurrencyCode == code {
			return total, true
		}
	}
	return CurrencyTotal{}, false
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
	})
	if err != nil {
		t.Fatalf("crear mantenimiento: %v", err)
	}
	if record.InstitutionName != "Proveedor Demo" {
		t.Fatalf("institución = %q", record.InstitutionName)
	}

	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemLabor, Description: "Mano de obra", Amount: 40, CurrencyID: usd,
	}); err != nil {
		t.Fatalf("crear mano de obra: %v", err)
	}
	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Pintura blanca", Quantity: 2, Unit: UnitGallon,
		Amount: 25.55, CurrencyID: usd, InstitutionID: &institutionID,
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
	if len(detail.TotalsByCurrency) != 1 {
		t.Fatalf("monedas en el detalle = %d", len(detail.TotalsByCurrency))
	}
	usdTotal, ok := totalFor(detail.TotalsByCurrency, "USD")
	if !ok {
		t.Fatal("falta el total en USD")
	}
	if usdTotal.LaborCost != 40 || usdTotal.SuppliesCost != 25.55 || usdTotal.Total != 65.55 {
		t.Fatalf("totales USD = %+v", usdTotal)
	}

	list, err := service.List(ctx, homeID, Filter{})
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("registros = %d", len(list.Items))
	}
	if len(list.Items[0].TotalsByCurrency) != 1 {
		t.Fatalf("totales por registro = %+v", list.Items[0].TotalsByCurrency)
	}
	if len(list.TotalsByCurrency) != 1 || list.TotalsByCurrency[0].Total != 65.55 {
		t.Fatalf("totales = %+v", list.TotalsByCurrency)
	}
}

func TestMultiCurrencyRepair(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, _ := newTestService(t)
	usd := currencyID(t, database, "USD")
	nio := currencyID(t, database, "NIO")

	record, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-08-10", Kind: KindCorrective, Area: AreaRoof,
		Description: "Techo con mano de obra local e insumo importado",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemLabor, Description: "Instalación", Amount: 500, CurrencyID: nio,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Lámina", Quantity: 3, Unit: UnitUnit,
		Amount: 900, CurrencyID: nio,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Tornillería importada", Amount: 12.5, CurrencyID: usd,
	}); err != nil {
		t.Fatal(err)
	}

	detail, err := service.Get(ctx, homeID, record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.TotalsByCurrency) != 2 {
		t.Fatalf("monedas = %d, quería 2", len(detail.TotalsByCurrency))
	}
	nioTotal, _ := totalFor(detail.TotalsByCurrency, "NIO")
	if nioTotal.LaborCost != 500 || nioTotal.SuppliesCost != 900 || nioTotal.Total != 1400 {
		t.Fatalf("totales NIO = %+v", nioTotal)
	}
	usdTotal, _ := totalFor(detail.TotalsByCurrency, "USD")
	if usdTotal.LaborCost != 0 || usdTotal.SuppliesCost != 12.5 || usdTotal.Total != 12.5 {
		t.Fatalf("totales USD = %+v", usdTotal)
	}

	list, err := service.List(ctx, homeID, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.TotalsByCurrency) != 2 {
		t.Fatalf("totales del histórico = %+v", list.TotalsByCurrency)
	}
}

func TestItemRequiresCurrency(t *testing.T) {
	ctx := context.Background()
	service, _, homeID, _ := newTestService(t)
	record, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-15", Kind: KindCorrective, Area: AreaOther,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemLabor, Description: "Mano de obra", Amount: 10,
	})
	if err == nil {
		t.Fatal("se esperaba error sin moneda")
	}
	var validation *ValidationError
	if !asValidation(err, &validation) || validation.Code != "currency_required" {
		t.Fatalf("error = %v", err)
	}
}

func TestLaborAcceptsInstitution(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, institutionID := newTestService(t)
	record, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-15", Kind: KindCorrective, Area: AreaOther,
	})
	if err != nil {
		t.Fatal(err)
	}

	item, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemLabor, Description: "Mano de obra", Amount: 10,
		CurrencyID: currencyID(t, database, "USD"), InstitutionID: &institutionID,
	})
	if err != nil {
		t.Fatalf("mano de obra con institución: %v", err)
	}
	if item.InstitutionID == nil || *item.InstitutionID != institutionID {
		t.Fatalf("institution_id = %v, quería %d", item.InstitutionID, institutionID)
	}
	if item.Quantity != 1 || item.Unit != "unit" {
		t.Fatalf("cantidad/unidad = %v/%v, quería 1/unit", item.Quantity, item.Unit)
	}
}

func TestLaborRejectsMissingInstitution(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, institutionID := newTestService(t)
	record, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-15", Kind: KindCorrective, Area: AreaOther,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM institutions WHERE id = ?", institutionID); err != nil {
		t.Fatalf("borrar institución: %v", err)
	}

	_, err = service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemLabor, Description: "Mano de obra", Amount: 10,
		CurrencyID: currencyID(t, database, "USD"), InstitutionID: &institutionID,
	})
	if !errors.Is(err, ErrInstitutionNotFound) {
		t.Fatalf("error = %v, quería ErrInstitutionNotFound", err)
	}
}

func TestFutureDateRejected(t *testing.T) {
	ctx := context.Background()
	service, _, homeID, _ := newTestService(t)

	_, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-11-24", Kind: KindCorrective, Area: AreaOther,
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
	usd := currencyID(t, database, "USD")
	record, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-15", Kind: KindPreventive, Area: AreaGarden,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Tierra", Quantity: 3, Unit: UnitSack,
		Amount: 15, CurrencyID: usd,
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
	usd := currencyID(t, database, "USD")
	record, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-15", Kind: KindCorrective, Area: AreaElectrical,
		InstitutionID: &institutionID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Cable", Quantity: 10, Unit: UnitMeter,
		Amount: 12, CurrencyID: usd, InstitutionID: &institutionID,
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
	if len(detail.TotalsByCurrency) != 1 || detail.TotalsByCurrency[0].Total != 12 {
		t.Fatalf("totales = %+v", detail.TotalsByCurrency)
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
		PerformedAt: "2026-01-10", Kind: KindCorrective, Area: AreaOther,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-02-10", Kind: KindCorrective, Area: AreaOther,
	})
	if err != nil {
		t.Fatal(err)
	}
	item, err := service.CreateItem(ctx, homeID, second.ID, ItemInput{
		ItemType: ItemLabor, Description: "Trabajo", Amount: 5, CurrencyID: usd,
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
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Cemento", Quantity: -1, Unit: UnitSack,
		Amount: 10, CurrencyID: currencyID(t, database, "USD"),
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
		PerformedAt: "2025-05-01", Kind: KindCorrective, Area: AreaOther,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, first.ID, ItemInput{
		ItemType: ItemLabor, Description: "Trabajo", Amount: 30, CurrencyID: usd,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-05-01", Kind: KindPreventive, Area: AreaGarden,
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

func TestCreateWithItemPersistsBothRows(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, institutionID := newTestService(t)
	usd := currencyID(t, database, "USD")

	record, err := service.CreateWithItem(ctx, homeID, Input{
		PerformedAt:   "2026-03-20",
		Kind:          KindCorrective,
		Area:          AreaPlumbing,
		InstitutionID: &institutionID,
		Description:   "Reparación demo",
	}, &ItemInput{
		ItemType:      ItemLabor,
		Description:   "Mano de obra demo",
		Amount:        50,
		CurrencyID:    usd,
		InstitutionID: &institutionID,
	})
	if err != nil {
		t.Fatalf("crear con línea: %v", err)
	}
	if len(record.Items) != 1 || record.Items[0].ItemType != ItemLabor {
		t.Fatalf("líneas = %+v", record.Items)
	}
	if record.Items[0].Quantity != 1 || record.Items[0].Unit != UnitUnit {
		t.Fatalf("mano de obra debía normalizar cantidad/unidad: %+v", record.Items[0])
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM home_maintenances`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("mantenimientos = %d, quería 1", count)
	}

	// Un fallo dentro de la transacción (FK de moneda) no debe dejar el
	// mantenimiento huérfano.
	if _, err := service.storage.CreateWithItem(ctx, homeID, Input{
		PerformedAt: "2026-03-21",
		Kind:        KindCorrective,
		Area:        AreaOther,
	}, ItemInput{
		ItemType: ItemSupplies, Description: "Insumo demo", Quantity: 1, Unit: UnitUnit,
		Amount: 10, CurrencyID: 999999,
	}); err == nil {
		t.Fatal("se esperaba error de FK en la línea")
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM home_maintenances`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("la transacción dejó un mantenimiento vacío: %d", count)
	}
}

func TestListFiltersByDateItemTypeAndQuery(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, _ := newTestService(t)
	usd := currencyID(t, database, "USD")

	first, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-10", Kind: KindCorrective, Area: AreaPlumbing, Description: "Cambio de llave demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, first.ID, ItemInput{
		ItemType: ItemLabor, Description: "Mano de obra demo", Amount: 40, CurrencyID: usd,
	}); err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-11", Kind: KindCorrective, Area: AreaElectrical, Description: "Instalación demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, second.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Cable demo", Quantity: 3, Unit: UnitMeter, Amount: 30, CurrencyID: usd,
	}); err != nil {
		t.Fatal(err)
	}
	third, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-10", Kind: KindCorrective, Area: AreaGarden, Description: "Tierra demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, third.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Tierra demo", Quantity: 1, Unit: UnitSack, Amount: 12, CurrencyID: usd,
	}); err != nil {
		t.Fatal(err)
	}

	byDate, err := service.List(ctx, homeID, Filter{Date: "2026-03-10", IncludeItems: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(byDate.Items) != 2 {
		t.Fatalf("fecha devolvió %d registros", len(byDate.Items))
	}
	for _, item := range byDate.Items {
		if len(item.Items) != 1 {
			t.Fatalf("include_items no cargó la línea de %d", item.ID)
		}
	}
	if byDate.Truncated {
		t.Fatal("no debía truncar")
	}

	byType, err := service.List(ctx, homeID, Filter{ItemType: ItemLabor})
	if err != nil {
		t.Fatal(err)
	}
	if len(byType.Items) != 1 || byType.Items[0].ID != first.ID {
		t.Fatalf("item_type labor devolvió %d registros", len(byType.Items))
	}

	byQuery, err := service.List(ctx, homeID, Filter{Query: "cable"})
	if err != nil {
		t.Fatal(err)
	}
	if len(byQuery.Items) != 1 || byQuery.Items[0].ID != second.ID {
		t.Fatalf("query devolvió %d registros", len(byQuery.Items))
	}

	var validation *ValidationError
	if _, err := service.List(ctx, homeID, Filter{Date: "2026-03-10", Year: 2026}); !asValidation(err, &validation) || validation.Code != "invalid_filter" {
		t.Fatalf("date+year debía ser invalid_filter: %v", err)
	}
	if _, err := service.List(ctx, homeID, Filter{ItemType: "otro"}); !asValidation(err, &validation) || validation.Code != "invalid_item_type" {
		t.Fatalf("item_type inválido debía fallar: %v", err)
	}
	if _, err := service.List(ctx, homeID, Filter{Query: strings.Repeat("x", 101)}); !asValidation(err, &validation) || validation.Code != "invalid_query" {
		t.Fatalf("query larga debía fallar: %v", err)
	}
}

func TestTotalsKeepBothColumnsWithItemTypeFilter(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, _ := newTestService(t)
	usd := currencyID(t, database, "USD")

	record, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-10", Kind: KindCorrective, Area: AreaPlumbing, Description: "Reparación demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemLabor, Description: "Mano de obra demo", Amount: 40, CurrencyID: usd,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, record.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Insumo demo", Quantity: 2, Unit: UnitUnit, Amount: 25.55, CurrencyID: usd,
	}); err != nil {
		t.Fatal(err)
	}

	list, err := service.List(ctx, homeID, Filter{ItemType: ItemLabor})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("mantenimientos = %d", len(list.Items))
	}
	total, ok := totalFor(list.TotalsByCurrency, "USD")
	if !ok {
		t.Fatal("falta total USD")
	}
	if total.LaborCost != 40 || total.SuppliesCost != 25.55 || total.Total != 65.55 {
		t.Fatalf("los totales debían conservar ambas columnas: %+v", total)
	}
}

func TestListItemsByMaintenanceIDsTruncates(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, _ := newTestService(t)
	usd := currencyID(t, database, "USD")

	first, err := service.Create(ctx, homeID, Input{PerformedAt: "2026-03-10", Kind: KindCorrective, Area: AreaPlumbing})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(ctx, homeID, Input{PerformedAt: "2026-03-10", Kind: KindCorrective, Area: AreaGarden})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{first.ID, second.ID} {
		if _, err := service.CreateItem(ctx, homeID, id, ItemInput{
			ItemType: ItemLabor, Description: "Mano de obra demo", Amount: 10, CurrencyID: usd,
		}); err != nil {
			t.Fatal(err)
		}
	}

	grouped, truncated, err := service.storage.ListItemsByMaintenanceIDs(ctx, []int64{first.ID, second.ID}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated || len(grouped) != 1 {
		t.Fatalf("esperaba truncado con una línea: %v %v", grouped, truncated)
	}

	grouped, truncated, err = service.storage.ListItemsByMaintenanceIDs(ctx, []int64{first.ID, second.ID}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if truncated || len(grouped) != 2 {
		t.Fatalf("esperaba dos grupos completos: %v %v", grouped, truncated)
	}
}

func TestQueryEscapesLikeWildcards(t *testing.T) {
	ctx := context.Background()
	service, database, homeID, _ := newTestService(t)
	usd := currencyID(t, database, "USD")

	percent, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-10", Kind: KindCorrective, Area: AreaOther, Description: "Descuento 100% demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, percent.ID, ItemInput{
		ItemType: ItemLabor, Description: "Mano de obra demo", Amount: 10, CurrencyID: usd,
	}); err != nil {
		t.Fatal(err)
	}
	other, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-10", Kind: KindCorrective, Area: AreaOther, Description: "Descuento 100x demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, other.ID, ItemInput{
		ItemType: ItemLabor, Description: "Mano de obra demo", Amount: 10, CurrencyID: usd,
	}); err != nil {
		t.Fatal(err)
	}

	byPercent, err := service.List(ctx, homeID, Filter{Query: "100%"})
	if err != nil {
		t.Fatal(err)
	}
	if len(byPercent.Items) != 1 || byPercent.Items[0].ID != percent.ID {
		t.Fatalf("query con %% devolvió %+v", byPercent.Items)
	}

	underscore, err := service.Create(ctx, homeID, Input{
		PerformedAt: "2026-03-10", Kind: KindCorrective, Area: AreaOther, Description: "Cable_a demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateItem(ctx, homeID, underscore.ID, ItemInput{
		ItemType: ItemSupplies, Description: "Insumo demo", Quantity: 1, Unit: UnitUnit, Amount: 10, CurrencyID: usd,
	}); err != nil {
		t.Fatal(err)
	}
	byUnderscore, err := service.List(ctx, homeID, Filter{Query: "Cable_a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(byUnderscore.Items) != 1 || byUnderscore.Items[0].ID != underscore.ID {
		t.Fatalf("query con _ devolvió %+v", byUnderscore.Items)
	}
}
