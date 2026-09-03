package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrate016CreateChildren(t *testing.T) {
	dir, err := os.MkdirTemp("", "mig016")
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

	for _, col := range []string{"id", "first_name", "last_name", "birth_date", "notes", "created_at", "updated_at"} {
		var colCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('children') WHERE name = ?", col).Scan(&colCount); err != nil {
			t.Fatal(err)
		}
		if colCount != 1 {
			t.Fatalf("columna %s no encontrada, count=%d", col, colCount)
		}
	}

	// Insert con notas NULL y con notas.
	_, err = db.Exec(`INSERT INTO children (first_name, last_name, birth_date, notes)
		VALUES ('María', 'Pérez', '2019-05-12', NULL)`)
	if err != nil {
		t.Fatalf("insert con notas NULL falló: %v", err)
	}
	_, err = db.Exec(`INSERT INTO children (first_name, last_name, birth_date, notes)
		VALUES ('Juan', 'Gómez', '2015-03-01', 'Alergia al maní')`)
	if err != nil {
		t.Fatalf("insert con notas falló: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM children").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("se esperaban 2 hijos, count=%d", count)
	}

	// Down: revertir.
	down, err := os.ReadFile(filepath.Join(migrationsDir, "0016_create_children.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}

	var tableCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='children'").Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if tableCount != 0 {
		t.Fatalf("la tabla children debería haber sido removida, count=%d", tableCount)
	}
}