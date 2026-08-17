-- Agregar hash MD5 del archivo importado para dedup (SPEC-041).
-- Un índice único parcial por servicio: el mismo PDF no puede importarse dos veces.
-- file_hash es NULL para facturas creadas manualmente (sin PDF).

ALTER TABLE bills ADD COLUMN file_hash TEXT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_bills_service_file_hash
    ON bills(service_id, file_hash)
    WHERE file_hash IS NOT NULL AND file_hash != '';