-- Tabla de registros mensuales de manutención (SPEC-049).
-- child_id: hijo al que corresponde el registro.
-- pension_category_id: categoría de gasto (ej: Colegio, Alimentación).
-- status: pending | paid | rejected.
-- Campos de pago: paid_at, payment_method, payment_reference, evidence_notes.
-- notes: motivo de rechazo (o nota general).
-- proof_file_name: nombre original del comprobante subido (el archivo vive en disco).
-- original_amount/original_currency/exchange_rate: conversión de moneda al pagar.

CREATE TABLE IF NOT EXISTS support_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    child_id INTEGER NOT NULL,
    pension_category_id INTEGER NOT NULL,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL,
    amount REAL NOT NULL,
    currency TEXT NOT NULL DEFAULT 'NIO',
    status TEXT NOT NULL DEFAULT 'pending',
    paid_at DATETIME NULL,
    payment_method TEXT NULL,
    payment_reference TEXT NULL,
    evidence_notes TEXT NULL,
    notes TEXT NULL,
    proof_file_name TEXT NULL,
    original_amount REAL NULL,
    original_currency TEXT NULL,
    exchange_rate REAL NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (child_id) REFERENCES children(id) ON DELETE CASCADE,
    FOREIGN KEY (pension_category_id) REFERENCES pension_categories(id) ON DELETE CASCADE,
    UNIQUE (child_id, pension_category_id, year, month)
);

CREATE INDEX IF NOT EXISTS idx_support_records_year_month ON support_records(year, month);
CREATE INDEX IF NOT EXISTS idx_support_records_child_id ON support_records(child_id);
CREATE INDEX IF NOT EXISTS idx_support_records_category_id ON support_records(pension_category_id);