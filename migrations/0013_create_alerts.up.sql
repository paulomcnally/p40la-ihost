CREATE TABLE IF NOT EXISTS alerts (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    key           TEXT    NOT NULL UNIQUE,
    title         TEXT    NOT NULL,
    description   TEXT    NOT NULL,
    mail_enabled  INTEGER NOT NULL DEFAULT 0,
    voice_enabled INTEGER NOT NULL DEFAULT 0,
    speech        TEXT    NOT NULL,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);
