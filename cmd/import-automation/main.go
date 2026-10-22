// Comando offline de importación de p40la-ihost-automation hacia la DB de
// p40la-ihost (SPEC-109). Se ejecuta con la aplicación detenida y sobre una
// copia de la DB origen hecha con `sqlite3 .backup` (nunca copia plana: se
// perdería el contenido del -wal).
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/paulomcnally/p40la-ihost/internal/db"
)

// tableCount acumula filas leídas e insertadas por tabla.
type tableCount struct {
	Read     int64
	Inserted int64
}

// importSummary es el resultado de una corrida (real o dry-run).
type importSummary struct {
	Tables   map[string]*tableCount
	Orphans  []int64
	Stubs    []string
	Warnings []string
}

func newSummary() *importSummary {
	return &importSummary{Tables: make(map[string]*tableCount)}
}

func (s *importSummary) count(table string) *tableCount {
	if s.Tables[table] == nil {
		s.Tables[table] = &tableCount{}
	}
	return s.Tables[table]
}

func (s *importSummary) tableNames() []string {
	names := make([]string, 0, len(s.Tables))
	for name := range s.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// print imprime el resumen legible para el operador.
func (s *importSummary) print(dryRun bool) {
	if dryRun {
		fmt.Println("== DRY RUN (no se escribió nada) ==")
	}
	for _, name := range s.tableNames() {
		c := s.Tables[name]
		fmt.Printf("%s: leídas=%d insertadas=%d omitidas=%d\n", name, c.Read, c.Inserted, c.Read-c.Inserted)
	}
	for _, stub := range s.Stubs {
		fmt.Printf("aviso: plugin ausente en origen, stub creado: %s\n", stub)
	}
	for _, w := range s.Warnings {
		fmt.Printf("aviso: %s\n", w)
	}
	if len(s.Orphans) == 0 {
		fmt.Println("huérfanos: services.automation_account_id -> []")
	} else {
		fmt.Printf("huérfanos: services.automation_account_id -> %v (resolver en la UI)\n", s.Orphans)
	}
}

type options struct {
	sourcePath    string
	dataDir       string
	migrationsDir string
	dryRun        bool
	verify        bool
}

func main() {
	var opts options
	flag.StringVar(&opts.sourcePath, "from", "", "ruta de la copia de la DB de automation (obligatorio)")
	flag.StringVar(&opts.dataDir, "data-dir", defaultDataDir(), "directorio de datos de p40la (default: DATA_DIR o ./data)")
	flag.StringVar(&opts.migrationsDir, "migrations", "./migrations", "directorio de migraciones de p40la")
	flag.BoolVar(&opts.dryRun, "dry-run", false, "validar y reportar sin escribir datos")
	flag.BoolVar(&opts.verify, "verify", false, "comparar conteos origen/destino al finalizar")
	flag.Parse()

	if strings.TrimSpace(opts.sourcePath) == "" {
		fmt.Fprintln(os.Stderr, "error: --from es obligatorio")
		flag.Usage()
		os.Exit(1)
	}

	if err := runImport(context.Background(), opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func defaultDataDir() string {
	if dir := os.Getenv("DATA_DIR"); dir != "" {
		return dir
	}
	return "./data"
}

func runImport(ctx context.Context, opts options) error {
	src, err := openSource(opts.sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()

	if err := validateSource(ctx, src); err != nil {
		return err
	}

	targetPath := filepath.Join(opts.dataDir, "app.db")
	summary := newSummary()

	if opts.dryRun {
		if err := dryRun(ctx, src, targetPath, summary); err != nil {
			return err
		}
		summary.print(true)
		return nil
	}

	dst, err := db.OpenDB(targetPath, opts.migrationsDir)
	if err != nil {
		return fmt.Errorf("abrir DB destino: %w", err)
	}
	defer dst.Close()

	if err := importAll(ctx, src, dst, summary); err != nil {
		return err
	}
	if err := collectOrphans(ctx, dst, summary); err != nil {
		return err
	}
	summary.print(false)

	if err := checkIntegrity(ctx, dst); err != nil {
		return err
	}
	if opts.verify {
		if err := verifyCounts(ctx, src, dst, summary); err != nil {
			return err
		}
	}
	return nil
}

// openSource abre la copia de automation en modo solo lectura.
func openSource(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("no se encontró la DB origen %s: %w", path, err)
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("abrir DB origen: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping DB origen: %w", err)
	}
	return db, nil
}

var requiredSourceTables = []string{"apps", "plugins", "accounts", "bills", "webhook_logs", "settings"}

func validateSource(ctx context.Context, src *sql.DB) error {
	var missing []string
	for _, table := range requiredSourceTables {
		ok, err := tableExists(ctx, src, table)
		if err != nil {
			return err
		}
		if !ok {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("la DB origen no tiene las tablas: %s", strings.Join(missing, ", "))
	}
	return nil
}

func tableExists(ctx context.Context, database *sql.DB, table string) (bool, error) {
	var name string
	err := database.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name = ?", table,
	).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("verificar tabla %s: %w", table, err)
	}
	return true, nil
}

// dryRun no escribe nada: cuenta el origen y, si el destino ya está migrado,
// reporta los huérfanos existentes.
func dryRun(ctx context.Context, src *sql.DB, targetPath string, summary *importSummary) error {
	for _, table := range requiredSourceTables {
		var n int64
		if err := src.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil {
			return fmt.Errorf("contar %s: %w", table, err)
		}
		summary.count(table).Read = n
	}

	dst, err := sql.Open("sqlite", "file:"+targetPath+"?mode=ro")
	if err != nil {
		return nil // sin destino legible el dry-run igual reporta el origen
	}
	defer dst.Close()
	migrated, err := tableExists(ctx, dst, "automation_accounts")
	if err != nil {
		return nil
	}
	if migrated {
		return collectOrphans(ctx, dst, summary)
	}
	summary.Warnings = append(summary.Warnings, "destino sin migrar: la corrida real aplicará la migración 0039")
	return nil
}

func importAll(ctx context.Context, src, dst *sql.DB, summary *importSummary) error {
	tx, err := dst.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transacción: %w", err)
	}
	defer tx.Rollback()

	if err := importPlugins(ctx, tx, src, summary); err != nil {
		return err
	}
	if err := importApps(ctx, tx, src, summary); err != nil {
		return err
	}
	if err := importAccounts(ctx, tx, src, summary); err != nil {
		return err
	}
	if err := importBills(ctx, tx, src, summary); err != nil {
		return err
	}
	if err := importWebhookLogs(ctx, tx, src, summary); err != nil {
		return err
	}
	if err := importSettings(ctx, tx, src, summary); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar import: %w", err)
	}
	return nil
}

