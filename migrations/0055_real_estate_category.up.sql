-- SPEC-140: categoría canónica para inmobiliarias.
-- Idempotente: no duplica si el usuario ya la creó a mano.

INSERT INTO institution_categories (key, name, description, icon_key)
VALUES (
    'real_estate',
    'Inmobiliaria',
    'Bienes raíces, corretaje y administración de propiedades',
    'home'
)
ON CONFLICT(key) DO NOTHING;
