package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func openMigrateTest(t *testing.T, suffix string) *sql.DB {
	t.Helper()
	dir, err := os.MkdirTemp("", suffix)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

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
	t.Cleanup(func() { db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db, migrationsDir); err != nil {
		t.Fatal(err)
	}
	return db
}

func assertTableColumns(t *testing.T, db *sql.DB, table string, cols []string) {
	t.Helper()
	for _, col := range cols {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('"+table+"') WHERE name = ?", col).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("columna %s.%s no encontrada, count=%d", table, col, count)
		}
	}
}

func assertTableRemoved(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("la tabla %s debería haber sido removida, count=%d", table, count)
	}
}

func readMigrationFile(t *testing.T, name string) ([]byte, error) {
	t.Helper()
	return os.ReadFile(filepath.Join("../../migrations", name))
}
