ALTER TABLE services ADD COLUMN billing_type TEXT NOT NULL DEFAULT 'variable' CHECK (billing_type IN ('fixed', 'variable'));
ALTER TABLE services ADD COLUMN billing_day INTEGER DEFAULT 1 CHECK (billing_day BETWEEN 1 AND 31);
ALTER TABLE services ADD COLUMN auto_generate BOOLEAN DEFAULT 0;

CREATE TABLE IF NOT EXISTS system_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO system_settings (key, value) VALUES ('billing_generation_hour', '0');
