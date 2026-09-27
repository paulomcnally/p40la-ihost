-- SPEC-094: Revertir vínculo categoría-presupuesto ↔ servicio.

DROP INDEX IF EXISTS idx_transactions_source_bill;
ALTER TABLE transactions DROP COLUMN source_bill_id;
DROP INDEX IF EXISTS idx_categories_account_link;
DROP INDEX IF EXISTS idx_categories_service_link;
ALTER TABLE categories DROP COLUMN account_id;
ALTER TABLE categories DROP COLUMN service_id;