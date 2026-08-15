-- Relajar la restricción CHECK de month para permitir 0 (facturas anuales)
-- SQLite no soporta ALTER CONSTRAINT, se recrea la tabla.

ALTER TABLE bills RENAME TO bills_old;

CREATE TABLE bills (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER NOT NULL,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL CHECK (month BETWEEN 0 AND 12),
    amount REAL NOT NULL,
    invoice_number TEXT,
    status TEXT NOT NULL CHECK (status IN ('pending', 'paid')) DEFAULT 'pending',
    drive_url TEXT,
    deleted_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (service_id) REFERENCES services(id),
    UNIQUE(service_id, year, month)
);

INSERT INTO bills (id, service_id, year, month, amount, invoice_number, status, drive_url, deleted_at, created_at, updated_at)
    SELECT id, service_id, year, month, amount, invoice_number, status, drive_url, deleted_at, created_at, updated_at
    FROM bills_old;

DROP TABLE bills_old;

CREATE INDEX IF NOT EXISTS idx_bills_service_id ON bills(service_id);
CREATE INDEX IF NOT EXISTS idx_bills_deleted_at ON bills(deleted_at);
