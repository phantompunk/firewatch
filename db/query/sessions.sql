-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?);

-- name: GetSessionUserID :one
SELECT user_id FROM sessions
WHERE id = ? AND expires_at > CURRENT_TIMESTAMP;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions WHERE user_id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= CURRENT_TIMESTAMP;
