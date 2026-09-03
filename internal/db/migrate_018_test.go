package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrate018CreateNotifications(t *testing.T) {
	dir, err := os.MkdirTemp("", "mig018")
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

	for _, col := range []string{"id", "name", "email", "active", "created_at", "updated_at"} {
		var colCount int
		if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('notifications') WHERE name = ?", col).Scan(&colCount); err != nil {
			t.Fatal(err)
		}
		if colCount != 1 {
			t.Fatalf("columna %s no encontrada, count=%d", col, colCount)
		}
	}

	// Insert sin active: debe usar default 1.
	_, err = db.Exec(`INSERT INTO notifications (name, email) VALUES ('María Pérez', 'maria@example.com')`)
	if err != nil {
		t.Fatalf("insert falló: %v", err)
	}
	var active int
	if err := db.QueryRow(`SELECT active FROM notifications WHERE email = 'maria@example.com'`).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatalf("default de active debería ser 1, got=%d", active)
	}

	// Email duplicado: debe fallar por UNIQUE.
	_, err = db.Exec(`INSERT INTO notifications (name, email) VALUES ('Otra', 'maria@example.com')`)
	if err == nil {
		t.Fatal("insert con email duplicado debería fallar")
	}

	// Down: la tabla debe desaparecer.
	down, err := os.ReadFile(filepath.Join(migrationsDir, "0018_create_notifications.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(down)); err != nil {
		t.Fatalf("down migration falló: %v", err)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='notifications'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("la tabla notifications debería haber sido removida, count=%d", count)
	}
}