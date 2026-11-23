-- SPEC-140: recordatorios recurrentes y deduplicación de envíos.

CREATE TABLE IF NOT EXISTS rental_reminders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    home_id INTEGER,
    lease_id INTEGER,
    kind TEXT NOT NULL DEFAULT 'custom' CHECK (kind IN ('gardening', 'maintenance', 'inspection', 'payment_reminder', 'custom')),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    schedule_type TEXT NOT NULL DEFAULT 'once' CHECK (schedule_type IN ('once', 'monthly', 'every_n_days', 'yearly')),
    interval_n INTEGER NOT NULL DEFAULT 1 CHECK (interval_n > 0),
    anchor_date TEXT NOT NULL,
    next_due_at TEXT NOT NULL,
    lead_days INTEGER NOT NULL DEFAULT 3 CHECK (lead_days >= 0),
    notify_roles TEXT NOT NULL DEFAULT 'tenant',
    extra_emails TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (home_id) REFERENCES homes(id) ON DELETE CASCADE,
    FOREIGN KEY (lease_id) REFERENCES rental_leases(id) ON DELETE CASCADE,
    CHECK (home_id IS NOT NULL OR lease_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_rental_reminders_next ON rental_reminders(enabled, next_due_at);
CREATE INDEX IF NOT EXISTS idx_rental_reminders_lease ON rental_reminders(lease_id);

CREATE TABLE IF NOT EXISTS rental_reminder_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reminder_id INTEGER NOT NULL,
    occurrence_key TEXT NOT NULL,
    due_at TEXT NOT NULL,
    sent_at DATETIME,
    result TEXT NOT NULL DEFAULT 'sent' CHECK (result IN ('sent', 'failed', 'skipped')),
    error_code TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (reminder_id) REFERENCES rental_reminders(id) ON DELETE CASCADE,
    UNIQUE (reminder_id, occurrence_key)
);

CREATE TABLE IF NOT EXISTS rental_email_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL,
    owner_id INTEGER NOT NULL,
    occurrence_key TEXT NOT NULL,
    recipients_count INTEGER NOT NULL DEFAULT 0,
    recipient_roles TEXT NOT NULL DEFAULT '',
    result TEXT NOT NULL DEFAULT 'sent',
    error_code TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (kind, owner_id, occurrence_key)
);