func importPlugins(ctx context.Context, tx *sql.Tx, src *sql.DB, summary *importSummary) error {
	rows, err := src.QueryContext(ctx,
		"SELECT id, name, version, source, COALESCE(description,''), status FROM plugins ORDER BY id")
	if err != nil {
		return fmt.Errorf("leer plugins: %w", err)
	}
	defer rows.Close()

	count := summary.count("plugins")
	for rows.Next() {
		var id int64
		var name, version, source, description, status string
		if err := rows.Scan(&id, &name, &version, &source, &description, &status); err != nil {
			return fmt.Errorf("scan plugin: %w", err)
		}
		count.Read++
		res, err := tx.ExecContext(ctx,
			"INSERT OR IGNORE INTO automation_plugins (id,name,version,source,description,status) VALUES (?,?,?,?,?,?)",
			id, name, version, source, description, status)
		if err != nil {
			return fmt.Errorf("insertar plugin %s: %w", name, err)
		}
		count.Inserted += mustAffected(res)
	}
	return rows.Err()
}

func importApps(ctx context.Context, tx *sql.Tx, src *sql.DB, summary *importSummary) error {
	rows, err := src.QueryContext(ctx,
		"SELECT id, name, COALESCE(CAST(created_at AS TEXT),''), COALESCE(CAST(updated_at AS TEXT),'') FROM apps ORDER BY id")
	if err != nil {
		return fmt.Errorf("leer apps: %w", err)
	}
	defer rows.Close()

	count := summary.count("apps")
	for rows.Next() {
		var id int64
		var name, created, updated string
		if err := rows.Scan(&id, &name, &created, &updated); err != nil {
			return fmt.Errorf("scan app: %w", err)
		}
		count.Read++
		res, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO automation_apps (id,name,created_at,updated_at)
			 VALUES (?,?,COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP),COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP))`,
			id, name, created, updated)
		if err != nil {
			return fmt.Errorf("insertar app %s: %w", name, err)
		}
		count.Inserted += mustAffected(res)
	}
	return rows.Err()
}

func importAccounts(ctx context.Context, tx *sql.Tx, src *sql.DB, summary *importSummary) error {
	// Stubs para plugin_name referenciados por cuentas pero ausentes del catálogo.
	known := map[string]bool{}
	pluginRows, err := src.QueryContext(ctx, "SELECT name FROM plugins")
	if err != nil {
		return fmt.Errorf("leer nombres de plugins: %w", err)
	}
	for pluginRows.Next() {
		var name string
		if err := pluginRows.Scan(&name); err != nil {
			pluginRows.Close()
			return fmt.Errorf("scan plugin name: %w", err)
		}
		known[name] = true
	}
	pluginRows.Close()

	stubRows, err := src.QueryContext(ctx,
		"SELECT DISTINCT plugin_name FROM accounts WHERE plugin_name IS NOT NULL AND plugin_name <> ''")
	if err != nil {
		return fmt.Errorf("leer plugins de cuentas: %w", err)
	}
	for stubRows.Next() {
		var name string
		if err := stubRows.Scan(&name); err != nil {
			stubRows.Close()
			return fmt.Errorf("scan plugin de cuenta: %w", err)
		}
		if known[name] {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT OR IGNORE INTO automation_plugins (name,version,source,status) VALUES (?,?,?,?)",
			name, "desconocida", "import", "active"); err != nil {
			stubRows.Close()
			return fmt.Errorf("crear stub de plugin %s: %w", name, err)
		}
		known[name] = true
		summary.Stubs = append(summary.Stubs, name)
	}
	if err := stubRows.Err(); err != nil {
		return err
	}
	stubRows.Close()

	rows, err := src.QueryContext(ctx, `
		SELECT id, app_id, identifier, COALESCE(label,''), credentials,
		       COALESCE(CAST(created_at AS TEXT),''), COALESCE(CAST(updated_at AS TEXT),''),
		       COALESCE(plugin_name,''), COALESCE(webhook_url,''), COALESCE(schedule_time,''),
		       COALESCE(schedule_enabled,0), COALESCE(CAST(last_run_at AS TEXT),''),
		       COALESCE(schedule_frequency,'daily'), schedule_interval, COALESCE(schedule_days,'')
		FROM accounts ORDER BY id`)
	if err != nil {
		return fmt.Errorf("leer cuentas: %w", err)
	}
	defer rows.Close()

	count := summary.count("accounts")
	for rows.Next() {
		var (
			id, appID, scheduleEnabled int64
			scheduleInterval           sql.NullInt64
			identifier, label, creds   string
			created, updated           string
			pluginName, webhookURL     string
			scheduleTime, lastRun      string
			frequency, scheduleDays    string
		)
		if err := rows.Scan(&id, &appID, &identifier, &label, &creds, &created, &updated,
			&pluginName, &webhookURL, &scheduleTime, &scheduleEnabled, &lastRun,
			&frequency, &scheduleInterval, &scheduleDays); err != nil {
			return fmt.Errorf("scan cuenta: %w", err)
		}
		count.Read++
		res, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO automation_accounts
			  (id,app_id,identifier,label,credentials,created_at,updated_at,plugin_name,
			   webhook_url,schedule_time,schedule_enabled,last_run_at,schedule_frequency,
			   schedule_interval,schedule_days)
			VALUES (?,?,?,NULLIF(?,''),?,COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP),
			        COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP),NULLIF(?,''),NULLIF(?,''),
			        NULLIF(?,''),?,NULLIF(?,''),?,?,NULLIF(?,''))`,
			id, appID, identifier, label, creds, created, updated, pluginName,
			webhookURL, scheduleTime, scheduleEnabled, lastRun, frequency,
			nullInt(scheduleInterval), scheduleDays)
		if err != nil {
			return fmt.Errorf("insertar cuenta %d: %w", id, err)
		}
		count.Inserted += mustAffected(res)
	}
	return rows.Err()
}

