package db

import "testing"

// TestMigrate062HomeWorkRequests verifica las tablas de reportes de viviendas
// (SPEC-164): columnas, defaults, CHECKs, FK en cascada y down reversible.
func TestMigrate062HomeWorkRequests(t *testing.T) {
	database := openMigrateTest(t, "mig062")
	database.SetMaxOpenConns(1)
	if _, err := database.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("activar foreign_keys: %v", err)
	}

	assertTableColumns(t, database, "home_work_requests", []string{
		"id", "home_id", "title", "description", "category", "priority",
		"status", "reported_at", "maintenance_id", "converted_at",
		"created_at", "updated_at",
	})
	assertTableColumns(t, database, "home_work_request_attachments", []string{
		"id", "work_request_id", "media_type", "file_name", "mime_type",
		"size_bytes", "created_at",
	})

	if _, err := database.Exec(`INSERT INTO homes (name) VALUES ('Casa Demo')`); err != nil {
		t.Fatalf("insertar vivienda: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO home_work_requests (home_id, title, reported_at)
		SELECT id, 'Filtración demo', '2026-11-26' FROM homes WHERE name = 'Casa Demo'
	`); err != nil {
		t.Fatalf("insertar reporte con defaults: %v", err)
	}

	var category, priority, status string
	if err := database.QueryRow(`
		SELECT category, priority, status FROM home_work_requests WHERE title = 'Filtración demo'
	`).Scan(&category, &priority, &status); err != nil {
		t.Fatal(err)
	}
	if category != "repair" || priority != "normal" || status != "reported" {
		t.Fatalf("defaults = %q/%q/%q", category, priority, status)
	}

	if _, err := database.Exec(`
		INSERT INTO home_work_requests (home_id, title, category, reported_at)
		SELECT id, 'Categoría inválida', 'broken', '2026-11-26' FROM homes WHERE name = 'Casa Demo'
	`); err == nil {
		t.Fatal("el CHECK de categoría debería rechazar un valor inválido")
	}
	if _, err := database.Exec(`
		INSERT INTO home_work_requests (home_id, title)
		SELECT id, 'Sin fecha' FROM homes WHERE name = 'Casa Demo'
	`); err == nil {
		t.Fatal("reported_at NOT NULL debería rechazar un reporte sin fecha")
	}

	if _, err := database.Exec(`
		INSERT INTO home_work_request_attachments
			(work_request_id, media_type, file_name, mime_type, size_bytes)
		SELECT id, 'image', 'foto-demo.jpg', 'image/jpeg', 1024
		FROM home_work_requests WHERE title = 'Filtración demo'
	`); err != nil {
		t.Fatalf("insertar adjunto: %v", err)
	}
	var workRequestID int64
	if err := database.QueryRow(`SELECT id FROM home_work_requests WHERE title = 'Filtración demo'`).Scan(&workRequestID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`DELETE FROM home_work_requests WHERE id = ?`, workRequestID); err != nil {
		t.Fatalf("eliminar reporte: %v", err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM home_work_request_attachments`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("la FK en cascada dejó %d adjuntos", count)
	}

	down, err := readMigrationFile(t, "0062_home_work_requests.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(down)); err != nil {
		t.Fatalf("ejecutar migración 0062 down: %v", err)
	}
	assertTableRemoved(t, database, "home_work_request_attachments")
	assertTableRemoved(t, database, "home_work_requests")
}
