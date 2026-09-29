-- SPEC-096: Categorías de presupuesto vinculadas a múltiples servicios.
-- Se reemplaza categories.service_id (1 servicio → 1 categoría) por una tabla
-- de enlace category_service_links (1 categoría → N servicios).
-- categories.account_id se conserva (una cuenta destino por categoría).

CREATE TABLE IF NOT EXISTS category_service_links (
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    service_id  INTEGER NOT NULL REFERENCES services(id),
    PRIMARY KEY (category_id, service_id)
);

CREATE INDEX IF NOT EXISTS idx_category_service_links_service
    ON category_service_links(service_id);

-- Migrar vínculos existentes desde categories.service_id.
INSERT INTO category_service_links (category_id, service_id)
    SELECT id, service_id FROM categories
    WHERE service_id IS NOT NULL;

DROP INDEX IF EXISTS idx_categories_service_link;
ALTER TABLE categories DROP COLUMN service_id;