-- SPEC-143: reverso de 0058. Recrea listing_enabled desde usage.
ALTER TABLE homes ADD COLUMN listing_enabled INTEGER NOT NULL DEFAULT 1;

UPDATE homes SET listing_enabled = CASE WHEN usage = 'rental' THEN 1 ELSE 0 END;

ALTER TABLE homes DROP COLUMN usage;
