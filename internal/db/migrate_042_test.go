package db

import "testing"

func TestMigrate042AutoMaintenanceInspection(t *testing.T) {
	database := openMigrateTest(t, "mig042")

	assertTableColumns(t, database, "auto_maintenances", []string{
		"id", "auto_id", "performed_at", "mileage", "institution_id",
		"workshop_name", "description", "labor_cost", "supplies_cost",
		"currency_id", "created_at", "updated_at",
	})
	assertTableColumns(t, database, "auto_inspections", []string{
		"id", "auto_id", "performed_at", "inspection_year", "institution_id",
		"result", "notes", "created_at", "updated_at",
	})
	assertTableColumns(t, database, "auto_attachments", []string{
		"id", "auto_id", "owner_type", "owner_id", "file_name", "size_bytes", "created_at",
	})

	for _, key := range []string{"workshop", "mechanical_inspection"} {
		var count int
		if err := database.QueryRow(
			"SELECT COUNT(*) FROM institution_categories WHERE key = ?", key,
		).Scan(&count); err != nil {
			t.Fatalf("buscar categoría %s: %v", key, err)
		}
		if count != 1 {
			t.Fatalf("categoría %s: esperado 1, got %d", key, count)
		}
	}

	// Una institución asociada a la categoría debe sobrevivir al down sin FK rota.
	var workshopID int64
	if err := database.QueryRow("SELECT id FROM institution_categories WHERE key = 'workshop'").Scan(&workshopID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(
		"INSERT INTO institutions (name, category_id) VALUES ('Taller Demo', ?)", workshopID,
	); err != nil {
		t.Fatalf("insertar institución: %v", err)
	}

	down, err := readMigrationFile(t, "0042_create_auto_maintenance_inspection.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(down)); err != nil {
		t.Fatalf("ejecutar migración 0042 down: %v", err)
	}

	assertTableRemoved(t, database, "auto_maintenances")
	assertTableRemoved(t, database, "auto_inspections")
	assertTableRemoved(t, database, "auto_attachments")

	var categoryID *int64
	if err := database.QueryRow("SELECT category_id FROM institutions WHERE name = 'Taller Demo'").Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	if categoryID != nil {
		t.Fatalf("la categoría de la institución debería quedar NULL, got %d", *categoryID)
	}

	up, err := readMigrationFile(t, "0042_create_auto_maintenance_inspection.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(up)); err != nil {
		t.Fatalf("re-ejecutar migración 0042 up: %v", err)
	}
	// Idempotencia: un segundo pase tampoco debe fallar.
	if _, err := database.Exec(string(up)); err != nil {
		t.Fatalf("segundo pase de la migración 0042: %v", err)
	}
}
