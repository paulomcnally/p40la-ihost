-- Tabla de notificaciones (destinatarios de email) del módulo Pensión Alimenticia (SPEC-046).
-- name: nombre del destinatario (requerido).
-- email: email del destinatario (requerido, único).
-- active: 1 = recibe emails, 0 = deshabilitado (default 1).

CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_active ON notifications(active);