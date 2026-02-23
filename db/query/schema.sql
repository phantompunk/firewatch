-- name: GetReportSchema :one
SELECT schema FROM report_schema
WHERE is_live = $1
ORDER BY id DESC
LIMIT 1;

-- name: CountReportSchemas :one
SELECT COUNT(*) FROM report_schema;

-- name: UpsertDraftSchema :exec
WITH removed AS (
    DELETE FROM report_schema WHERE is_live = FALSE
)
INSERT INTO report_schema (version, is_live, schema, updated_at, updated_by)
VALUES ($1, FALSE, $2, $3, $4);

-- name: InsertReportSchemaRow :exec
INSERT INTO report_schema (version, is_live, schema, updated_at)
VALUES ($1, $2, $3, NOW());

-- name: DemoteLiveSchemas :exec
UPDATE report_schema SET is_live = FALSE WHERE is_live = TRUE;

-- name: PromoteLatestDraft :exec
UPDATE report_schema
SET is_live = TRUE, updated_by = $1, updated_at = NOW()
WHERE id = (
    SELECT id FROM report_schema
    WHERE is_live = FALSE
    ORDER BY id DESC
    LIMIT 1
);
