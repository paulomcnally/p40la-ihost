-- SPEC-094: Vincular categorías de presupuesto con servicios.
-- categories.service_id (FK services) + categories.account_id (FK accounts),
-- transactions.source_bill_id (idempotencia de transacciones auto-generadas).

ALTER TABLE categories ADD COLUMN service_id INTEGER REFERENCES services(id);
ALTER TABLE categories ADD COLUMN account_id INTEGER REFERENCES accounts(id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_service_link
  ON categories(service_id) WHERE service_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_categories_account_link
  ON categories(account_id) WHERE account_id IS NOT NULL;

ALTER TABLE transactions ADD COLUMN source_bill_id INTEGER;

CREATE UNIQUE INDEX IF NOT EXISTS idx_transactions_source_bill
  ON transactions(source_bill_id) WHERE source_bill_id IS NOT NULL;