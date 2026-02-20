CREATE TABLE IF NOT EXISTS settings (
    id         INT PRIMARY KEY DEFAULT 1,
    data       BYTEA NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT single_row CHECK (id = 1)
);
