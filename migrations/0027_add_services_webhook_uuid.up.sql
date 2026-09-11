ALTER TABLE services ADD COLUMN webhook_uuid TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_services_webhook_uuid ON services(webhook_uuid) WHERE webhook_uuid IS NOT NULL;