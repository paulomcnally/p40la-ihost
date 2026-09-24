-- SPEC-092: rollback del vínculo servicio ↔ cuenta de automation.

ALTER TABLE services DROP COLUMN automation_account_id;