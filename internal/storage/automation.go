package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/paulomcnally/p40la-ihost/internal/models"
)

// AutomationStorage encapsula las consultas de las tablas automation_*
// (SPEC-109) dentro de la DB de p40la.
type AutomationStorage struct {
	db *sql.DB
}

func NewAutomationStorage(db *sql.DB) *AutomationStorage {
	return &AutomationStorage{db: db}
}

// automationAccountColumns omite credentials a propósito: nunca se devuelve en
// listados ni en el detalle HTTP.
const automationAccountColumns = `
	a.id, a.institution_id, COALESCE(i.name,''), a.identifier, COALESCE(a.label,''),
	COALESCE(a.plugin_name,''), COALESCE(a.webhook_url,''), COALESCE(a.schedule_time,''),
	a.schedule_enabled, a.last_run_at, COALESCE(a.schedule_frequency,'daily'),
	a.schedule_interval, COALESCE(a.schedule_days,''), a.created_at, a.updated_at
`

func (s *AutomationStorage) ListAccounts(ctx context.Context) ([]models.AutomationAccount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+automationAccountColumns+`
		FROM automation_accounts a
		LEFT JOIN institutions i ON i.id = a.institution_id
		ORDER BY COALESCE(i.name,''), a.identifier`)
	if err != nil {
		return nil, fmt.Errorf("listar cuentas de automation: %w", err)
	}
	defer rows.Close()

	var accounts []models.AutomationAccount
	for rows.Next() {
		account, err := scanAutomationAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, *account)
	}
	return accounts, rows.Err()
}

func (s *AutomationStorage) GetAccount(ctx context.Context, id int64) (*models.AutomationAccount, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+automationAccountColumns+`
		FROM automation_accounts a
		LEFT JOIN institutions i ON i.id = a.institution_id
		WHERE a.id = ?`, id)
	account, err := scanAutomationAccount(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return account, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAutomationAccount(row rowScanner) (*models.AutomationAccount, error) {
	var (
		account          models.AutomationAccount
		pluginName       string
		webhookURL       string
		scheduleTime     string
		scheduleDays     string
		lastRunAt        sql.NullTime
		scheduleInterval sql.NullInt64
	)
	if err := row.Scan(
		&account.ID, &account.InstitutionID, &account.InstitutionName, &account.Identifier,
		&account.Label, &pluginName, &webhookURL, &scheduleTime, &account.ScheduleEnabled,
		&lastRunAt, &account.ScheduleFrequency, &scheduleInterval, &scheduleDays,
		&account.CreatedAt, &account.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan cuenta de automation: %w", err)
	}
	if pluginName != "" {
		account.PluginName = &pluginName
	}
	if webhookURL != "" {
		account.WebhookURL = &webhookURL
	}
	if scheduleTime != "" {
		account.ScheduleTime = &scheduleTime
	}
	if scheduleDays != "" {
		account.ScheduleDays = &scheduleDays
	}
	if scheduleInterval.Valid {
		v := int(scheduleInterval.Int64)
		account.ScheduleInterval = &v
	}
	if lastRunAt.Valid {
		account.LastRunAt = &lastRunAt.Time
	}
	return &account, nil
}

func (s *AutomationStorage) ListPlugins(ctx context.Context) ([]models.AutomationPlugin, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, version, source, COALESCE(description,''), status, created_at, updated_at
		FROM automation_plugins ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listar plugins de automation: %w", err)
	}
	defer rows.Close()

	var plugins []models.AutomationPlugin
	for rows.Next() {
		var p models.AutomationPlugin
		if err := rows.Scan(&p.ID, &p.Name, &p.Version, &p.Source, &p.Description,
			&p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan plugin: %w", err)
		}
		plugins = append(plugins, p)
	}
	return plugins, rows.Err()
}

func (s *AutomationStorage) AccountExists(ctx context.Context, id int64) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx,
		"SELECT 1 FROM automation_accounts WHERE id = ?", id).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("verificar cuenta de automation: %w", err)
	}
	return true, nil
}

