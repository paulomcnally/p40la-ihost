CREATE TABLE IF NOT EXISTS institutions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS institution_analyzers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    institution_id INTEGER NOT NULL,
    analyzer_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (institution_id) REFERENCES institutions(id),
    UNIQUE(institution_id, analyzer_id)
);

ALTER TABLE services ADD COLUMN institution_id INTEGER REFERENCES institutions(id);
ALTER TABLE services ADD COLUMN institution_analyzer_id INTEGER REFERENCES institution_analyzers(id);

CREATE INDEX IF NOT EXISTS idx_institution_analyzers_institution_id ON institution_analyzers(institution_id);
