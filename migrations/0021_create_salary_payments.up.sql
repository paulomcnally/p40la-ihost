-- Tabla de pagos mensuales de salario (SPEC-049).
-- salary_id: salario/ingreso del que proviene el pago.
-- status: pending | received.
-- received_amount: monto realmente recibido (puede diferir del esperado).
-- UNIQUE (salary_id, year, month): un solo pago por salario y mes.

CREATE TABLE IF NOT EXISTS salary_payments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    salary_id INTEGER NOT NULL,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL,
    amount REAL NOT NULL,
    currency TEXT NOT NULL DEFAULT 'NIO',
    status TEXT NOT NULL DEFAULT 'pending',
    received_amount REAL NULL,
    received_at DATETIME NULL,
    notes TEXT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (salary_id) REFERENCES salaries(id) ON DELETE CASCADE,
    UNIQUE (salary_id, year, month)
);

CREATE INDEX IF NOT EXISTS idx_salary_payments_year_month ON salary_payments(year, month);
CREATE INDEX IF NOT EXISTS idx_salary_payments_salary_id ON salary_payments(salary_id);