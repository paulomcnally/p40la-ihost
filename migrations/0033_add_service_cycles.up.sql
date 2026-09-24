-- Ciclos de servicio (SPEC-091): un servicio de seguro se renueva por períodos.
-- Cada renovación crea un ciclo nuevo (sequence creciente); las facturas quedan
-- asociadas a su ciclo vía bills.cycle_id, dejando rastro de a qué período
-- pertenecen (no más "facturas infinitas" sin contexto).

CREATE TABLE IF NOT EXISTS service_cycles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER NOT NULL,
    sequence INTEGER NOT NULL,
    start_date TEXT,
    end_date TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (service_id) REFERENCES services(id),
    UNIQUE(service_id, sequence)
);
CREATE INDEX IF NOT EXISTS idx_service_cycles_service_id ON service_cycles(service_id);

ALTER TABLE bills ADD COLUMN cycle_id INTEGER;
CREATE INDEX IF NOT EXISTS idx_bills_cycle_id ON bills(cycle_id);

-- Backfill: ciclo 1 para cada servicio con vigencia (start/end definidos).
-- Los servicios sin vigencia quedan sin ciclo (bills.cycle_id NULL).
INSERT INTO service_cycles (service_id, sequence, start_date, end_date)
SELECT id, 1, start_date, end_date
FROM services
WHERE deleted_at IS NULL AND start_date IS NOT NULL AND end_date IS NOT NULL;

-- Asociar las facturas existentes de esos servicios a su ciclo 1.
UPDATE bills
SET cycle_id = (
    SELECT sc.id FROM service_cycles sc
    WHERE sc.service_id = bills.service_id
    ORDER BY sc.sequence DESC LIMIT 1
)
WHERE cycle_id IS NULL
  AND EXISTS (SELECT 1 FROM service_cycles sc WHERE sc.service_id = bills.service_id);