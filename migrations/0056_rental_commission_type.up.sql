-- SPEC-140: comisión de intermediación por porcentaje o monto fijo.
-- El modo fijo cubre casos como "un mes de alquiler" o un depósito pactado.

ALTER TABLE rental_leases
    ADD COLUMN commission_type TEXT NOT NULL DEFAULT 'percent'
    CHECK (commission_type IN ('percent', 'fixed'));

ALTER TABLE rental_leases
    ADD COLUMN commission_amount REAL NOT NULL DEFAULT 0
    CHECK (commission_amount >= 0);
