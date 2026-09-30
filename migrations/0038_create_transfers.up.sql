-- SPEC-099: Transferencias entre cuentas del presupuesto.
-- Una transferencia mueve dinero de una cuenta (origen) a otra (destino)
-- sin afectar la actividad ni los ingresos del presupuesto (es neutra).

CREATE TABLE IF NOT EXISTS transfers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_account_id INTEGER NOT NULL,
    to_account_id INTEGER NOT NULL,
    currency_id INTEGER NOT NULL,
    date TEXT NOT NULL,
    payee TEXT,
    memo TEXT,
    amount REAL NOT NULL DEFAULT 0,
    cleared INTEGER NOT NULL DEFAULT 0,
    deleted_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_account_id) REFERENCES accounts(id),
    FOREIGN KEY (to_account_id) REFERENCES accounts(id),
    FOREIGN KEY (currency_id) REFERENCES currencies(id),
    CHECK (from_account_id != to_account_id),
    CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_transfers_from_account ON transfers(from_account_id);
CREATE INDEX IF NOT EXISTS idx_transfers_to_account ON transfers(to_account_id);
CREATE INDEX IF NOT EXISTS idx_transfers_date ON transfers(date);