func importBills(ctx context.Context, tx *sql.Tx, src *sql.DB, summary *importSummary) error {
	rows, err := src.QueryContext(ctx, `
		SELECT id, account_id, plugin_name, plugin_version, source, status,
		       COALESCE(error,''), raw, COALESCE(CAST(fetched_at AS TEXT),''), COALESCE(CAST(created_at AS TEXT),'')
		FROM bills ORDER BY id`)
	if err != nil {
		return fmt.Errorf("leer facturas: %w", err)
	}
	defer rows.Close()

	count := summary.count("bills")
	for rows.Next() {
		var id, accountID int64
		var pluginName, pluginVersion, source, status, errText, raw, fetched, created string
		if err := rows.Scan(&id, &accountID, &pluginName, &pluginVersion, &source, &status, &errText, &raw, &fetched, &created); err != nil {
			return fmt.Errorf("scan factura: %w", err)
		}
		count.Read++
		res, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO automation_bills
			  (id,account_id,plugin_name,plugin_version,source,status,error,raw,fetched_at,created_at)
			VALUES (?,?,?,?,?,?,NULLIF(?,''),?,COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP),COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP))`,
			id, accountID, pluginName, pluginVersion, source, status, errText, raw, fetched, created)
		if err != nil {
			return fmt.Errorf("insertar factura %d: %w", id, err)
		}
		count.Inserted += mustAffected(res)
	}
	return rows.Err()
}

func importWebhookLogs(ctx context.Context, tx *sql.Tx, src *sql.DB, summary *importSummary) error {
	rows, err := src.QueryContext(ctx, `
		SELECT id, account_id, webhook_url, status, payload, response,
		       COALESCE(CAST(ran_at AS TEXT),''), http_status
		FROM webhook_logs ORDER BY id`)
	if err != nil {
		return fmt.Errorf("leer logs de webhook: %w", err)
	}
	defer rows.Close()

	count := summary.count("webhook_logs")
	for rows.Next() {
		var id, accountID int64
		var webhookURL, status, payload, response, ranAt string
		var httpStatus sql.NullInt64
		if err := rows.Scan(&id, &accountID, &webhookURL, &status, &payload, &response, &ranAt, &httpStatus); err != nil {
			return fmt.Errorf("scan log de webhook: %w", err)
		}
		count.Read++
		res, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO automation_webhook_logs
			  (id,account_id,webhook_url,status,payload,response,ran_at,http_status)
			VALUES (?,?,?,?,?,?,COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP),?)`,
			id, accountID, webhookURL, status, payload, response, ranAt, nullInt(httpStatus))
		if err != nil {
			return fmt.Errorf("insertar log de webhook %d: %w", id, err)
		}
		count.Inserted += mustAffected(res)
	}
	return rows.Err()
}

