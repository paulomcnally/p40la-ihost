CREATE TABLE IF NOT EXISTS services (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    home_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    institution TEXT,
    currency_id INTEGER NOT NULL,
    frequency TEXT NOT NULL CHECK (frequency IN ('monthly', 'yearly')),
    suggested_amount REAL NOT NULL,
    active BOOLEAN DEFAULT 1,
    icon_key TEXT NOT NULL,
    deleted_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (home_id) REFERENCES homes(id),
    FOREIGN KEY (currency_id) REFERENCES currencies(id)
);

CREATE INDEX IF NOT EXISTS idx_services_home_id ON services(home_id);
CREATE INDEX IF NOT EXISTS idx_services_deleted_at ON services(deleted_at);
