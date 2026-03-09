-- name: InsertReportEvent :exec
INSERT INTO report_events (submitted_at, fields_filled)
VALUES (CURRENT_TIMESTAMP, ?);

-- name: CountAllReportEvents :one
SELECT COUNT(*) FROM report_events;

-- name: CountReportEventsSince :one
SELECT COUNT(*) FROM report_events WHERE submitted_at >= ?;

-- name: LatestReportEventTime :one
SELECT submitted_at FROM report_events ORDER BY submitted_at DESC LIMIT 1;

-- name: ReportEventsByDay :many
SELECT date(submitted_at) AS day, CAST(COUNT(*) AS INTEGER) AS count
FROM report_events
WHERE submitted_at >= ?
GROUP BY date(submitted_at)
ORDER BY day ASC;
