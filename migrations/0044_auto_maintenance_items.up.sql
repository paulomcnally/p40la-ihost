DROP TABLE IF EXISTS temp.auto_maintenance_costs;
CREATE TEMP TABLE auto_maintenance_costs AS
SELECT id, labor_cost, supplies_cost FROM auto_maintenances;

DROP TABLE IF EXISTS auto_maintenances_new;

CREATE TABLE auto_maintenances_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auto_id INTEGER NOT NULL,
    performed_at TEXT NOT NULL,
    mileage INTEGER NOT NULL CHECK (mileage >= 0),
    institution_id INTEGER NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    currency_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (auto_id) REFERENCES autos(id) ON DELETE CASCADE,
    FOREIGN KEY (institution_id) REFERENCES institutions(id),
    FOREIGN KEY (currency_id) REFERENCES currencies(id)
);

INSERT INTO auto_maintenances_new (
    id, auto_id, performed_at, mileage, institution_id, description,
    currency_id, created_at, updated_at
)
SELECT
    id, auto_id, performed_at, mileage, institution_id, description,
    currency_id, created_at, updated_at
FROM auto_maintenances;

DROP TABLE auto_maintenances;
ALTER TABLE auto_maintenances_new RENAME TO auto_maintenances;

CREATE INDEX IF NOT EXISTS idx_auto_maintenances_auto_date
    ON auto_maintenances(auto_id, performed_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_auto_maintenances_institution
    ON auto_maintenances(institution_id);

CREATE TABLE IF NOT EXISTS auto_maintenance_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    maintenance_id INTEGER NOT NULL,
    item_type TEXT NOT NULL CHECK (item_type IN ('labor', 'supplies')),
    description TEXT NOT NULL,
    amount REAL NOT NULL DEFAULT 0 CHECK (amount >= 0),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (maintenance_id) REFERENCES auto_maintenances(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_auto_maintenance_items_maintenance
    ON auto_maintenance_items(maintenance_id, item_type, id);

INSERT INTO auto_maintenance_items (maintenance_id, item_type, description, amount)
SELECT id, 'labor', 'Mano de obra', labor_cost
FROM temp.auto_maintenance_costs
WHERE labor_cost > 0;

INSERT INTO auto_maintenance_items (maintenance_id, item_type, description, amount)
SELECT id, 'supplies', 'Insumos', supplies_cost
FROM temp.auto_maintenance_costs
WHERE supplies_cost > 0;

DROP TABLE temp.auto_maintenance_costs;
