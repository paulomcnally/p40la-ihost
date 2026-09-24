-- Revertir ciclos de servicio (SPEC-091).
DROP INDEX IF EXISTS idx_bills_cycle_id;
ALTER TABLE bills DROP COLUMN cycle_id;
DROP INDEX IF EXISTS idx_service_cycles_service_id;
DROP TABLE IF EXISTS service_cycles;