// importSettings migra solo webhook_enabled y language. NO toca webhook_api_key
// de p40la (ADR-003) ni auth_cookie_secret.
func importSettings(ctx context.Context, tx *sql.Tx, src *sql.DB, summary *importSummary) error {
	rows, err := src.QueryContext(ctx, "SELECT key, value FROM settings WHERE key IN ('webhook_enabled','language')")
	if err != nil {
		return fmt.Errorf("leer settings: %w", err)
	}
	defer rows.Close()

	count := summary.count("settings")
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return fmt.Errorf("scan setting: %w", err)
		}
		count.Read++
		var res sql.Result
		if key == "webhook_enabled" {
			res, err = tx.ExecContext(ctx,
				`INSERT INTO system_settings (key,value) VALUES (?,?)
				 ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=CURRENT_TIMESTAMP`,
				key, value)
		} else {
			res, err = tx.ExecContext(ctx,
				"INSERT OR IGNORE INTO system_settings (key,value) VALUES (?,?)", key, value)
		}
		if err != nil {
			return fmt.Errorf("insertar setting %s: %w", key, err)
		}
		count.Inserted += mustAffected(res)
	}
	return rows.Err()
}

func collectOrphans(ctx context.Context, dst *sql.DB, summary *importSummary) error {
	rows, err := dst.QueryContext(ctx, `
		SELECT DISTINCT automation_account_id FROM services
		WHERE automation_account_id IS NOT NULL
		  AND automation_account_id NOT IN (SELECT id FROM automation_accounts)
		ORDER BY automation_account_id`)
	if err != nil {
		return fmt.Errorf("buscar huérfanos: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan huérfano: %w", err)
		}
		summary.Orphans = append(summary.Orphans, id)
	}
	return rows.Err()
}

