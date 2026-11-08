package db

import "testing"

func TestMigrate044AutoMaintenanceItems(t *testing.T) {
	database := openMigrateTest(t, "mig044")

	// Estado final: sin columnas escalares y con tabla de líneas.
	for _, column := range []string{"labor_cost", "supplies_cost"} {
		var count int
		if err := database.QueryRow(
			"SELECT COUNT(*) FROM pragma_table_info('auto_maintenances') WHERE name = ?", column,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s no debería existir después de 0044", column)
		}
	}
	var itemsTable int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='auto_maintenance_items'",
	).Scan(&itemsTable); err != nil {
		t.Fatal(err)
	}
	if itemsTable != 1 {
		t.Fatal("auto_maintenance_items debería existir después de 0044")
	}

	down, err := readMigrationFile(t, "0044_auto_maintenance_items.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(down)); err != nil {
		t.Fatalf("ejecutar 0044 down: %v", err)
	}

	var categoryID int64
	if err := database.QueryRow(
		"SELECT id FROM institution_categories WHERE key = 'workshop'",
	).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	res, err := database.Exec(
		"INSERT INTO institutions (name, category_id) VALUES ('Taller Demo 0044', ?)", categoryID,
	)
	if err != nil {
		t.Fatalf("insertar institución: %v", err)
	}
	institutionID, _ := res.LastInsertId()

	res, err = database.Exec(
		"INSERT INTO autos (year, model, brand, color) VALUES (2020, 'Corolla', 'Toyota', 'white')",
	)
	if err != nil {
		t.Fatalf("insertar auto: %v", err)
	}
	autoID, _ := res.LastInsertId()

	res, err = database.Exec(`
		INSERT INTO auto_maintenances
			(auto_id, performed_at, mileage, institution_id, description, labor_cost, supplies_cost, currency_id)
		VALUES (?, '2026-10-15', 1000, ?, 'Demo', 10, 20, 1)
	`, autoID, institutionID)
	if err != nil {
		t.Fatalf("insertar mantenimiento: %v", err)
	}
	maintenanceID, _ := res.LastInsertId()

	up, err := readMigrationFile(t, "0044_auto_maintenance_items.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(up)); err != nil {
		t.Fatalf("ejecutar 0044 up: %v", err)
	}

	rows, err := database.Query(`
		SELECT item_type, description, amount
		FROM auto_maintenance_items
		WHERE maintenance_id = ?
		ORDER BY id
	`, maintenanceID)
	if err != nil {
		t.Fatalf("listar líneas migradas: %v", err)
	}
	defer rows.Close()

	got := map[string]struct {
		description string
		amount      float64
	}{}
	for rows.Next() {
		var itemType, description string
		var amount float64
		if err := rows.Scan(&itemType, &description, &amount); err != nil {
			t.Fatal(err)
		}
		got[itemType] = struct {
			description string
			amount      float64
		}{description, amount}
	}
	if labor, ok := got["labor"]; !ok || labor.description != "Mano de obra" || labor.amount != 10 {
		t.Fatalf("línea de mano de obra migrada inesperada: %+v", labor)
	}
	if supplies, ok := got["supplies"]; !ok || supplies.description != "Insumos" || supplies.amount != 20 {
		t.Fatalf("línea de insumos migrada inesperada: %+v", supplies)
	}
}
