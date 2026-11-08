DROP TABLE IF EXISTS auto_maintenances_old;

CREATE TABLE auto_maintenances_old (
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

INSERT INTO auto_maintenances_old (
    id, auto_id, performed_at, mileage, institution_id, workshop_name, description,
    labor_cost, supplies_cost, currency_id, created_at, updated_at
)
SELECT
    m.id, m.auto_id, m.performed_at, m.mileage, m.institution_id, COALESCE(i.name, ''), m.description,
    m.labor_cost, m.supplies_cost, m.currency_id, m.created_at, m.updated_at
FROM auto_maintenances m
LEFT JOIN institutions i ON i.id = m.institution_id;

DROP TABLE auto_maintenances;
ALTER TABLE auto_maintenances_old RENAME TO auto_maintenances;

CREATE INDEX IF NOT EXISTS idx_auto_maintenances_auto_date
    ON auto_maintenances(auto_id, performed_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_auto_maintenances_institution
    ON auto_maintenances(institution_id);
