-- Módulo de Presupuesto (SPEC-093).
-- Tablas: category_groups, categories, accounts, budget_months,
--         recurring_rules, assignments, transactions.
-- Patrón YNAB: assigned/activity/available por categoría y mes.

CREATE TABLE IF NOT EXISTS category_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    icon TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_group_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    icon TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    target_amount REAL,
    target_type TEXT CHECK (target_type IN ('monthly','by_date') OR target_type IS NULL),
    target_date TEXT,
    deleted_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (category_group_id) REFERENCES category_groups(id)
);

CREATE TABLE IF NOT EXISTS accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('checking','savings','credit_card','cash')),
    currency_id INTEGER NOT NULL,
    starting_balance REAL NOT NULL DEFAULT 0,
    deleted_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (currency_id) REFERENCES currencies(id)
);

CREATE TABLE IF NOT EXISTS budget_months (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL CHECK (month BETWEEN 1 AND 12),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(year, month)
);

CREATE TABLE IF NOT EXISTS recurring_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category_id INTEGER NOT NULL,
    amount REAL NOT NULL DEFAULT 0,
    frequency TEXT NOT NULL DEFAULT 'monthly' CHECK (frequency = 'monthly'),
    start_month TEXT NOT NULL,
    end_month TEXT,
    active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE TABLE IF NOT EXISTS assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    budget_month_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    currency_id INTEGER NOT NULL,
    amount REAL NOT NULL DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'one_time' CHECK (source IN ('recurring','one_time')),
    recurring_rule_id INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(budget_month_id, category_id, currency_id),
    FOREIGN KEY (budget_month_id) REFERENCES budget_months(id),
    FOREIGN KEY (category_id) REFERENCES categories(id),
    FOREIGN KEY (currency_id) REFERENCES currencies(id),
    FOREIGN KEY (recurring_rule_id) REFERENCES recurring_rules(id)
);

CREATE TABLE IF NOT EXISTS transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    category_id INTEGER,
    currency_id INTEGER NOT NULL,
    date TEXT NOT NULL,
    payee TEXT,
    memo TEXT,
    outflow REAL NOT NULL DEFAULT 0,
    inflow REAL NOT NULL DEFAULT 0,
    cleared INTEGER NOT NULL DEFAULT 0,
    deleted_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (account_id) REFERENCES accounts(id),
    FOREIGN KEY (category_id) REFERENCES categories(id),
    FOREIGN KEY (currency_id) REFERENCES currencies(id)
);

CREATE INDEX IF NOT EXISTS idx_categories_group ON categories(category_group_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_category_date ON transactions(category_id, date);
CREATE INDEX IF NOT EXISTS idx_assignments_month_category ON assignments(budget_month_id, category_id);
CREATE INDEX IF NOT EXISTS idx_recurring_rules_category ON recurring_rules(category_id);