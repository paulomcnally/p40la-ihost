-- SPEC-143: uso de la vivienda. Reemplaza listing_enabled (ADR-001).
ALTER TABLE homes ADD COLUMN usage TEXT NOT NULL DEFAULT 'rental'
    CHECK (usage IN ('rental', 'inhabited', 'unused'));

UPDATE homes SET usage = 'unused' WHERE listing_enabled = 0;

ALTER TABLE homes DROP COLUMN listing_enabled;
