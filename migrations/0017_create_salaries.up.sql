CREATE TABLE IF NOT EXISTS salaries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    employer TEXT NOT NULL,
    amount REAL NOT NULL,
    currency_id INTEGER NOT NULL,
    payment_day INTEGER NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    note TEXT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (currency_id) REFERENCES currencies(id)
);

CREATE INDEX IF NOT EXISTS idx_salaries_currency_id ON salaries(currency_id);