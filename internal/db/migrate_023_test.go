package db

import "testing"

func TestMigrate023CreateChildSupportConfigs(t *testing.T) {
	db := openMigrateTest(t, "mig023")

	assertTableColumns(t, db, "child_support_configs", []string{
		"id", "child_id", "pension_category_id", "amount", "currency",
		"is_active", "auto_generate", "created_at", "updated_at",
	})

	_, err := db.Exec(`INSERT INTO children (first_name, last_name, birth_date) VALUES ('Juan', 'Pérez', '2015-01-01')`)
	if err != nil {
		t.Fatalf("insert hijo falló: %v", err)
	}
	_, err = db.Exec(`INSERT INTO pension_categories (name) VALUES ('Colegio')`)
	if err != nil {
		t.Fatalf("insert categoría falló: %v", err)
	}
	_, err = db.Exec(`INSERT INTO child_support_configs (child_id, pension_category_id, amount, currency) VALUES (1, 1, 1500, 'NIO')`)
	if err != nil {
		t.Fatalf("insert config falló: %v", err)
	}

	var isActive, autoGen int
	if err := db.QueryRow(`SELECT is_active, auto_generate FROM child_support_configs WHERE id = 1`).Scan(&isActive, &autoGen); err != nil {
		t.Fatal(err)
	}
	if isActive != 1 || autoGen != 0 {
		t.Fatalf("defaults incorrectos: is_active=%d auto_generate=%d", isActive, autoGen)
	}

	// UNIQUE (child_id, pension_category_id)
	_, err = db.Exec(`INSERT INTO child_support_configs (child_id, pension_category_id, amount) VALUES (1, 1, 900)`)
	if err == nil {
		t.Fatal("se esperaba error UNIQUE por duplicado de hijo+categoría")
	}

	down, err := readMigrationFile(t, "0023_create_child_support_configs.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}
	assertTableRemoved(t, db, "child_support_configs")
}
