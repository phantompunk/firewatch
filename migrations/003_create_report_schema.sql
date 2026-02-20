CREATE TABLE IF NOT EXISTS report_schema (
    id         BIGSERIAL PRIMARY KEY,
    version    INT NOT NULL DEFAULT 1,
    is_live    BOOLEAN NOT NULL DEFAULT FALSE,
    schema     JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by TEXT
);

CREATE INDEX IF NOT EXISTS report_schema_is_live_idx ON report_schema (is_live);
