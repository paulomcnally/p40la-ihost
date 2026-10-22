-- SPEC-109: unificación de automation dentro de p40la-ihost.
-- Crea las tablas con prefijo automation_ para evitar colisiones con las
-- tablas existentes (bills, users, settings). Los datos se importan con
-- cmd/import-automation (offline e idempotente).

CREATE TABLE IF NOT EXISTS automation_apps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS automation_plugins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    version TEXT NOT NULL,
    source TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS automation_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    institution_id INTEGER NOT NULL REFERENCES institutions(id),
    identifier TEXT NOT NULL,
    label TEXT,
    credentials TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    plugin_name TEXT REFERENCES automation_plugins(name),
    webhook_url TEXT,
    schedule_time TEXT,
    schedule_enabled INTEGER NOT NULL DEFAULT 0,
    last_run_at DATETIME,
    schedule_frequency TEXT NOT NULL DEFAULT 'daily',
    schedule_interval INTEGER,
    schedule_days TEXT,
    UNIQUE(institution_id, identifier)
);

CREATE INDEX IF NOT EXISTS idx_automation_accounts_institution_id ON automation_accounts(institution_id);

CREATE TABLE IF NOT EXISTS automation_bills (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL REFERENCES automation_accounts(id) ON DELETE CASCADE,
    plugin_name TEXT NOT NULL,
    plugin_version TEXT NOT NULL,
    source TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ok',
    error TEXT,
    raw TEXT NOT NULL,
    fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_automation_bills_account_id ON automation_bills(account_id);

CREATE TABLE IF NOT EXISTS automation_webhook_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL REFERENCES automation_accounts(id) ON DELETE CASCADE,
    webhook_url TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ok',
    payload TEXT NOT NULL,
    response TEXT NOT NULL,
    ran_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    http_status INTEGER
);

CREATE INDEX IF NOT EXISTS idx_automation_webhook_logs_account_id ON automation_webhook_logs(account_id);

-- Acelera el selector de cuentas y el join desde services (SPEC-111).
CREATE INDEX IF NOT EXISTS idx_services_automation_account_id ON services(automation_account_id);
