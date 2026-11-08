-- SPEC-127: reconocimiento de comprobantes bancarios a nivel proyecto.
-- Guarda la vista previa del reconocimiento y, al confirmar, la asociación
-- con una cuota de deuda. No modifica el esquema de debts ni debt_bills.

CREATE TABLE IF NOT EXISTS document_recognitions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    document_hash TEXT NOT NULL UNIQUE,
    recognizer_id TEXT NOT NULL DEFAULT '',
    document_type TEXT NOT NULL DEFAULT 'unknown',
    status TEXT NOT NULL DEFAULT 'recognized'
        CHECK (status IN ('recognized', 'associated', 'rejected', 'expired')),
    reference TEXT NOT NULL DEFAULT '',
    paid_at DATETIME,
    amount REAL NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT '',
    installment_number INTEGER,
    concept_text TEXT NOT NULL DEFAULT '',
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id INTEGER,
    confidence REAL NOT NULL DEFAULT 0,
    warnings TEXT NOT NULL DEFAULT '[]',
    inbox_path TEXT NOT NULL DEFAULT '',
    final_path TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    confirmed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_document_recognitions_status
    ON document_recognitions(status);

CREATE INDEX IF NOT EXISTS idx_document_recognitions_reference
    ON document_recognitions(reference);

CREATE INDEX IF NOT EXISTS idx_document_recognitions_entity
    ON document_recognitions(entity_type, entity_id);
