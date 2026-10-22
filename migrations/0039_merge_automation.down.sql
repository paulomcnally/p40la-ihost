-- SPEC-109: revierte el esquema de unificación. No toca tablas de p40la.
DROP INDEX IF EXISTS idx_services_automation_account_id;
DROP INDEX IF EXISTS idx_automation_webhook_logs_account_id;
DROP TABLE IF EXISTS automation_webhook_logs;
DROP INDEX IF EXISTS idx_automation_bills_account_id;
DROP TABLE IF EXISTS automation_bills;
DROP INDEX IF EXISTS idx_automation_accounts_institution_id;
DROP TABLE IF EXISTS automation_accounts;
DROP TABLE IF EXISTS automation_plugins;
