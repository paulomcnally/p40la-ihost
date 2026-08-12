package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

// OpenDB abre la base de datos SQLite y aplica las migraciones.
func OpenDB(dbPath, migrationsDir string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("abrir base de datos: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping base de datos: %w", err)
	}
	if err := Migrate(db, migrationsDir); err != nil {
		return nil, fmt.Errorf("migrar base de datos: %w", err)
	}
	return db, nil
}

// Migrate aplica todos los archivos .up.sql del directorio de migraciones en orden.
func Migrate(db *sql.DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("leer migraciones: %w", err)
	}

	var ups []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			ups = append(ups, e.Name())
		}
	}
	sort.Strings(ups)

	for _, name := range ups {
		data, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return fmt.Errorf("leer migración %s: %w", name, err)
		}
		if _, err := db.Exec(string(data)); err != nil {
			return fmt.Errorf("ejecutar migración %s: %w", name, err)
		}
	}
	return nil
}
