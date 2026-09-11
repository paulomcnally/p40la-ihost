DROP INDEX IF EXISTS idx_services_webhook_uuid;
ALTER TABLE services DROP COLUMN webhook_uuid;