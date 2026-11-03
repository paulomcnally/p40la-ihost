-- SPEC-119: revierte el importador de gastos escolares y pensión.

DROP TABLE IF EXISTS pension_import_rows;
DROP TABLE IF EXISTS pension_import_batches;
DROP TABLE IF EXISTS pension_category_keywords;

DROP TRIGGER IF EXISTS trg_pension_categories_share_insert;
DROP TRIGGER IF EXISTS trg_pension_categories_share_update;

ALTER TABLE pension_categories DROP COLUMN currency;
ALTER TABLE pension_categories DROP COLUMN share_percent;
