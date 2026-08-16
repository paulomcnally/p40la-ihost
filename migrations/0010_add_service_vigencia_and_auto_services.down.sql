DROP TABLE IF EXISTS auto_services;
ALTER TABLE services DROP COLUMN start_date;
ALTER TABLE services DROP COLUMN end_date;
ALTER TABLE services DROP COLUMN is_recurring;
