package db

import "testing"

func TestMigrate057HomeMaintenance(t *testing.T) {
	database := openMigrateTest(t, "mig057")

	for _, table := range []string{"home_maintenances", "home_maintenance_items"} {
		var count int
		if err := database.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("%s debería existir después de 0057", table)
		}
	}

	var categoryID int64
	if err := database.QueryRow(
		"SELECT id FROM institution_categories WHERE key = 'hardware'",
	).Scan(&categoryID); err != nil {
		t.Fatalf("categoría hardware: %v", err)
	}

	// SPEC-141: la moneda vive en cada línea, no en la reparación.
	var currencyInMaintenance int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('home_maintenances') WHERE name = 'currency_id'",
	).Scan(&currencyInMaintenance); err != nil {
		t.Fatal(err)
	}
	if currencyInMaintenance != 0 {
		t.Fatal("home_maintenances no debería tener currency_id")
	}
	var currencyInItem int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('home_maintenance_items') WHERE name = 'currency_id'",
	).Scan(&currencyInItem); err != nil {
		t.Fatal(err)
	}
	if currencyInItem != 1 {
		t.Fatal("home_maintenance_items debería tener currency_id")
	}
	var currencyOnDelete string
	if err := database.QueryRow(`
		SELECT "on_delete" FROM pragma_foreign_key_list('home_maintenance_items')
		WHERE "table" = 'currencies'
	`).Scan(&currencyOnDelete); err != nil {
		t.Fatalf("fk moneda: %v", err)
	}
	if currencyOnDelete == "" {
		t.Fatal("home_maintenance_items.currency_id debería referenciar currencies")
	}

	// La FK de institución debe preservar el histórico al borrar.
	var onDelete string
	if err := database.QueryRow(`
		SELECT "on_delete" FROM pragma_foreign_key_list('home_maintenances')
		WHERE "table" = 'institutions'
	`).Scan(&onDelete); err != nil {
		t.Fatalf("fk institución: %v", err)
	}
	if onDelete != "SET NULL" {
		t.Fatalf("on_delete = %q, quería SET NULL", onDelete)
	}

	down, err := readMigrationFile(t, "0057_home_maintenance.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(down)); err != nil {
		t.Fatalf("ejecutar 0057 down: %v", err)
	}

	for _, table := range []string{"home_maintenances", "home_maintenance_items"} {
		var count int
		if err := database.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s debería eliminarse con el down", table)
		}
	}

	var hardware int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM institution_categories WHERE key = 'hardware'",
	).Scan(&hardware); err != nil {
		t.Fatal(err)
	}
	if hardware != 0 {
		t.Fatal("la categoría hardware debería eliminarse con el down")
	}
}
