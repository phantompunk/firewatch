-- name: GetSettings :one
SELECT data FROM settings WHERE id = 1;

-- name: UpsertSettings :exec
INSERT INTO settings (id, data, updated_at) VALUES (1, $1, NOW())
ON CONFLICT (id) DO UPDATE
    SET data = EXCLUDED.data,
        updated_at = EXCLUDED.updated_at;
