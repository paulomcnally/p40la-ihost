-- SPEC-143: índices para el resumen de ingresos y gastos.
CREATE INDEX IF NOT EXISTS idx_rental_payments_paid_at ON rental_payments(paid_at);
CREATE INDEX IF NOT EXISTS idx_rental_charges_due_date ON rental_charges(due_date);
CREATE INDEX IF NOT EXISTS idx_home_maintenances_performed_at ON home_maintenances(performed_at);
