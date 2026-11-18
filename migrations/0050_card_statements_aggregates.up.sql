-- SPEC-139: agregados del estado de cuenta que imprimen los bancos.
-- Aditiva: los estados existentes conservan sus valores (0 en los nuevos).

ALTER TABLE card_statements ADD COLUMN interest_bonifiable REAL NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN interest_bonifiable_usd REAL NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN mora_amount REAL NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN mora_amount_usd REAL NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN arrears_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN over_limit_amount REAL NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN over_limit_amount_usd REAL NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN points_previous INTEGER NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN points_earned INTEGER NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN points_used INTEGER NOT NULL DEFAULT 0;
ALTER TABLE card_statements ADD COLUMN points_available INTEGER NOT NULL DEFAULT 0;
