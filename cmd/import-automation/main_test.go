package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/paulomcnally/p40la-ihost/internal/db"
)

const migrationsDir = "../../migrations"

func execScript(t *testing.T, database *sql.DB, script string) {
	t.Helper()
	if _, err := database.Exec(script); err != nil {
		t.Fatalf("ejecutar script: %v", err)
	}
}

// newSource crea una DB de automation mínima con las tablas que lee el import.
func newSource(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "automation.db")
	src, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("abrir source: %v", err)
	}
	t.Cleanup(func() { src.Close() })

	execScript(t, src, `
		CREATE TABLE apps (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE plugins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			version TEXT NOT NULL,
			source TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			app_id INTEGER NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
			identifier TEXT NOT NULL,
			label TEXT,
			credentials TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			plugin_name TEXT REFERENCES plugins(name),
			webhook_url TEXT,
			schedule_time TEXT,
			schedule_enabled INTEGER NOT NULL DEFAULT 0,
			last_run_at DATETIME,
			schedule_frequency TEXT NOT NULL DEFAULT 'daily',
			schedule_interval INTEGER,
			schedule_days TEXT,
			UNIQUE(app_id, identifier)
		);
		CREATE TABLE bills (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			plugin_name TEXT NOT NULL,
			plugin_version TEXT NOT NULL,
			source TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'ok',
			error TEXT,
			raw TEXT NOT NULL,
			fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE webhook_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			webhook_url TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'ok',
			payload TEXT NOT NULL,
			response TEXT NOT NULL,
			ran_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			http_status INTEGER
		);
		CREATE TABLE settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			value TEXT NOT NULL
		);

		INSERT INTO plugins (id,name,version,source,description,status) VALUES
			(1,'claro.nicaragua','1.0.0','builtin','Claro','active');
		INSERT INTO apps (id,name) VALUES (1,'Claro Nicaragua');
		INSERT INTO accounts (id,app_id,identifier,label,credentials,plugin_name,webhook_url,schedule_time,schedule_enabled,schedule_frequency)
			VALUES
			(10,1,'84126107','Casa','{"user":"x"}','claro.nicaragua','http://p40la/webhooks/abc','07:00',1,'daily'),
			(11,1,'999','Oficina','{"user":"y"}','plugin.ausente','http://p40la/webhooks/def','08:00',0,'weekly');
		INSERT INTO bills (id,account_id,plugin_name,plugin_version,source,status,raw)
			VALUES (100,10,'claro.nicaragua','1.0.0','plugin','ok','{"amount":10}'),
			       (101,11,'plugin.ausente','1.0.0','plugin','ok','{"amount":20}');
		INSERT INTO webhook_logs (id,account_id,webhook_url,status,payload,response,http_status)
			VALUES (200,10,'http://p40la/webhooks/abc','ok','{}','{}',200),
			       (201,11,'http://p40la/webhooks/def','error','{}','boom',500);
		INSERT INTO settings (key,value) VALUES
			('webhook_api_key','NO_DEBE_COPIARSE'),
			('webhook_enabled','true'),
			('language','es');
	`)
	return src
}

func newTargetDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	target, err := db.OpenDB(filepath.Join(dir, "app.db"), migrationsDir)
	if err != nil {
		t.Fatalf("crear target: %v", err)
	}
	// La key de p40la no debe ser pisada por el import.
	if _, err := target.Exec(
		"INSERT INTO system_settings (key,value) VALUES ('webhook_api_key','KEY_P40LA')"); err != nil {
		t.Fatalf("insertar key p40la: %v", err)
	}
	target.Close()
	return dir
}

