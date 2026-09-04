DELETE FROM system_settings WHERE key IN (
    'currency_thousands_separator',
    'currency_decimal_separator',
    'currency_decimal_digits'
);