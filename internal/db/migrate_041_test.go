package db

import "testing"

func TestMigrate041PensionImportSchema(t *testing.T) {
	database := openMigrateTest(t, "mig041")

	for _, table := range []string{"pension_category_keywords", "pension_import_batches", "pension_import_rows"} {
		var name string
		if err := database.QueryRow(
			"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table,
		).Scan(&name); err != nil {
			t.Fatalf("falta la tabla %s: %v", table, err)
		}
	}

	var shareCol, currencyCol int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('pension_categories') WHERE name = 'share_percent'",
	).Scan(&shareCol); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('pension_categories') WHERE name = 'currency'",
	).Scan(&currencyCol); err != nil {
		t.Fatal(err)
	}
	if shareCol != 1 || currencyCol != 1 {
		t.Fatalf("faltan columnas share_percent/currency: %d/%d", shareCol, currencyCol)
	}

	res, err := database.Exec(
		"INSERT INTO pension_categories (name, description, auto_generate) VALUES ('Demo', '', 0)",
	)
	if err != nil {
		t.Fatalf("insertar categoría: %v", err)
	}
	id, _ := res.LastInsertId()

	if _, err := database.Exec("UPDATE pension_categories SET share_percent = 75 WHERE id = ?", id); err == nil {
		t.Fatalf("se esperaba error al guardar share_percent = 75")
	}
	if _, err := database.Exec("UPDATE pension_categories SET share_percent = 100 WHERE id = ?", id); err != nil {
		t.Fatalf("share_percent = 100 debería ser válido: %v", err)
	}
}
