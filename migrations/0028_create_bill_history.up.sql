CREATE TABLE IF NOT EXISTS bill_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bill_id INTEGER NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('created', 'updated', 'paid')),
    source TEXT NOT NULL CHECK (source IN ('dashboard', 'webhook')),
    changes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bill_id) REFERENCES bills(id)
);

CREATE INDEX IF NOT EXISTS idx_bill_history_bill_id ON bill_history(bill_id);