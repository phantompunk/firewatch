-- name: CreateInvite :exec
INSERT INTO invitation_tokens (id, email, role, token_hash, expires_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetInviteByTokenHash :one
SELECT id, email, role, token_hash, expires_at, used
FROM invitation_tokens
WHERE token_hash = $1
  AND used = FALSE
  AND expires_at > NOW();

-- name: MarkInviteUsed :exec
UPDATE invitation_tokens SET used = TRUE WHERE id = $1;