func (s *AutomationStorage) ListBills(ctx context.Context, accountID int64) ([]models.AutomationBill, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, plugin_name, plugin_version, source, status,
		       error, fetched_at, created_at
		FROM automation_bills WHERE account_id = ? ORDER BY fetched_at DESC, id DESC`, accountID)
	if err != nil {
		return nil, fmt.Errorf("listar facturas de automation: %w", err)
	}
	defer rows.Close()

	var bills []models.AutomationBill
	for rows.Next() {
		var (
			b         models.AutomationBill
			errorText sql.NullString
		)
		if err := rows.Scan(&b.ID, &b.AccountID, &b.PluginName, &b.PluginVersion,
			&b.Source, &b.Status, &errorText, &b.FetchedAt, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan factura de automation: %w", err)
		}
		if errorText.Valid {
			b.Error = &errorText.String
		}
		bills = append(bills, b)
	}
	return bills, rows.Err()
}

func (s *AutomationStorage) ListWebhookLogs(ctx context.Context, accountID int64, limit, offset int) ([]models.AutomationWebhookLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, account_id, webhook_url, status, COALESCE(response,''), ran_at, http_status
		FROM automation_webhook_logs WHERE account_id = ?
		ORDER BY ran_at DESC, id DESC LIMIT ? OFFSET ?`, accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listar logs de webhook: %w", err)
	}
	defer rows.Close()

	var logs []models.AutomationWebhookLog
	for rows.Next() {
		var (
			l          models.AutomationWebhookLog
			httpStatus sql.NullInt64
			ranAt      time.Time
		)
		if err := rows.Scan(&l.ID, &l.AccountID, &l.WebhookURL, &l.Status,
			&l.Response, &ranAt, &httpStatus); err != nil {
			return nil, fmt.Errorf("scan log de webhook: %w", err)
		}
		l.RanAt = ranAt
		if httpStatus.Valid {
			v := int(httpStatus.Int64)
			l.HTTPStatus = &v
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// InstitutionExists verifica que la institución exista.
func (s *AutomationStorage) InstitutionExists(ctx context.Context, id int64) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM institutions WHERE id = ?", id).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("verificar institución: %w", err)
	}
	return true, nil
}

// PluginExists verifica que el plugin esté en el catálogo.
func (s *AutomationStorage) PluginExists(ctx context.Context, name string) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM automation_plugins WHERE name = ?", name).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("verificar plugin: %w", err)
	}
	return true, nil
}

// CreateAccount inserta una cuenta y devuelve la fila resultante.
func (s *AutomationStorage) CreateAccount(ctx context.Context, account *models.AutomationAccount, credentials string) (*models.AutomationAccount, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO automation_accounts
		  (institution_id, identifier, label, credentials, plugin_name, webhook_url,
		   schedule_time, schedule_enabled, schedule_frequency, schedule_interval, schedule_days)
		VALUES (?,?,NULLIF(?,''),?,NULLIF(?,''),NULLIF(?,''),NULLIF(?,''),?,?,?,NULLIF(?,''))`,
		account.InstitutionID, account.Identifier, account.Label, credentials,
		nullStringPtr(account.PluginName), nullStringPtr(account.WebhookURL),
		nullStringPtr(account.ScheduleTime), account.ScheduleEnabled,
		account.ScheduleFrequency, nullIntPtr(account.ScheduleInterval), nullStringPtr(account.ScheduleDays))
	if err != nil {
		return nil, fmt.Errorf("crear cuenta de automation: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("id de cuenta de automation: %w", err)
	}
	return s.GetAccount(ctx, id)
}

// UpdateAccount actualiza los campos editables. Si credentials es vacío, conserva
// las credenciales existentes.
func (s *AutomationStorage) UpdateAccount(ctx context.Context, account *models.AutomationAccount, credentials string) (*models.AutomationAccount, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE automation_accounts SET
		  institution_id = ?, identifier = ?, label = NULLIF(?,''),
		  plugin_name = NULLIF(?,''), webhook_url = NULLIF(?,''),
		  schedule_time = NULLIF(?,''), schedule_enabled = ?,
		  schedule_frequency = ?, schedule_interval = ?, schedule_days = NULLIF(?,''),
		  updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		account.InstitutionID, account.Identifier, account.Label,
		nullStringPtr(account.PluginName), nullStringPtr(account.WebhookURL),
		nullStringPtr(account.ScheduleTime), account.ScheduleEnabled,
		account.ScheduleFrequency, nullIntPtr(account.ScheduleInterval),
		nullStringPtr(account.ScheduleDays), account.ID)
	if err != nil {
		return nil, fmt.Errorf("actualizar cuenta de automation: %w", err)
	}
	if credentials != "" {
		if _, err := s.db.ExecContext(ctx,
			"UPDATE automation_accounts SET credentials = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			credentials, account.ID); err != nil {
			return nil, fmt.Errorf("actualizar credenciales: %w", err)
		}
	}
	return s.GetAccount(ctx, account.ID)
}

// DeleteAccount elimina una cuenta.
func (s *AutomationStorage) DeleteAccount(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM automation_accounts WHERE id = ?", id); err != nil {
		return fmt.Errorf("eliminar cuenta de automation: %w", err)
	}
	return nil
}

// CountServicesUsingAccount cuenta servicios activos vinculados a la cuenta.
func (s *AutomationStorage) CountServicesUsingAccount(ctx context.Context, id int64) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM services WHERE automation_account_id = ? AND deleted_at IS NULL", id,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("contar servicios de la cuenta: %w", err)
	}
	return count, nil
}

// CredentialsForAccount devuelve las credenciales JSON de una cuenta.
func (s *AutomationStorage) CredentialsForAccount(ctx context.Context, id int64) (string, error) {
	var credentials string
	err := s.db.QueryRowContext(ctx,
		"SELECT credentials FROM automation_accounts WHERE id = ?", id).Scan(&credentials)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("obtener credenciales de la cuenta: %w", err)
	}
	return credentials, nil
}

func nullStringPtr(v *string) any {
	if v == nil || *v == "" {
		return nil
	}
	return *v
}

func nullIntPtr(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}
