CREATE TABLE IF NOT EXISTS invitation_tokens (
    id              TEXT PRIMARY KEY,
    email_encrypted BLOB NOT NULL,
    role            TEXT NOT NULL CHECK (role IN ('admin', 'super_admin')),
    token_hash      TEXT NOT NULL,
    expires_at      TEXT NOT NULL,
    used            INTEGER NOT NULL DEFAULT 0 CHECK (used IN (0, 1))
);
