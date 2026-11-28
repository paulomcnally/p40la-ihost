-- Reverso de SPEC-165 (ampliación). Los tokens son efímeros y no hay pérdida
-- de reportes ni de adjuntos existentes.

DROP TABLE IF EXISTS work_request_upload_tokens;

DROP INDEX IF EXISTS idx_home_work_request_attachments_sha;
ALTER TABLE home_work_request_attachments DROP COLUMN sha256;

DROP INDEX IF EXISTS idx_home_work_requests_client_request;
ALTER TABLE home_work_requests DROP COLUMN client_request_id;
