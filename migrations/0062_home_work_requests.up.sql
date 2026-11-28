-- SPEC-164: reportes de problemas y mantenimiento en viviendas.
-- Entidad separada del histórico home_maintenances (ADR-001): un reporte es una
-- solicitud pendiente sin costos, con evidencia multimedia propia.

CREATE TABLE IF NOT EXISTS home_work_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    home_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT 'repair'
        CHECK (category IN ('repair', 'improvement', 'maintenance', 'other')),
    priority TEXT NOT NULL DEFAULT 'normal'
        CHECK (priority IN ('low', 'normal', 'high')),
    status TEXT NOT NULL DEFAULT 'reported'
        CHECK (status IN ('reported', 'converted', 'dismissed')),
    reported_at TEXT NOT NULL,
    maintenance_id INTEGER,
    converted_at TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (home_id) REFERENCES homes(id) ON DELETE CASCADE,
    FOREIGN KEY (maintenance_id) REFERENCES home_maintenances(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_home_work_requests_home_date
    ON home_work_requests(home_id, reported_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_home_work_requests_status
    ON home_work_requests(home_id, status);

CREATE TABLE IF NOT EXISTS home_work_request_attachments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    work_request_id INTEGER NOT NULL,
    media_type TEXT NOT NULL CHECK (media_type IN ('image', 'video')),
    file_name TEXT NOT NULL,
    mime_type TEXT NOT NULL DEFAULT '',
    size_bytes INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (work_request_id) REFERENCES home_work_requests(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_home_work_request_attachments_owner
    ON home_work_request_attachments(work_request_id, id);
