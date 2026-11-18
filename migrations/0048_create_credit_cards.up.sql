-- SPEC-137: tarjetas de crédito, estados de cuenta y movimientos.
-- Montos en REAL (convención del proyecto). Signo: positivo = cargo, negativo = abono.
-- Fechas de negocio en TEXT ISO (YYYY-MM-DD).

CREATE TABLE IF NOT EXISTS credit_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bank_id INTEGER NOT NULL,
    alias TEXT NOT NULL COLLATE NOCASE,
    network TEXT NOT NULL CHECK (network IN ('visa', 'mastercard', 'amex', 'diners', 'other')),
    last4 TEXT NOT NULL CHECK (last4 GLOB '[0-9][0-9][0-9][0-9]'),
    holder TEXT NOT NULL DEFAULT '',
    currency_id INTEGER NOT NULL,
    secondary_currency_id INTEGER,
    credit_limit REAL NOT NULL DEFAULT 0,
    credit_limit_usd REAL NOT NULL DEFAULT 0,
    statement_day INTEGER NOT NULL DEFAULT 1 CHECK (statement_day BETWEEN 1 AND 31),
    payment_due_day INTEGER NOT NULL DEFAULT 1 CHECK (payment_due_day BETWEEN 1 AND 31),
    min_payment_pct REAL NOT NULL DEFAULT 4 CHECK (min_payment_pct BETWEEN 0 AND 100),
    interest_current_rate REAL NOT NULL DEFAULT 0 CHECK (interest_current_rate >= 0),
    interest_late_rate REAL NOT NULL DEFAULT 0 CHECK (interest_late_rate >= 0),
    interest_current_rate_usd REAL NOT NULL DEFAULT 0 CHECK (interest_current_rate_usd >= 0),
    interest_late_rate_usd REAL NOT NULL DEFAULT 0 CHECK (interest_late_rate_usd >= 0),
    color TEXT NOT NULL DEFAULT 'blue',
    icon TEXT NOT NULL DEFAULT 'credit',
    active INTEGER NOT NULL DEFAULT 1,
    notes TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bank_id) REFERENCES banks(id) ON DELETE RESTRICT,
    FOREIGN KEY (currency_id) REFERENCES currencies(id),
    FOREIGN KEY (secondary_currency_id) REFERENCES currencies(id),
    UNIQUE (bank_id, alias),
    CHECK (secondary_currency_id IS NULL OR secondary_currency_id <> currency_id)
);

CREATE INDEX IF NOT EXISTS idx_credit_cards_bank ON credit_cards(bank_id);

CREATE TABLE IF NOT EXISTS card_statements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL,
    period_start TEXT,
    period_end TEXT NOT NULL,
    min_payment_due_date TEXT NOT NULL,
    full_payment_due_date TEXT NOT NULL,
    previous_balance REAL NOT NULL DEFAULT 0,
    previous_balance_usd REAL NOT NULL DEFAULT 0,
    interest_current REAL NOT NULL DEFAULT 0,
    interest_current_usd REAL NOT NULL DEFAULT 0,
    min_payment REAL NOT NULL DEFAULT 0,
    min_payment_usd REAL NOT NULL DEFAULT 0,
    total_payment REAL NOT NULL DEFAULT 0,
    total_payment_usd REAL NOT NULL DEFAULT 0,
    new_balance REAL NOT NULL DEFAULT 0,
    new_balance_usd REAL NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'unpaid' CHECK (status IN ('unpaid', 'partial', 'paid')),
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'import')),
    source_module_id TEXT NOT NULL DEFAULT '',
    import_hash TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (card_id) REFERENCES credit_cards(id) ON DELETE CASCADE,
    UNIQUE (card_id, period_end)
);

CREATE INDEX IF NOT EXISTS idx_card_statements_card
    ON card_statements(card_id, period_end DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_card_statements_import
    ON card_statements(card_id, import_hash) WHERE import_hash <> '';

CREATE TABLE IF NOT EXISTS card_transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL,
    statement_id INTEGER,
    posted_at TEXT NOT NULL,
    concept TEXT NOT NULL,
    category_id INTEGER,
    type TEXT NOT NULL CHECK (type IN (
        'purchase', 'payment', 'payment_bonus', 'cash_advance', 'cash_advance_fee',
        'interest_current', 'interest_late', 'fee', 'tax', 'bonus', 'refund',
        'adjustment', 'other')),
    amount_local REAL NOT NULL DEFAULT 0,
    amount_usd REAL NOT NULL DEFAULT 0,
    currency_original TEXT NOT NULL DEFAULT 'NIO' CHECK (currency_original IN ('NIO', 'USD')),
    status TEXT NOT NULL DEFAULT 'posted' CHECK (status IN ('posted', 'pending')),
    external_ref TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (card_id) REFERENCES credit_cards(id) ON DELETE CASCADE,
    FOREIGN KEY (statement_id) REFERENCES card_statements(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_card_transactions_card
    ON card_transactions(card_id, posted_at DESC);

CREATE INDEX IF NOT EXISTS idx_card_transactions_statement
    ON card_transactions(statement_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_card_transactions_ref
    ON card_transactions(card_id, external_ref) WHERE external_ref <> '';
