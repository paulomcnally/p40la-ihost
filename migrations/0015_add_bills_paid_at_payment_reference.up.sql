-- Agregar fecha de pago y referencia de pago a las facturas (SPEC-043).
-- paid_at: fecha en que se realizó el pago (YYYY-MM-DD, nullable).
-- payment_reference: referencia interna del pago (ej. número de transacción, nullable).

ALTER TABLE bills ADD COLUMN paid_at DATETIME;
ALTER TABLE bills ADD COLUMN payment_reference TEXT;