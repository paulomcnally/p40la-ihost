-- SPEC-119: importador mensual de gastos escolares y pensión.
-- Agrega reparto y moneda a las categorías, el diccionario de palabras clave
-- y las tablas de auditoría de importaciones.

ALTER TABLE pension_categories ADD COLUMN share_percent INTEGER NOT NULL DEFAULT 50;
ALTER TABLE pension_categories ADD COLUMN currency TEXT NOT NULL DEFAULT 'NIO';

-- SQLite no permite CHECK en ALTER TABLE ADD COLUMN: se valida con triggers.
CREATE TRIGGER trg_pension_categories_share_insert
BEFORE INSERT ON pension_categories
FOR EACH ROW WHEN NEW.share_percent NOT IN (50, 100)
BEGIN
    SELECT RAISE(ABORT, 'share_percent debe ser 50 o 100');
END;

CREATE TRIGGER trg_pension_categories_share_update
BEFORE UPDATE OF share_percent ON pension_categories
FOR EACH ROW WHEN NEW.share_percent NOT IN (50, 100)
BEGIN
    SELECT RAISE(ABORT, 'share_percent debe ser 50 o 100');
END;

CREATE TABLE IF NOT EXISTS pension_category_keywords (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pension_category_id INTEGER NOT NULL REFERENCES pension_categories(id) ON DELETE CASCADE,
    keyword TEXT NOT NULL,
    normalized_keyword TEXT NOT NULL UNIQUE,
    source TEXT NOT NULL DEFAULT 'learned' CHECK (source IN ('seed', 'learned', 'manual')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pension_category_keywords_category
    ON pension_category_keywords(pension_category_id);

CREATE TABLE IF NOT EXISTS pension_import_batches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL DEFAULT '',
    file_hash TEXT NOT NULL DEFAULT '',
    period_year INTEGER NOT NULL,
    period_month INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'preview' CHECK (status IN ('preview', 'confirmed', 'discarded')),
    total_lines INTEGER NOT NULL DEFAULT 0,
    declared_total REAL NOT NULL DEFAULT 0,
    computed_total REAL NOT NULL DEFAULT 0,
    created_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    confirmed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_pension_import_batches_period
    ON pension_import_batches(period_year, period_month);

CREATE TABLE IF NOT EXISTS pension_import_rows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    batch_id INTEGER NOT NULL REFERENCES pension_import_batches(id) ON DELETE CASCADE,
    line_no INTEGER NOT NULL,
    raw_text TEXT NOT NULL DEFAULT '',
    concept TEXT NOT NULL DEFAULT '',
    amount_total REAL NOT NULL DEFAULT 0,
    amount_to_record REAL NOT NULL DEFAULT 0,
    rounding_adjustment REAL NOT NULL DEFAULT 0,
    share_percent INTEGER NOT NULL DEFAULT 50,
    currency TEXT NOT NULL DEFAULT 'NIO',
    suggested_category_id INTEGER REFERENCES pension_categories(id),
    chosen_category_id INTEGER REFERENCES pension_categories(id),
    child_id INTEGER REFERENCES children(id),
    confidence REAL NOT NULL DEFAULT 0,
    match_reason TEXT NOT NULL DEFAULT '',
    validation_status TEXT NOT NULL DEFAULT 'ok' CHECK (validation_status IN ('ok', 'invalid')),
    validation_code TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL DEFAULT 'create' CHECK (action IN ('create', 'skip', 'existing', 'error')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pension_import_rows_batch
    ON pension_import_rows(batch_id);

-- Semilla de reparto por nombre de categoría (idempotente).
UPDATE pension_categories SET share_percent = 100
WHERE name IN ('Pensión Alimenticia', 'Colegio - Transporte');

UPDATE pension_categories SET share_percent = 50
WHERE name NOT IN ('Pensión Alimenticia', 'Colegio - Transporte');

-- Semilla de palabras clave (solo si la categoría existe).
INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
SELECT id, 'PENSION', 'PENSION', 'seed' FROM pension_categories
WHERE name = 'Pensión Alimenticia'
  AND NOT EXISTS (SELECT 1 FROM pension_category_keywords WHERE normalized_keyword = 'PENSION');

INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
SELECT id, 'COLEGIATURA', 'COLEGIATURA', 'seed' FROM pension_categories
WHERE name = 'Colegio - Mensualidad'
  AND NOT EXISTS (SELECT 1 FROM pension_category_keywords WHERE normalized_keyword = 'COLEGIATURA');

INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
SELECT id, 'MENSUALIDAD', 'MENSUALIDAD', 'seed' FROM pension_categories
WHERE name = 'Colegio - Mensualidad'
  AND NOT EXISTS (SELECT 1 FROM pension_category_keywords WHERE normalized_keyword = 'MENSUALIDAD');

INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
SELECT id, 'COMBUSTIBLE', 'COMBUSTIBLE', 'seed' FROM pension_categories
WHERE name = 'Colegio - Transporte'
  AND NOT EXISTS (SELECT 1 FROM pension_category_keywords WHERE normalized_keyword = 'COMBUSTIBLE');

INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
SELECT id, 'TRANSPORTE', 'TRANSPORTE', 'seed' FROM pension_categories
WHERE name = 'Colegio - Transporte'
  AND NOT EXISTS (SELECT 1 FROM pension_category_keywords WHERE normalized_keyword = 'TRANSPORTE');

INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
SELECT id, 'UNIFORME', 'UNIFORME', 'seed' FROM pension_categories
WHERE name = 'Colegio - Uniforme'
  AND NOT EXISTS (SELECT 1 FROM pension_category_keywords WHERE normalized_keyword = 'UNIFORME');

INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
SELECT id, 'SEGURO', 'SEGURO', 'seed' FROM pension_categories
WHERE name = 'Colegio - Seguro'
  AND NOT EXISTS (SELECT 1 FROM pension_category_keywords WHERE normalized_keyword = 'SEGURO');

INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
SELECT id, 'MATRICULA', 'MATRICULA', 'seed' FROM pension_categories
WHERE name = 'Colegio - Matrícula'
  AND NOT EXISTS (SELECT 1 FROM pension_category_keywords WHERE normalized_keyword = 'MATRICULA');

INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
SELECT id, 'LIBROS', 'LIBROS', 'seed' FROM pension_categories
WHERE name = 'Colegio - Libros'
  AND NOT EXISTS (SELECT 1 FROM pension_category_keywords WHERE normalized_keyword = 'LIBROS');

INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
SELECT id, 'HOPPERS', 'HOPPERS', 'seed' FROM pension_categories
WHERE name = 'Colegio - Actividad - Hoppers'
  AND NOT EXISTS (SELECT 1 FROM pension_category_keywords WHERE normalized_keyword = 'HOPPERS');

INSERT INTO pension_category_keywords (pension_category_id, keyword, normalized_keyword, source)
SELECT id, 'ENCUENTRO CON CRISTO', 'ENCUENTRO CON CRISTO', 'seed' FROM pension_categories
WHERE name = 'Colegio - Actividad - Encuentro con Cristo'
  AND NOT EXISTS (SELECT 1 FROM pension_category_keywords WHERE normalized_keyword = 'ENCUENTRO CON CRISTO');
