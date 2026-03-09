CREATE TABLE IF NOT EXISTS report_events (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    submitted_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fields_filled TEXT NOT NULL DEFAULT '[]'
);

CREATE INDEX IF NOT EXISTS report_events_submitted_at_idx ON report_events (submitted_at DESC);
