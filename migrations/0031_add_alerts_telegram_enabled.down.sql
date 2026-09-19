-- Rebuild de tabla (compatible con cualquier versión de SQLite) para
-- quitar telegram_enabled. SPEC-088.
ALTER TABLE alerts RENAME TO alerts_old;

CREATE TABLE alerts (
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

INSERT INTO alerts (id, key, title, description, mail_enabled, voice_enabled, speech, created_at, updated_at)
SELECT id, key, title, description, mail_enabled, voice_enabled, speech, created_at, updated_at
FROM alerts_old;

DROP TABLE alerts_old;