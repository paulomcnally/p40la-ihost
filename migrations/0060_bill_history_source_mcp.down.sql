-- Revierte el CHECK de origen de bill_history al conjunto anterior (SPEC-145).
-- Los eventos con source='mcp' se conservan como 'dashboard' para no perder
-- registros de auditoría.

ALTER TABLE bill_history RENAME TO bill_history_old;

CREATE TABLE bill_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bill_id INTEGER NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('created', 'updated', 'paid')),
    source TEXT NOT NULL CHECK (source IN ('dashboard', 'webhook')),
    changes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bill_id) REFERENCES bills(id)
);

INSERT INTO bill_history (id, bill_id, action, source, changes, created_at)
    SELECT id, bill_id, action,
           CASE WHEN source = 'mcp' THEN 'dashboard' ELSE source END,
           changes, created_at
    FROM bill_history_old;

DROP TABLE bill_history_old;

CREATE INDEX IF NOT EXISTS idx_bill_history_bill_id ON bill_history(bill_id);