func TestImportPreservesDataAndIsIdempotent(t *testing.T) {
	src := newSource(t)
	dir := newTargetDir(t)

	opts := options{sourcePath: sourcePath(t, src), dataDir: dir, migrationsDir: migrationsDir, verify: true}
	if err := runImport(context.Background(), opts); err != nil {
		t.Fatalf("primer import: %v", err)
	}
	if err := runImport(context.Background(), opts); err != nil {
		t.Fatalf("segundo import (idempotencia): %v", err)
	}

	dst, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "app.db")+"?mode=ro")
	if err != nil {
		t.Fatalf("abrir target: %v", err)
	}
	defer dst.Close()

	checks := []struct {
		query string
		want  int64
	}{
		{"SELECT COUNT(*) FROM automation_apps", 1},
		{"SELECT COUNT(*) FROM automation_plugins", 2}, // 1 real + 1 stub
		{"SELECT COUNT(*) FROM automation_accounts", 2},
		{"SELECT COUNT(*) FROM automation_bills", 2},
		{"SELECT COUNT(*) FROM automation_webhook_logs", 2},
	}
	for _, c := range checks {
		var got int64
		if err := dst.QueryRow(c.query).Scan(&got); err != nil {
			t.Fatalf("query %q: %v", c.query, err)
		}
		if got != c.want {
			t.Errorf("%q = %d, esperado %d", c.query, got, c.want)
		}
	}

	var apiKey, enabled, lang string
	if err := dst.QueryRow("SELECT value FROM system_settings WHERE key='webhook_api_key'").Scan(&apiKey); err != nil {
		t.Fatal(err)
	}
	if apiKey != "KEY_P40LA" {
		t.Errorf("webhook_api_key = %q, no debe ser pisada", apiKey)
	}
	if err := dst.QueryRow("SELECT value FROM system_settings WHERE key='webhook_enabled'").Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != "true" {
		t.Errorf("webhook_enabled = %q", enabled)
	}
	if err := dst.QueryRow("SELECT value FROM system_settings WHERE key='language'").Scan(&lang); err != nil {
		t.Fatal(err)
	}
	if lang != "es" {
		t.Errorf("language = %q", lang)
	}

	var stub int
	if err := dst.QueryRow(
		"SELECT COUNT(*) FROM automation_plugins WHERE name='plugin.ausente' AND source='import'").Scan(&stub); err != nil {
		t.Fatal(err)
	}
	if stub != 1 {
		t.Errorf("stub de plugin ausente = %d, esperado 1", stub)
	}

	// Verifica que los IDs se preserven (services.automation_account_id apunta a ellos).
	var identifier string
	if err := dst.QueryRow("SELECT identifier FROM automation_accounts WHERE id=10").Scan(&identifier); err != nil {
		t.Fatal(err)
	}
	if identifier != "84126107" {
		t.Errorf("identifier de cuenta 10 = %q", identifier)
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	src := newSource(t)
	dir := t.TempDir()

	opts := options{sourcePath: sourcePath(t, src), dataDir: dir, migrationsDir: migrationsDir, dryRun: true}
	if err := runImport(context.Background(), opts); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if _, err := filepath.Glob(filepath.Join(dir, "app.db*")); err != nil {
		t.Fatal(err)
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, "app.db*")); len(matches) != 0 {
		t.Errorf("dry-run creó archivos: %v", matches)
	}
}

func TestValidateSourceMissingTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "incompleta.db")
	database, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	execScript(t, database, "CREATE TABLE apps (id INTEGER PRIMARY KEY, name TEXT);")

	err = validateSource(context.Background(), database)
	if err == nil {
		t.Fatal("se esperaba error por tablas faltantes")
	}
}

func TestCollectOrphans(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orphans.db")
	database, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	execScript(t, database, `
		CREATE TABLE automation_accounts (id INTEGER PRIMARY KEY);
		CREATE TABLE services (id INTEGER PRIMARY KEY, automation_account_id INTEGER);
		INSERT INTO automation_accounts (id) VALUES (10);
		INSERT INTO services (id, automation_account_id) VALUES (1, 10), (2, 999), (3, NULL);
	`)

	summary := newSummary()
	if err := collectOrphans(context.Background(), database, summary); err != nil {
		t.Fatalf("collectOrphans: %v", err)
	}
	if len(summary.Orphans) != 1 || summary.Orphans[0] != 999 {
		t.Errorf("huérfanos = %v, esperado [999]", summary.Orphans)
	}
}

// sourcePath materializa el *sql.DB de test en una ruta concreta.
func sourcePath(t *testing.T, source *sql.DB) string {
	t.Helper()
	var path string
	if err := source.QueryRow("PRAGMA database_list").Scan(new(int), new(string), &path); err != nil {
		t.Fatalf("obtener ruta de source: %v", err)
	}
	return path
}
