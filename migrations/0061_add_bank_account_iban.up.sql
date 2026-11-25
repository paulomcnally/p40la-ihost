-- Agrega el IBAN a las cuentas bancarias (SPEC-162).
-- Campo opcional en la práctica: las cuentas existentes quedan con ''.
ALTER TABLE bank_accounts ADD COLUMN iban TEXT NOT NULL DEFAULT '';
