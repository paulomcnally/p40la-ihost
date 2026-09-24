-- Vínculo servicio ↔ cuenta de p40la-ihost-automation (SPEC-092).
-- Permite que el endpoint de sync sepa qué cuenta de automation ejecutar
-- (el job corre el plugin + el webhook del servicio). Nullable: los servicios
-- sin automation_account_id no son sincronizables por API/bot.

ALTER TABLE services ADD COLUMN automation_account_id INTEGER;