func checkIntegrity(ctx context.Context, dst *sql.DB) error {
	var integrity string
	if err := dst.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity); err != nil {
		return fmt.Errorf("integrity_check: %w", err)
	}
	if integrity != "ok" {
		return fmt.Errorf("integrity_check falló: %s", integrity)
	}

	rows, err := dst.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("foreign_key_check: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("foreign_key_check encontró violaciones")
	}
	return rows.Err()
}

func verifyCounts(ctx context.Context, src, dst *sql.DB, summary *importSummary) error {
	pairs := map[string]string{
		"automation_apps":         "apps",
		"automation_accounts":     "accounts",
		"automation_bills":        "bills",
		"automation_webhook_logs": "webhook_logs",
	}
	for target, source := range pairs {
		var srcCount, dstCount int64
		if err := src.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+source).Scan(&srcCount); err != nil {
			return fmt.Errorf("contar origen %s: %w", source, err)
		}
		if err := dst.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+target).Scan(&dstCount); err != nil {
			return fmt.Errorf("contar destino %s: %w", target, err)
		}
		if srcCount != dstCount {
			return fmt.Errorf("verify: %s origen=%d destino=%d", target, srcCount, dstCount)
		}
		fmt.Printf("verify: %s=%d ok\n", target, dstCount)
	}

	// En plugins el destino puede tener stubs de plugins ausentes (REQ-013):
	// se verifica que cada plugin del origen exista, no la igualdad de conteos.
	rows, err := src.QueryContext(ctx, "SELECT name FROM plugins ORDER BY name")
	if err != nil {
		return fmt.Errorf("verify: leer plugins de origen: %w", err)
	}
	defer rows.Close()
	verified := 0
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("verify: scan plugin: %w", err)
		}
		var n int64
		if err := dst.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM automation_plugins WHERE name = ?", name).Scan(&n); err != nil {
			return fmt.Errorf("verify: contar plugin %s: %w", name, err)
		}
		if n == 0 {
			return fmt.Errorf("verify: plugin %s no está en el destino", name)
		}
		verified++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	fmt.Printf("verify: automation_plugins=%d ok (%d stubs)\n", verified, len(summary.Stubs))
	return nil
}

func mustAffected(res sql.Result) int64 {
	n, err := res.RowsAffected()
	if err != nil {
		return 0
	}
	return n
}

func nullInt(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}
