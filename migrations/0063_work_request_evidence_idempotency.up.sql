-- SPEC-165 (ampliación): idempotencia de alta y evidencia por MCP.
-- Aditiva: agrega deduplicación por client_request_id/sha256 y la tabla efímera
-- de tokens de carga. No borra ni reemplaza datos existentes.

ALTER TABLE home_work_requests ADD COLUMN client_request_id TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_home_work_requests_client_request
    ON home_work_requests(home_id, client_request_id)
    WHERE client_request_id <> '';

ALTER TABLE home_work_request_attachments ADD COLUMN sha256 TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_home_work_request_attachments_sha
    ON home_work_request_attachments(work_request_id, sha256)
    WHERE sha256 <> '';

CREATE TABLE IF NOT EXISTS work_request_upload_tokens (
    jti TEXT PRIMARY KEY,
    work_request_id INTEGER NOT NULL,
    max_files INTEGER NOT NULL,
    files_used INTEGER NOT NULL DEFAULT 0,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (work_request_id) REFERENCES home_work_requests(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_work_request_upload_tokens_expiry
    ON work_request_upload_tokens(expires_at);
