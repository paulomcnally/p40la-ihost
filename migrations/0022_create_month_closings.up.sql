-- Tabla de cierres de mes (SPEC-049).
-- Una fila por (year, month) cerrado: la presencia indica que el mes está cerrado.
-- Cuando un mes está cerrado, los registros y pagos de ese mes no pueden modificarse.

CREATE TABLE IF NOT EXISTS month_closings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL,
    closed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (year, month)
);

CREATE INDEX IF NOT EXISTS idx_month_closings_year_month ON month_closings(year, month);