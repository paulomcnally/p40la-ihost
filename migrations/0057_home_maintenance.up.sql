-- SPEC-141: reparaciones y mantenimientos del hogar (histórico de insumos y mano de obra).
-- Importes en REAL (convención del proyecto), fechas de negocio en TEXT ISO.

INSERT INTO institution_categories (key, name, description, icon_key)
VALUES ('hardware', 'Ferretería', 'Ferreterías y comercios de materiales para el hogar', 'wrench')
ON CONFLICT(key) DO NOTHING;

CREATE TABLE IF NOT EXISTS home_maintenances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    home_id INTEGER NOT NULL,
    performed_at TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'corrective'
        CHECK (kind IN ('corrective', 'preventive')),
    area TEXT NOT NULL DEFAULT 'other'
        CHECK (area IN ('plumbing', 'electrical', 'roof', 'paint', 'carpentry',
                        'appliances', 'garden', 'cleaning', 'other')),
    institution_id INTEGER,
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (home_id) REFERENCES homes(id) ON DELETE CASCADE,
    FOREIGN KEY (institution_id) REFERENCES institutions(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_home_maintenances_home_date
    ON home_maintenances(home_id, performed_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_home_maintenances_institution
    ON home_maintenances(institution_id);

CREATE TABLE IF NOT EXISTS home_maintenance_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    maintenance_id INTEGER NOT NULL,
    item_type TEXT NOT NULL CHECK (item_type IN ('labor', 'supplies')),
    description TEXT NOT NULL,
    quantity REAL NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit TEXT NOT NULL DEFAULT 'unit'
        CHECK (unit IN ('unit', 'm', 'm2', 'kg', 'lb', 'l', 'gal', 'sack', 'box')),
    amount REAL NOT NULL DEFAULT 0 CHECK (amount >= 0),
    currency_id INTEGER NOT NULL,
    institution_id INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (maintenance_id) REFERENCES home_maintenances(id) ON DELETE CASCADE,
    FOREIGN KEY (currency_id) REFERENCES currencies(id),
    FOREIGN KEY (institution_id) REFERENCES institutions(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_home_maintenance_items_maintenance
    ON home_maintenance_items(maintenance_id, item_type, id);
CREATE INDEX IF NOT EXISTS idx_home_maintenance_items_currency
    ON home_maintenance_items(currency_id);
