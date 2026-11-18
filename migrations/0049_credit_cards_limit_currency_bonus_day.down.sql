-- SPEC-138: revierte las cuatro columnas agregadas a credit_cards.

ALTER TABLE credit_cards DROP COLUMN category;
ALTER TABLE credit_cards DROP COLUMN bonus_due_day;
ALTER TABLE credit_cards DROP COLUMN exchange_rate;
ALTER TABLE credit_cards DROP COLUMN limit_currency;
