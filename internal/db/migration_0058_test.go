package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSpec143MigrationsReversible(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mig.db")
	database, err := OpenDB(dbPath, filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatalf("abrir: %v", err)
	}
	defer database.Close()

	run := func(name string) {
		data, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if err != nil {
			t.Fatalf("leer %s: %v", name, err)
		}
		if _, err := database.Exec(string(data)); err != nil {
			t.Fatalf("ejecutar %s: %v", name, err)
		}
	}
	hasColumn := func(table, column string) bool {
		rows, err := database.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
		if err != nil {
			t.Fatalf("tabla %s: %v", table, err)
		}
		defer rows.Close()
		for rows.Next() {
			var cid, notnull, pk int
			var name, ctype string
			var dflt any
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
				t.Fatalf("scan: %v", err)
			}
			if name == column {
				return true
			}
		}
		return false
	}

	if !hasColumn("homes", "usage") || hasColumn("homes", "listing_enabled") {
		t.Fatal("estado inesperado tras up")
	}
	run("0059_portfolio_indexes.down.sql")
	run("0058_homes_usage.down.sql")
	if !hasColumn("homes", "listing_enabled") || hasColumn("homes", "usage") {
		t.Fatal("estado inesperado tras down")
	}
	run("0058_homes_usage.up.sql")
	run("0059_portfolio_indexes.up.sql")
	if !hasColumn("homes", "usage") || hasColumn("homes", "listing_enabled") {
		t.Fatal("estado inesperado tras re-up")
	}
}
