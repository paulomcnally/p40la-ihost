-- SPEC-140: cargos, numeración, facturas internas, pagos, recibos y adjuntos.

CREATE TABLE IF NOT EXISTS rental_charges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lease_id INTEGER NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('rent', 'maintenance', 'deposit', 'late_fee', 'utility', 'other')),
    period TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    amount REAL NOT NULL DEFAULT 0 CHECK (amount >= 0),
    currency TEXT NOT NULL DEFAULT 'USD',
    due_date TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'invoiced', 'partially_paid', 'paid', 'void')),
    invoice_id INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (lease_id) REFERENCES rental_leases(id) ON DELETE CASCADE,
    UNIQUE (lease_id, period, kind)
);

CREATE INDEX IF NOT EXISTS idx_rental_charges_lease ON rental_charges(lease_id);
CREATE INDEX IF NOT EXISTS idx_rental_charges_status ON rental_charges(status);

CREATE TABLE IF NOT EXISTS rental_sequences (
    kind TEXT NOT NULL CHECK (kind IN ('invoice', 'receipt')),
    series TEXT NOT NULL,
    year INTEGER NOT NULL,
    next_number INTEGER NOT NULL DEFAULT 1 CHECK (next_number > 0),
    PRIMARY KEY (kind, series, year)
);

CREATE TABLE IF NOT EXISTS rental_invoices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lease_id INTEGER NOT NULL,
    series TEXT NOT NULL,
    number INTEGER NOT NULL,
    year INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'issued' CHECK (status IN ('issued', 'void')),
    issue_date TEXT NOT NULL,
    due_date TEXT NOT NULL,
    currency TEXT NOT NULL,
    subtotal REAL NOT NULL DEFAULT 0,
    total REAL NOT NULL DEFAULT 0,
    snapshot_json TEXT NOT NULL DEFAULT '{}',
    pdf_path TEXT NOT NULL DEFAULT '',
    pdf_sha256 TEXT NOT NULL DEFAULT '',
    pdf_size_bytes INTEGER NOT NULL DEFAULT 0,
    sent_at DATETIME,
    void_reason TEXT NOT NULL DEFAULT '',
    voided_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (lease_id) REFERENCES rental_leases(id) ON DELETE RESTRICT,
    UNIQUE (series, year, number)
);

CREATE INDEX IF NOT EXISTS idx_rental_invoices_lease ON rental_invoices(lease_id);

CREATE TABLE IF NOT EXISTS rental_invoice_lines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    invoice_id INTEGER NOT NULL,
    charge_id INTEGER,
    description TEXT NOT NULL,
    amount REAL NOT NULL DEFAULT 0,
    FOREIGN KEY (invoice_id) REFERENCES rental_invoices(id) ON DELETE CASCADE,
    FOREIGN KEY (charge_id) REFERENCES rental_charges(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_rental_invoice_lines_invoice ON rental_invoice_lines(invoice_id);

CREATE TABLE IF NOT EXISTS rental_payments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lease_id INTEGER NOT NULL,
    invoice_id INTEGER,
    paid_at TEXT NOT NULL,
    amount REAL NOT NULL CHECK (amount > 0),
    currency TEXT NOT NULL,
    method TEXT NOT NULL DEFAULT 'transfer' CHECK (method IN ('cash', 'transfer', 'check', 'card', 'other')),
    reference TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (lease_id) REFERENCES rental_leases(id) ON DELETE CASCADE,
    FOREIGN KEY (invoice_id) REFERENCES rental_invoices(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_rental_payments_lease ON rental_payments(lease_id);
CREATE INDEX IF NOT EXISTS idx_rental_payments_invoice ON rental_payments(invoice_id);

CREATE TABLE IF NOT EXISTS rental_payment_proofs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    payment_id INTEGER NOT NULL,
    original_name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (payment_id) REFERENCES rental_payments(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rental_payment_proofs_payment ON rental_payment_proofs(payment_id);

CREATE TABLE IF NOT EXISTS rental_receipts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    payment_id INTEGER NOT NULL UNIQUE,
    series TEXT NOT NULL,
    number INTEGER NOT NULL,
    year INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'issued' CHECK (status IN ('issued', 'void')),
    issue_date TEXT NOT NULL,
    amount REAL NOT NULL,
    currency TEXT NOT NULL,
    pdf_path TEXT NOT NULL DEFAULT '',
    pdf_sha256 TEXT NOT NULL DEFAULT '',
    pdf_size_bytes INTEGER NOT NULL DEFAULT 0,
    sent_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (payment_id) REFERENCES rental_payments(id) ON DELETE CASCADE,
    UNIQUE (series, year, number)
);

CREATE INDEX IF NOT EXISTS idx_rental_receipts_payment ON rental_receipts(payment_id);

CREATE TABLE IF NOT EXISTS rental_lease_documents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lease_id INTEGER NOT NULL,
    doc_type TEXT NOT NULL CHECK (doc_type IN ('contract', 'addendum', 'inventory', 'id', 'income_proof', 'insurance', 'agent_credential', 'other')),
    original_name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (lease_id) REFERENCES rental_leases(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rental_lease_documents_lease ON rental_lease_documents(lease_id);
