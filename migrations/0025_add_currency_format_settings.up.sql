-- Configuración de formato de moneda (SPEC-058).
-- Default Nicaragua: miles con coma, decimales con punto, 2 dígitos decimales.
-- Formato de ejemplo: 1,500.00
INSERT INTO system_settings (key, value) VALUES
    ('currency_thousands_separator', ','),
    ('currency_decimal_separator', '.'),
    ('currency_decimal_digits', '2');