ALTER TABLE auto_maintenances ADD COLUMN labor_cost REAL NOT NULL DEFAULT 0 CHECK (labor_cost >= 0);
ALTER TABLE auto_maintenances ADD COLUMN supplies_cost REAL NOT NULL DEFAULT 0 CHECK (supplies_cost >= 0);

UPDATE auto_maintenances
SET labor_cost = COALESCE((
    SELECT SUM(amount) FROM auto_maintenance_items
    WHERE maintenance_id = auto_maintenances.id AND item_type = 'labor'
), 0);

UPDATE auto_maintenances
SET supplies_cost = COALESCE((
    SELECT SUM(amount) FROM auto_maintenance_items
    WHERE maintenance_id = auto_maintenances.id AND item_type = 'supplies'
), 0);

DROP TABLE IF EXISTS auto_maintenance_items;
