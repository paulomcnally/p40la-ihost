-- Agregar fechas de emisión y vencimiento a facturas (SPEC-081).
-- issue_date: fecha en que el proveedor generó la factura (YYYY-MM-DD, nullable).
--             No es created_at: la provee el sistema externo vía webhook.
-- due_date:   fecha de vencimiento (YYYY-MM-DD, nullable). Permite saber cuántos
--             días quedan antes de que suspendan el servicio.
-- Ambas opcionales: los sistemas externos pueden no proveerlas (NULL por defecto).

ALTER TABLE bills ADD COLUMN issue_date TEXT;
ALTER TABLE bills ADD COLUMN due_date TEXT;

-- debt_bills: solo issue_date; el vencimiento ya existe como due_date (SPEC-054).
-- Autorización explícita del usuario (ADR-002 de SPEC-081).
ALTER TABLE debt_bills ADD COLUMN issue_date TEXT;