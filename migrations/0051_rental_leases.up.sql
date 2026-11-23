-- SPEC-140: personas, contratos de alquiler y partes.
-- Importes en REAL (convención del proyecto), fechas de negocio en TEXT ISO.

ALTER TABLE homes ADD COLUMN listing_enabled INTEGER NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS rental_people (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    tax_id TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    occupation TEXT NOT NULL DEFAULT '',
    employer TEXT NOT NULL DEFAULT '',
    monthly_income REAL NOT NULL DEFAULT 0 CHECK (monthly_income >= 0),
    marital_status TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    deleted_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rental_people_name ON rental_people(full_name COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_rental_people_email ON rental_people(email);

CREATE TABLE IF NOT EXISTS rental_leases (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    home_id INTEGER NOT NULL,
    agency_institution_id INTEGER,
    agent_person_id INTEGER,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'ended', 'terminated', 'cancelled')),
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    term_months INTEGER NOT NULL CHECK (term_months BETWEEN 1 AND 120),
    rent_amount REAL NOT NULL DEFAULT 0 CHECK (rent_amount >= 0),
    maintenance_amount REAL NOT NULL DEFAULT 0 CHECK (maintenance_amount >= 0),
    currency TEXT NOT NULL DEFAULT 'USD',
    payment_day INTEGER NOT NULL DEFAULT 1 CHECK (payment_day BETWEEN 1 AND 28),
    deposit_amount REAL NOT NULL DEFAULT 0 CHECK (deposit_amount >= 0),
    late_fee_type TEXT NOT NULL DEFAULT 'none' CHECK (late_fee_type IN ('none', 'fixed', 'percent')),
    late_fee_value REAL NOT NULL DEFAULT 0 CHECK (late_fee_value >= 0),
    late_fee_cap REAL NOT NULL DEFAULT 0 CHECK (late_fee_cap >= 0),
    auto_renew INTEGER NOT NULL DEFAULT 0,
    notice_days INTEGER NOT NULL DEFAULT 30 CHECK (notice_days >= 0),
    commission_percent REAL NOT NULL DEFAULT 0 CHECK (commission_percent BETWEEN 0 AND 100),
    commission_base TEXT NOT NULL DEFAULT 'collected' CHECK (commission_base IN ('collected', 'invoiced')),
    owner_bank_account TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    deleted_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (home_id) REFERENCES homes(id) ON DELETE RESTRICT,
    FOREIGN KEY (agency_institution_id) REFERENCES institutions(id) ON DELETE SET NULL,
    FOREIGN KEY (agent_person_id) REFERENCES rental_people(id) ON DELETE SET NULL,
    CHECK (end_date > start_date)
);

CREATE INDEX IF NOT EXISTS idx_rental_leases_home ON rental_leases(home_id);
CREATE INDEX IF NOT EXISTS idx_rental_leases_status ON rental_leases(status);
CREATE INDEX IF NOT EXISTS idx_rental_leases_dates ON rental_leases(start_date, end_date);

CREATE TABLE IF NOT EXISTS rental_lease_parties (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    lease_id INTEGER NOT NULL,
    person_id INTEGER NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('tenant', 'guarantor', 'owner', 'agency_contact', 'agent')),
    notify_email INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (lease_id) REFERENCES rental_leases(id) ON DELETE CASCADE,
    FOREIGN KEY (person_id) REFERENCES rental_people(id) ON DELETE CASCADE,
    UNIQUE (lease_id, person_id, role)
);

CREATE INDEX IF NOT EXISTS idx_rental_lease_parties_lease ON rental_lease_parties(lease_id);
