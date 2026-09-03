package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrate019CreatePensionCategories(t *testing.T) {
	dir, err := os.MkdirTemp("", "mig019")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	migrationsDir, err := filepath.Abs("../../migrations")
	if err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(dir, "test.db")
	dsn := "file:" + dbPath
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db, migrationsDir); err != nil {
		t.Fatal(err)
	}

	for _, col := range []string{"id", "name", "description", "auto_generate", "created_at", "updated_at"} {
		var colCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('pension_categories') WHERE name = ?", col).Scan(&colCount); err != nil {
			t.Fatal(err)
		}
		if colCount != 1 {
			t.Fatalf("columna %s no encontrada, count=%d", col, colCount)
		}
	}

	// Insert con descripción NULL y auto_generate por defecto (0).
	_, err = db.Exec(`INSERT INTO pension_categories (name, description)
		VALUES ('Alimentación', NULL)`)
	if err != nil {
		t.Fatalf("insert con descripción NULL falló: %v", err)
	}
	_, err = db.Exec(`INSERT INTO pension_categories (name, description, auto_generate)
		VALUES ('Educación', 'Colegios y universidades', 1)`)
	if err != nil {
		t.Fatalf("insert con auto_generate falló: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM pension_categories").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("se esperaban 2 categorías, count=%d", count)
	}

	var defaultAuto int
	if err := db.QueryRow("SELECT auto_generate FROM pension_categories WHERE name = 'Alimentación'").Scan(&defaultAuto); err != nil {
		t.Fatal(err)
	}
	if defaultAuto != 0 {
		t.Fatalf("auto_generate default debería ser 0, got=%d", defaultAuto)
	}

	// Down: revertir.
	down, err := os.ReadFile(filepath.Join(migrationsDir, "0019_create_pension_categories.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}

	var tableCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='pension_categories'").Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if tableCount != 0 {
		t.Fatalf("la tabla pension_categories debería haber sido removida, count=%d", tableCount)
	}
}