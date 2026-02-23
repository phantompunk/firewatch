-- name: CreateInvite :exec
INSERT INTO invitation_tokens (id, email_encrypted, role, token_hash, expires_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetInviteByTokenHash :one
SELECT id, email_encrypted, role, token_hash, expires_at, used
FROM invitation_tokens
WHERE token_hash = ?
  AND used = FALSE
  AND expires_at > CURRENT_TIMESTAMP;

-- name: MarkInviteUsed :exec
UPDATE invitation_tokens SET used = TRUE WHERE id = ?;
