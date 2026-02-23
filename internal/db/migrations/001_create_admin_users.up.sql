PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS admin_users (
    id             TEXT PRIMARY KEY,
    username       TEXT UNIQUE NOT NULL,
    email_hmac     TEXT UNIQUE NOT NULL,
    email_encrypted BLOB NOT NULL,
    password_hash  TEXT NOT NULL,
    role           TEXT NOT NULL CHECK (role IN ('admin', 'super_admin')),
    status         TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at     TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at  TEXT
);
