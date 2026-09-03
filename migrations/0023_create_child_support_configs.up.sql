-- Tabla de configuraciones de pensión alimenticia por hijo y categoría (SPEC-051).
-- child_id: hijo al que aplica la config.
-- pension_category_id: categoría de gasto.
-- amount: monto a generar automáticamente.
-- currency: código de moneda (3 letras).
-- is_active: 1 = la config está vigente.
-- auto_generate: 1 = se generan registros automáticamente cada mes.

CREATE TABLE IF NOT EXISTS child_support_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    child_id INTEGER NOT NULL,
    pension_category_id INTEGER NOT NULL,
    amount REAL NOT NULL,
    currency TEXT NOT NULL DEFAULT 'NIO',
    is_active INTEGER NOT NULL DEFAULT 1,
    auto_generate INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (child_id) REFERENCES children(id) ON DELETE CASCADE,
    FOREIGN KEY (pension_category_id) REFERENCES pension_categories(id) ON DELETE CASCADE,
    UNIQUE (child_id, pension_category_id)
);

CREATE INDEX IF NOT EXISTS idx_child_support_configs_child ON child_support_configs(child_id);
CREATE INDEX IF NOT EXISTS idx_child_support_configs_category ON child_support_configs(pension_category_id);