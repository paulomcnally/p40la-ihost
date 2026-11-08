package db

import "testing"

func TestMigrate043AutoMaintenanceInstitutionReference(t *testing.T) {
	database := openMigrateTest(t, "mig043")

	// Estado final: sin workshop_name e institution_id NOT NULL.
	var workshopColumn int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('auto_maintenances') WHERE name = 'workshop_name'",
	).Scan(&workshopColumn); err != nil {
		t.Fatal(err)
	}
	if workshopColumn != 0 {
		t.Fatalf("workshop_name no debería existir después de 0043")
	}
	var notNull int
	if err := database.QueryRow(
		"SELECT \"notnull\" FROM pragma_table_info('auto_maintenances') WHERE name = 'institution_id'",
	).Scan(&notNull); err != nil {
		t.Fatal(err)
	}
	if notNull != 1 {
		t.Fatalf("institution_id debería ser NOT NULL después de 0043")
	}

	// Volver a 0042: primero deshacer 0044 (líneas) y luego 0043 (taller).
	for _, name := range []string{
		"0044_auto_maintenance_items.down.sql",
		"0043_auto_maintenance_institution_reference.down.sql",
	} {
		data, err := readMigrationFile(t, name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(string(data)); err != nil {
			t.Fatalf("ejecutar %s: %v", name, err)
		}
	}

	res, err := database.Exec(
		"INSERT INTO autos (year, model, brand, color) VALUES (2020, 'Corolla', 'Toyota', 'white')",
	)
	if err != nil {
		t.Fatalf("insertar auto: %v", err)
	}
	autoID, _ := res.LastInsertId()

	res, err = database.Exec(`
		INSERT INTO auto_maintenances
			(auto_id, performed_at, mileage, workshop_name, description, labor_cost, supplies_cost, currency_id)
		VALUES (?, '2026-10-15', 1000, 'Taller Legacy Demo', 'Demo', 10, 20, 1)
	`, autoID)
	if err != nil {
		t.Fatalf("insertar mantenimiento legacy: %v", err)
	}
	maintenanceID, _ := res.LastInsertId()

	for _, name := range []string{
		"0043_auto_maintenance_institution_reference.up.sql",
		"0044_auto_maintenance_items.up.sql",
	} {
		data, err := readMigrationFile(t, name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.Exec(string(data)); err != nil {
			t.Fatalf("ejecutar %s: %v", name, err)
		}
	}

	var institutionID int64
	var institutionName string
	if err := database.QueryRow(`
		SELECT m.institution_id, i.name
		FROM auto_maintenances m
		JOIN institutions i ON i.id = m.institution_id
		WHERE m.id = ?
	`, maintenanceID).Scan(&institutionID, &institutionName); err != nil {
		t.Fatalf("leer mantenimiento migrado: %v", err)
	}
	if institutionName != "Taller Legacy Demo" {
		t.Fatalf("institución migrada: esperado Taller Legacy Demo, got %q", institutionName)
	}

	var categoryKey string
	if err := database.QueryRow(`
		SELECT ic.key
		FROM institutions i
		JOIN institution_categories ic ON ic.id = i.category_id
		WHERE i.id = ?
	`, institutionID).Scan(&categoryKey); err != nil {
		t.Fatalf("leer categoría de la institución migrada: %v", err)
	}
	if categoryKey != "workshop" {
		t.Fatalf("categoría de la institución migrada: esperado workshop, got %q", categoryKey)
	}

	// La columna workshop_name ya no existe y no se puede insertar sin institución.
	if _, err := database.Exec(`
		INSERT INTO auto_maintenances
			(auto_id, performed_at, mileage, description, currency_id)
		VALUES (?, '2026-10-16', 2000, 'Demo', 1)
	`, autoID); err == nil {
		t.Fatal("insertar sin institution_id debería fallar")
	}
}
