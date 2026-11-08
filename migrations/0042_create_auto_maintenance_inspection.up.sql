CREATE TABLE IF NOT EXISTS auto_maintenances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auto_id INTEGER NOT NULL,
    performed_at TEXT NOT NULL,
    mileage INTEGER NOT NULL CHECK (mileage >= 0),
    institution_id INTEGER,
    workshop_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    labor_cost REAL NOT NULL DEFAULT 0 CHECK (labor_cost >= 0),
    supplies_cost REAL NOT NULL DEFAULT 0 CHECK (supplies_cost >= 0),
    currency_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (auto_id) REFERENCES autos(id) ON DELETE CASCADE,
    FOREIGN KEY (institution_id) REFERENCES institutions(id) ON DELETE SET NULL,
    FOREIGN KEY (currency_id) REFERENCES currencies(id)
);

CREATE INDEX IF NOT EXISTS idx_auto_maintenances_auto_date
    ON auto_maintenances(auto_id, performed_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_auto_maintenances_institution
    ON auto_maintenances(institution_id);

CREATE TABLE IF NOT EXISTS auto_inspections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auto_id INTEGER NOT NULL,
    performed_at TEXT NOT NULL,
    inspection_year INTEGER NOT NULL,
    institution_id INTEGER NOT NULL,
    result TEXT NOT NULL CHECK (result IN ('approved', 'rejected')),
    notes TEXT NOT NULL DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (auto_id) REFERENCES autos(id) ON DELETE CASCADE,
    FOREIGN KEY (institution_id) REFERENCES institutions(id)
);

CREATE INDEX IF NOT EXISTS idx_auto_inspections_auto_year
    ON auto_inspections(auto_id, inspection_year DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_auto_inspections_institution
    ON auto_inspections(institution_id);

CREATE TABLE IF NOT EXISTS auto_attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auto_id INTEGER NOT NULL,
    owner_type TEXT NOT NULL CHECK (owner_type IN ('maintenance', 'inspection')),
    owner_id INTEGER NOT NULL,
    file_name TEXT NOT NULL,
    size_bytes INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (auto_id) REFERENCES autos(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_auto_attachments_owner
    ON auto_attachments(owner_type, owner_id, id);
CREATE INDEX IF NOT EXISTS idx_auto_attachments_auto
    ON auto_attachments(auto_id);

INSERT INTO institution_categories (key, name, description, icon_key)
VALUES
    ('workshop', 'Taller Mecánico',
     'Talleres de mantenimiento y reparación vehicular', 'wrench'),
    ('mechanical_inspection', 'Inspección Mecánica',
     'Centros de inspección técnica vehicular', 'vehicle')
ON CONFLICT(key) DO NOTHING;
