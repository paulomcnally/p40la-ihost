// Comando offline de importación de p40la-ihost-automation hacia la DB de
// p40la-ihost (SPEC-109). Se ejecuta con la aplicación detenida y sobre una
// copia de la DB origen hecha con `sqlite3 .backup` (nunca copia plana: se
// perdería el contenido del -wal).
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/paulomcnally/p40la-ihost/internal/db"
)

// tableCount acumula filas leídas e insertadas por tabla.
type tableCount struct {
	Read     int64
	Inserted int64
}

// institutionStats resume la resolución de apps → institutions.
type institutionStats struct {
	Apps         int64
	Reused       int64
	Created      int64
	Mapped       int64
	CreatedNames []string
	MappedNames  []string
}

// importSummary es el resultado de una corrida (real o dry-run).
type importSummary struct {
	Tables       map[string]*tableCount
	Institutions institutionStats
	Orphans      []int64
	Stubs        []string
	Warnings     []string
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
	if s.Institutions.Apps > 0 {
		fmt.Printf("instituciones: apps=%d reutilizadas=%d creadas=%d mapeadas=%d\n",
			s.Institutions.Apps, s.Institutions.Reused, s.Institutions.Created, s.Institutions.Mapped)
	}
	for _, name := range s.Institutions.CreatedNames {
		fmt.Printf("aviso: institución creada desde app (revisar duplicados): %s\n", name)
	}
	for _, name := range s.Institutions.MappedNames {
		fmt.Printf("aviso: app mapeada por --institution-map: %s\n", name)
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
	sourcePath         string
	sourceJSONPath     string
	dataDir            string
	migrationsDir      string
	institutionMapPath string
	dryRun             bool
	verify             bool
}

func main() {
	var opts options
	flag.StringVar(&opts.sourcePath, "from", "", "ruta de la copia de la DB de automation")
	flag.StringVar(&opts.sourceJSONPath, "from-json", "", "ruta del export JSON de p40la-ihost-automation (alternativa a --from)")
	flag.StringVar(&opts.dataDir, "data-dir", defaultDataDir(), "directorio de datos de p40la (default: DATA_DIR o ./data)")
	flag.StringVar(&opts.migrationsDir, "migrations", "./migrations", "directorio de migraciones de p40la")
	flag.StringVar(&opts.institutionMapPath, "institution-map", "", "JSON opcional para mapear app → institución (nombre o ID)")
	flag.BoolVar(&opts.dryRun, "dry-run", false, "validar y reportar sin escribir datos")
	flag.BoolVar(&opts.verify, "verify", false, "comparar conteos origen/destino al finalizar")
	flag.Parse()

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
	hasDB := strings.TrimSpace(opts.sourcePath) != ""
	hasJSON := strings.TrimSpace(opts.sourceJSONPath) != ""
	if hasDB == hasJSON {
		return errors.New("usar exactamente uno de --from o --from-json")
	}

	if hasJSON {
		dbPath, cleanup, err := materializeJSONSource(opts.sourceJSONPath)
		if err != nil {
			return err
		}
		defer cleanup()
		opts.sourcePath = dbPath
		fmt.Println("origen: export JSON materializado en SQLite temporal")
	}

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

	overrides, err := loadInstitutionMap(opts.institutionMapPath)
	if err != nil {
		return err
	}

	dst, err := db.OpenDB(targetPath, opts.migrationsDir)
	if err != nil {
		return fmt.Errorf("abrir DB destino: %w", err)
	}
	defer dst.Close()

	if err := importAll(ctx, src, dst, overrides, summary); err != nil {
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
	summary.Institutions.Apps = summary.count("apps").Read

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

func importAll(ctx context.Context, src, dst *sql.DB, overrides map[string]string, summary *importSummary) error {
	tx, err := dst.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transacción: %w", err)
	}
	defer tx.Rollback()

	if err := importPlugins(ctx, tx, src, summary); err != nil {
		return err
	}
	appToInstitution, err := importInstitutions(ctx, tx, src, overrides, summary)
	if err != nil {
		return err
	}
	if err := importAccounts(ctx, tx, src, appToInstitution, summary); err != nil {
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

// importInstitutions resuelve cada app de automation contra la tabla
// institutions existente (SPEC-109 REQ-014/REQ-015) y devuelve el mapa
// app_id → institution_id que usa importAccounts.
func importInstitutions(ctx context.Context, tx *sql.Tx, src *sql.DB, overrides map[string]string, summary *importSummary) (map[int64]int64, error) {
	apps, err := readApps(ctx, src)
	if err != nil {
		return nil, err
	}

	byID, exact, normalized, err := loadInstitutions(ctx, tx)
	if err != nil {
		return nil, err
	}
	appToInstitution := make(map[int64]int64, len(apps))

	for _, app := range apps {
		summary.Institutions.Apps++

		if override, ok := overrides[app.Name]; ok {
			institutionID, err := resolveOverride(override, byID, exact, normalized)
			if err != nil {
				return nil, fmt.Errorf("institution-map para %q: %w", app.Name, err)
			}
			appToInstitution[app.ID] = institutionID
			summary.Institutions.Mapped++
			summary.Institutions.MappedNames = append(summary.Institutions.MappedNames,
				fmt.Sprintf("%s → %s", app.Name, byID[institutionID]))
			continue
		}

		if id, ok := exact[app.Name]; ok {
			appToInstitution[app.ID] = id
			summary.Institutions.Reused++
			continue
		}
		if id, ok := normalized[normalizeInstitutionName(app.Name)]; ok {
			appToInstitution[app.ID] = id
			summary.Institutions.Reused++
			continue
		}

		res, err := tx.ExecContext(ctx, "INSERT INTO institutions (name) VALUES (?)", app.Name)
		if err != nil {
			return nil, fmt.Errorf("crear institución %q: %w", app.Name, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("id de institución %q: %w", app.Name, err)
		}
		byID[id] = app.Name
		exact[app.Name] = id
		normalized[normalizeInstitutionName(app.Name)] = id
		appToInstitution[app.ID] = id
		summary.Institutions.Created++
		summary.Institutions.CreatedNames = append(summary.Institutions.CreatedNames, app.Name)
	}

	return appToInstitution, nil
}

type automationApp struct {
	ID   int64
	Name string
}

func readApps(ctx context.Context, src *sql.DB) ([]automationApp, error) {
	rows, err := src.QueryContext(ctx, "SELECT id, name FROM apps ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("leer apps: %w", err)
	}
	defer rows.Close()

	var apps []automationApp
	for rows.Next() {
		var app automationApp
		if err := rows.Scan(&app.ID, &app.Name); err != nil {
			return nil, fmt.Errorf("scan app: %w", err)
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

func loadInstitutions(ctx context.Context, tx *sql.Tx) (byID map[int64]string, exact, normalized map[string]int64, err error) {
	byID = make(map[int64]string)
	exact = make(map[string]int64)
	normalized = make(map[string]int64)

	rows, err := tx.QueryContext(ctx, "SELECT id, name FROM institutions")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("leer instituciones: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, nil, nil, fmt.Errorf("scan institución: %w", err)
		}
		byID[id] = name
		exact[name] = id
		key := normalizeInstitutionName(name)
		if _, exists := normalized[key]; !exists {
			normalized[key] = id
		}
	}
	return byID, exact, normalized, rows.Err()
}

// resolveOverride resuelve el valor del mapa (ID o nombre) contra las
// instituciones existentes. Si no existe, falla (REQ-015): el operador debe
// mapear a una institución real, no crear duplicados silenciosos.
func resolveOverride(value string, byID map[int64]string, exact, normalized map[string]int64) (int64, error) {
	if id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
		if _, ok := byID[id]; !ok {
			return 0, fmt.Errorf("no existe la institución con ID %d", id)
		}
		return id, nil
	}
	if id, ok := exact[value]; ok {
		return id, nil
	}
	if id, ok := normalized[normalizeInstitutionName(value)]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("no existe la institución %q (creála en p40la o corregí el mapa)", value)
}

var accentReplacer = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u", "Ü", "u", "Ñ", "n",
)

// normalizeInstitutionName unifica mayúsculas, acentos, separadores y el
// sufijo "Nicaragua" para comparar apps de automation con institutions.
func normalizeInstitutionName(name string) string {
	name = accentReplacer.Replace(strings.TrimSpace(strings.ToLower(name)))
	name = strings.NewReplacer("-", " ", "_", " ", ".", " ", ",", " ").Replace(name)
	name = strings.Join(strings.Fields(name), " ")
	name = strings.TrimSuffix(name, " de nicaragua")
	name = strings.TrimSuffix(name, " nicaragua")
	return strings.TrimSpace(name)
}

func loadInstitutionMap(path string) (map[string]string, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("leer institution-map: %w", err)
	}
	var overrides map[string]string
	if err := json.Unmarshal(data, &overrides); err != nil {
		return nil, fmt.Errorf("parsear institution-map: %w", err)
	}
	return overrides, nil
}

func importAccounts(ctx context.Context, tx *sql.Tx, src *sql.DB, appToInstitution map[int64]int64, summary *importSummary) error {
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
		institutionID, ok := appToInstitution[appID]
		if !ok {
			return fmt.Errorf("cuenta %d referencia app %d sin institución resuelta", id, appID)
		}
		count.Read++
		res, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO automation_accounts
			  (id,institution_id,identifier,label,credentials,created_at,updated_at,plugin_name,
			   webhook_url,schedule_time,schedule_enabled,last_run_at,schedule_frequency,
			   schedule_interval,schedule_days)
			VALUES (?,?,?,NULLIF(?,''),?,COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP),
			        COALESCE(NULLIF(?,''),CURRENT_TIMESTAMP),NULLIF(?,''),NULLIF(?,''),
			        NULLIF(?,''),?,NULLIF(?,''),?,?,NULLIF(?,''))`,
			id, institutionID, identifier, label, creds, created, updated, pluginName,
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

	// Toda cuenta debe apuntar a una institución existente.
	var orphanAccounts int64
	if err := dst.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM automation_accounts a
		LEFT JOIN institutions i ON i.id = a.institution_id
		WHERE i.id IS NULL`).Scan(&orphanAccounts); err != nil {
		return fmt.Errorf("verify: cuentas sin institución: %w", err)
	}
	if orphanAccounts != 0 {
		return fmt.Errorf("verify: %d cuentas sin institución", orphanAccounts)
	}
	fmt.Println("verify: automation_accounts.institution_id ok")
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
