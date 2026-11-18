-- SPEC-139: revierte los agregados del estado de cuenta.

ALTER TABLE card_statements DROP COLUMN points_available;
ALTER TABLE card_statements DROP COLUMN points_used;
ALTER TABLE card_statements DROP COLUMN points_earned;
ALTER TABLE card_statements DROP COLUMN points_previous;
ALTER TABLE card_statements DROP COLUMN over_limit_amount_usd;
ALTER TABLE card_statements DROP COLUMN over_limit_amount;
ALTER TABLE card_statements DROP COLUMN arrears_count;
ALTER TABLE card_statements DROP COLUMN mora_amount_usd;
ALTER TABLE card_statements DROP COLUMN mora_amount;
ALTER TABLE card_statements DROP COLUMN interest_bonifiable_usd;
ALTER TABLE card_statements DROP COLUMN interest_bonifiable;
