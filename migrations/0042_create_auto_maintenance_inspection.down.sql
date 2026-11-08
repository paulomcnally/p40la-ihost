UPDATE institutions
SET category_id = NULL
WHERE category_id IN (
    SELECT id FROM institution_categories
    WHERE key IN ('workshop', 'mechanical_inspection')
);

DELETE FROM institution_categories
WHERE key IN ('workshop', 'mechanical_inspection');

DROP TABLE IF EXISTS auto_attachments;
DROP TABLE IF EXISTS auto_inspections;
DROP TABLE IF EXISTS auto_maintenances;
