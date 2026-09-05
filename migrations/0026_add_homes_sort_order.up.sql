ALTER TABLE homes ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

-- Backfill: preserva el orden alfabético actual (visualización previa por ORDER BY name).
UPDATE homes
SET sort_order = (
    SELECT seq FROM (
        SELECT id, ROW_NUMBER() OVER (ORDER BY name) - 1 AS seq
        FROM homes
        WHERE deleted_at IS NULL
    ) ranked
    WHERE ranked.id = homes.id
)
WHERE deleted_at IS NULL;