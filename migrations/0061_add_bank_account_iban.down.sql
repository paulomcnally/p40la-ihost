-- Revierte el IBAN de las cuentas bancarias (SPEC-162).
-- Solo se pierde el IBAN cargado; el resto de la cuenta se conserva.
ALTER TABLE bank_accounts DROP COLUMN iban;
