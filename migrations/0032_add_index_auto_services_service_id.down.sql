-- Quitar el índice del reverse lookup de pólizas (SPEC-091).
DROP INDEX IF EXISTS idx_auto_services_service_id;