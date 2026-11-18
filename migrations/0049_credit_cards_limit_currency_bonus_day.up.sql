-- SPEC-138: límite en una moneda, tipo de cambio, fecha de bonificación y categoría.
-- Aditiva: no reescribe datos existentes (los límites ya cargados se conservan).

ALTER TABLE credit_cards ADD COLUMN limit_currency TEXT NOT NULL DEFAULT 'NIO'
    CHECK (limit_currency IN ('NIO', 'USD'));
ALTER TABLE credit_cards ADD COLUMN exchange_rate REAL NOT NULL DEFAULT 0
    CHECK (exchange_rate >= 0);
ALTER TABLE credit_cards ADD COLUMN bonus_due_day INTEGER NOT NULL DEFAULT 0
    CHECK (bonus_due_day BETWEEN 0 AND 31);
ALTER TABLE credit_cards ADD COLUMN category TEXT NOT NULL DEFAULT ''
    CHECK (category IN ('', 'classic', 'gold', 'platinum', 'black', 'business', 'other'));
