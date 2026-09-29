-- SPEC-096: revertir la tabla de enlace category_service_links.
-- Se restaura categories.service_id (con el primer servicio de cada categoría)
-- y se elimina la tabla de enlace.

ALTER TABLE categories ADD COLUMN service_id INTEGER REFERENCES services(id);

UPDATE categories SET service_id = (
    SELECT service_id FROM category_service_links
    WHERE category_id = categories.id
    ORDER BY service_id LIMIT 1
) WHERE id IN (SELECT category_id FROM category_service_links);

CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_service_link
    ON categories(service_id) WHERE service_id IS NOT NULL AND deleted_at IS NULL;

DROP TABLE IF EXISTS category_service_links;