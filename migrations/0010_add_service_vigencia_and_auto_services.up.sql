ALTER TABLE services ADD COLUMN start_date TEXT;
ALTER TABLE services ADD COLUMN end_date TEXT;
ALTER TABLE services ADD COLUMN is_recurring BOOLEAN DEFAULT 0;

CREATE TABLE IF NOT EXISTS auto_services (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auto_id INTEGER NOT NULL,
    service_id INTEGER NOT NULL,
    coverage_type TEXT NOT NULL CHECK (coverage_type IN ('daños_a_terceros', 'full_cover')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (auto_id) REFERENCES autos(id),
    FOREIGN KEY (service_id) REFERENCES services(id),
    UNIQUE(auto_id, service_id)
);
