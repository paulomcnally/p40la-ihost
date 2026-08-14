CREATE TABLE IF NOT EXISTS currencies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    deleted_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO currencies (code, name, symbol) VALUES
    ('NIO', 'Córdoba Nicaragüense', 'C$'),
    ('USD', 'Dólar Estadounidense', '$')
ON CONFLICT(code) DO NOTHING;
