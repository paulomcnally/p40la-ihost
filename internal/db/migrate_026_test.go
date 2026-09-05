package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrate026AddHomesSortOrder(t *testing.T) {
	dir, err := os.MkdirTemp("", "mig026")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	dbPath := filepath.Join(dir, "test.db")
	dsn := "file:" + dbPath
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })

	// Reproduce el esquema de homes previo a la migración 0026.
	if _, err := database.Exec(`
		CREATE TABLE homes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			address TEXT,
			deleted_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_homes_deleted_at ON homes(deleted_at);
	`); err != nil {
		t.Fatal(err)
	}
	// Insertar casas en orden de inserción NO alfabético para validar el backfill.
	if _, err := database.Exec(`INSERT INTO homes (name) VALUES ('Casa Zeta'), ('Casa Alfa'), ('Casa Medio')`); err != nil {
		t.Fatalf("insertar casas falló: %v", err)
	}

	up, err := readMigrationFile(t, "0026_add_homes_sort_order.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(up)); err != nil {
		t.Fatalf("up migration falló: %v", err)
	}

	assertTableColumns(t, database, "homes", []string{
		"id", "name", "address", "sort_order", "deleted_at", "created_at", "updated_at",
	})

	// Backfill debe preservar el orden alfabético previo: Alfa(0), Medio(1), Zeta(2).
	rows, err := database.Query(`SELECT name, sort_order FROM homes ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	got := map[string]int{}
	for rows.Next() {
		var name string
		var order int
		if err := rows.Scan(&name, &order); err != nil {
			t.Fatal(err)
		}
		got[name] = order
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	expected := map[string]int{"Casa Alfa": 0, "Casa Medio": 1, "Casa Zeta": 2}
	for name, order := range expected {
		if got[name] != order {
			t.Errorf("backfill incorrecto para %q: sort_order=%d, se esperaba %d", name, got[name], order)
		}
	}

	// Un insert sin sort_order explícito usa el DEFAULT 0.
	if _, err := database.Exec(`INSERT INTO homes (name) VALUES ('Casa Nueva')`); err != nil {
		t.Fatalf("insertar casa sin sort_order: %v", err)
	}
	var defaultOrder int
	if err := database.QueryRow(`SELECT sort_order FROM homes WHERE name = 'Casa Nueva'`).Scan(&defaultOrder); err != nil {
		t.Fatal(err)
	}
	if defaultOrder != 0 {
		t.Errorf("DEFAULT de sort_order debería ser 0, got %d", defaultOrder)
	}

	down, err := readMigrationFile(t, "0026_add_homes_sort_order.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM pragma_table_info('homes') WHERE name = 'sort_order'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("la columna sort_order debería haberse removido, count=%d", count)
	}
}
