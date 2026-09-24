-- Índice para el reverse lookup de pólizas (SPEC-091).
-- auto_services tiene PK (auto_id, service_id); la vista inversa
-- (autos asociados a un servicio) filtra por service_id.
CREATE INDEX IF NOT EXISTS idx_auto_services_service_id ON auto_services(service_id);