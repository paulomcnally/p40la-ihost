-- SPEC-140: apartados (reservas) con vencimiento y bloqueo de promoción.

CREATE TABLE IF NOT EXISTS rental_reservations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    home_id INTEGER NOT NULL,
    applicant_person_id INTEGER NOT NULL,
    agent_person_id INTEGER,
    converted_lease_id INTEGER,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'converted', 'expired', 'cancelled', 'forfeited')),
    amount REAL NOT NULL DEFAULT 0 CHECK (amount >= 0),
    currency TEXT NOT NULL DEFAULT 'USD',
    reference TEXT NOT NULL DEFAULT '',
    applied_to_deposit INTEGER NOT NULL DEFAULT 1,
    starts_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    contract_date TEXT NOT NULL DEFAULT '',
    move_in_date TEXT NOT NULL DEFAULT '',
    rent_amount REAL NOT NULL DEFAULT 0 CHECK (rent_amount >= 0),
    maintenance_amount REAL NOT NULL DEFAULT 0 CHECK (maintenance_amount >= 0),
    owner_split_percent REAL NOT NULL DEFAULT 50 CHECK (owner_split_percent BETWEEN 0 AND 100),
    agency_split_percent REAL NOT NULL DEFAULT 50 CHECK (agency_split_percent BETWEEN 0 AND 100),
    refund_policy TEXT NOT NULL DEFAULT 'forfeitable' CHECK (refund_policy IN ('refundable', 'forfeitable', 'custom')),
    required_documents TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (home_id) REFERENCES homes(id) ON DELETE RESTRICT,
    FOREIGN KEY (applicant_person_id) REFERENCES rental_people(id) ON DELETE RESTRICT,
    FOREIGN KEY (agent_person_id) REFERENCES rental_people(id) ON DELETE SET NULL,
    FOREIGN KEY (converted_lease_id) REFERENCES rental_leases(id) ON DELETE SET NULL,
    CHECK (expires_at > starts_at),
    CHECK (owner_split_percent + agency_split_percent <= 100.001)
);

CREATE INDEX IF NOT EXISTS idx_rental_reservations_home ON rental_reservations(home_id);
CREATE INDEX IF NOT EXISTS idx_rental_reservations_status ON rental_reservations(status);
CREATE INDEX IF NOT EXISTS idx_rental_reservations_expires ON rental_reservations(expires_at);

-- Una sola reserva activa por casa (ADR-002: el estado de promoción es derivado).
CREATE UNIQUE INDEX IF NOT EXISTS idx_rental_reservations_active_home
    ON rental_reservations(home_id) WHERE status = 'active';
