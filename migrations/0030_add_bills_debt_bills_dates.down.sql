-- Revertir migración 0030: quitar issue_date/due_date de bills e issue_date de debt_bills.
-- Precedente 0029: DROP COLUMN soportado por el driver modernc.org/sqlite.

ALTER TABLE bills DROP COLUMN issue_date;
ALTER TABLE bills DROP COLUMN due_date;

ALTER TABLE debt_bills DROP COLUMN issue_date;