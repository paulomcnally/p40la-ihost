-- Tablas de deudas y sus cuotas (SPEC-054).
-- debts: deuda con acreedor (institución), identificador, descripción, totales,
--        moneda, número de cuotas, monto por cuota, tasa de interés, día de
--        pago del mes y fecha de inicio.
-- debt_bills: cuotas generadas automáticamente con estados pending/paid
--             (misma semántica que la tabla bills).

CREATE TABLE IF NOT EXISTS debts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    institution_id INTEGER NOT NULL,
    identifier TEXT NOT NULL,
    description TEXT NOT NULL,
    total REAL NOT NULL DEFAULT 0,
    principal REAL NOT NULL DEFAULT 0,
    currency_id INTEGER NOT NULL,
    installments_total INTEGER NOT NULL CHECK (installments_total >= 1),
    installment_amount REAL NOT NULL DEFAULT 0,
    interest_rate REAL NOT NULL DEFAULT 0,
    payment_day INTEGER NOT NULL CHECK (payment_day BETWEEN 1 AND 31),
    start_date TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'activa' CHECK (status IN ('activa', 'inactiva', 'finalizada')),
    deleted_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (institution_id) REFERENCES institutions(id),
    FOREIGN KEY (currency_id) REFERENCES currencies(id)
);

CREATE TABLE IF NOT EXISTS debt_bills (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    debt_id INTEGER NOT NULL,
    installment_number INTEGER NOT NULL CHECK (installment_number >= 1),
    due_date TEXT NOT NULL,
    amount REAL NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid')),
    paid_at DATETIME,
    payment_reference TEXT,
    deleted_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (debt_id) REFERENCES debts(id),
    UNIQUE(debt_id, installment_number)
);

CREATE INDEX IF NOT EXISTS idx_debt_bills_debt_id ON debt_bills(debt_id);
CREATE INDEX IF NOT EXISTS idx_debt_bills_due_date ON debt_bills(due_date);