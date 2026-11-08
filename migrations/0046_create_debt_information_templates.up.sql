-- SPEC-131: plantillas de información y forma de pago por deuda.
-- No modifica el esquema de debts ni de debt_bills.

CREATE TABLE IF NOT EXISTS debt_information_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    fields_json TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS debt_payment_information (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    debt_id INTEGER NOT NULL UNIQUE,
    template_id INTEGER NOT NULL,
    values_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (debt_id) REFERENCES debts(id) ON DELETE CASCADE,
    FOREIGN KEY (template_id) REFERENCES debt_information_templates(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_debt_payment_information_template
    ON debt_payment_information(template_id);
