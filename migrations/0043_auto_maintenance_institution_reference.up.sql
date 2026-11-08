DROP TABLE IF EXISTS auto_maintenances_new;

-- Migrar talleres libres existentes a instituciones para no perder el nombre.
INSERT INTO institutions (name, category_id)
SELECT DISTINCT TRIM(m.workshop_name), (SELECT id FROM institution_categories WHERE key = 'workshop')
FROM auto_maintenances m
WHERE m.institution_id IS NULL
  AND TRIM(m.workshop_name) <> ''
  AND NOT EXISTS (SELECT 1 FROM institutions i WHERE i.name = TRIM(m.workshop_name));

UPDATE auto_maintenances
SET institution_id = (
    SELECT i.id FROM institutions i WHERE i.name = TRIM(auto_maintenances.workshop_name)
)
WHERE institution_id IS NULL
  AND TRIM(workshop_name) <> '';

-- Fallback para registros legacy sin institución ni nombre de taller.
INSERT INTO institutions (name, category_id)
SELECT 'Taller sin especificar', (SELECT id FROM institution_categories WHERE key = 'workshop')
WHERE EXISTS (SELECT 1 FROM auto_maintenances WHERE institution_id IS NULL)
  AND NOT EXISTS (SELECT 1 FROM institutions WHERE name = 'Taller sin especificar');

UPDATE auto_maintenances
SET institution_id = (SELECT id FROM institutions WHERE name = 'Taller sin especificar')
WHERE institution_id IS NULL;

-- Reconstruir la tabla sin workshop_name y con institution_id obligatorio.
CREATE TABLE auto_maintenances_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auto_id INTEGER NOT NULL,
    performed_at TEXT NOT NULL,
    mileage INTEGER NOT NULL CHECK (mileage >= 0),
    institution_id INTEGER NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    labor_cost REAL NOT NULL DEFAULT 0 CHECK (labor_cost >= 0),
    supplies_cost REAL NOT NULL DEFAULT 0 CHECK (supplies_cost >= 0),
    currency_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (auto_id) REFERENCES autos(id) ON DELETE CASCADE,
    FOREIGN KEY (institution_id) REFERENCES institutions(id),
    FOREIGN KEY (currency_id) REFERENCES currencies(id)
);

INSERT INTO auto_maintenances_new (
    id, auto_id, performed_at, mileage, institution_id, description,
    labor_cost, supplies_cost, currency_id, created_at, updated_at
)
SELECT
    id, auto_id, performed_at, mileage, institution_id, description,
    labor_cost, supplies_cost, currency_id, created_at, updated_at
FROM auto_maintenances;

DROP TABLE auto_maintenances;
ALTER TABLE auto_maintenances_new RENAME TO auto_maintenances;

CREATE INDEX IF NOT EXISTS idx_auto_maintenances_auto_date
    ON auto_maintenances(auto_id, performed_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_auto_maintenances_institution
    ON auto_maintenances(institution_id);
