-- SPEC-133: bancos, cuentas bancarias y formatos de transferencia.
-- Los formatos guardan un JSON de pares llave/valor en fields_json.

CREATE TABLE IF NOT EXISTS banks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    short_code TEXT NOT NULL DEFAULT '',
    country TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bank_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bank_id INTEGER NOT NULL,
    alias TEXT NOT NULL,
    account_number TEXT NOT NULL,
    account_type TEXT NOT NULL DEFAULT 'savings' CHECK (account_type IN ('savings', 'checking')),
    currency_id INTEGER NOT NULL,
    beneficiary TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bank_id) REFERENCES banks(id) ON DELETE RESTRICT,
    FOREIGN KEY (currency_id) REFERENCES currencies(id),
    UNIQUE(bank_id, account_number)
);

CREATE INDEX IF NOT EXISTS idx_bank_accounts_bank_id ON bank_accounts(bank_id);

CREATE TABLE IF NOT EXISTS bank_transfer_formats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bank_account_id INTEGER NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('transfer', 'ach')),
    fields_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bank_account_id) REFERENCES bank_accounts(id) ON DELETE CASCADE,
    UNIQUE(bank_account_id, kind)
);

CREATE INDEX IF NOT EXISTS idx_bank_transfer_formats_account_id
    ON bank_transfer_formats(bank_account_id);
