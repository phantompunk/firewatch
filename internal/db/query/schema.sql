-- -- name: GetReportSchema :one
-- SELECT schema FROM report_schema
-- WHERE is_live = ?
-- ORDER BY id DESC
-- LIMIT 1;
--
-- -- name: CountReportSchemas :one
-- SELECT COUNT(*) FROM report_schema;
--
--  -- name: DeleteDraftSchemas :exec
--  DELETE FROM report_schema WHERE is_live = 0;
--
--  -- name: InsertDraftSchema :exec
--  INSERT INTO report_schema (version, is_live, schema, updated_at, updated_by)
--  VALUES (?, 0, ?, CURRENT_TIMESTAMP, ?);
--
-- -- name: DemoteLiveSchemas :exec
-- UPDATE report_schema SET is_live = FALSE WHERE is_live = TRUE;
--
-- -- name: PromoteLatestDraft :exec
-- UPDATE report_schema
-- SET is_live = TRUE, updated_by = ?, updated_at = CURRENT_TIMESTAMP
-- WHERE id = (
--     SELECT id FROM report_schema
--     WHERE is_live = FALSE
--     ORDER BY id DESC
--     LIMIT 1
-- );

-- name: GetReportSchema :one
SELECT schema FROM report_schema
WHERE is_live = ?
ORDER BY id DESC
LIMIT 1;

-- name: CountReportSchemas :one
SELECT COUNT(*) FROM report_schema;

-- name: DeleteDraftSchemas :exec
DELETE FROM report_schema WHERE is_live = 0;

-- name: InsertDraftSchema :exec
INSERT INTO report_schema (version, is_live, schema, updated_at, updated_by)
VALUES (:version, 0, :schema_data, CURRENT_TIMESTAMP, :updated_by);

-- name: DemoteLiveSchemas :exec
UPDATE report_schema SET is_live = 0 WHERE is_live = 1;

-- name: PromoteLatestDraft :exec
UPDATE report_schema
SET is_live = 1, updated_by = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = (
    SELECT id FROM report_schema
    WHERE is_live = 0
    ORDER BY id DESC
    LIMIT 1
);
