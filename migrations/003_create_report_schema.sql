CREATE TABLE IF NOT EXISTS report_schema (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    version    INTEGER NOT NULL DEFAULT 1,
    is_live    INTEGER NOT NULL DEFAULT 0 CHECK (is_live IN (0, 1)),
    schema     TEXT NOT NULL, -- Store JSON as TEXT
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT,
    -- Ensure only one schema is 'live' at a time (Optional logic)
    CONSTRAINT valid_json CHECK (json_valid(schema))
);

CREATE INDEX IF NOT EXISTS report_schema_is_live_idx ON report_schema (is_live);
