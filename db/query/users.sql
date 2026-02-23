-- name: CountAdminUsers :one
SELECT COUNT(*) FROM admin_users;

-- name: CreateAdminUser :exec
INSERT INTO admin_users (id, username, email_hmac, email_encrypted, password_hash, role)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetAdminUserByEmailHMAC :one
SELECT id, username, email_encrypted, email_hmac, password_hash, role, status, created_at, last_login_at
FROM admin_users
WHERE email_hmac = ?;

-- name: GetAdminUserByUsername :one
SELECT id, username, email_encrypted, email_hmac, password_hash, role, status, created_at, last_login_at
FROM admin_users
WHERE username = ?;

-- name: GetAdminUserByID :one
SELECT id, username, role, status, created_at, last_login_at
FROM admin_users
WHERE id = ?;

-- name: ListAdminUsers :many
SELECT id, username, role, status, created_at, last_login_at
FROM admin_users
ORDER BY created_at;

-- name: UpdateAdminUserRoleAndStatus :exec
UPDATE admin_users SET role = ?, status = ? WHERE id = ?;

-- name: UpdateAdminUserPassword :exec
UPDATE admin_users SET password_hash = ? WHERE id = ?;

-- name: UpdateAdminUserLastLogin :exec
UPDATE admin_users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: CountActiveSuperAdmins :one
SELECT COUNT(*) FROM admin_users
WHERE role = 'super_admin' AND status = 'active';

-- name: GetAdminUserRoleByID :one
SELECT role FROM admin_users WHERE id = ?;

-- name: DeleteAdminUser :exec
DELETE FROM admin_users WHERE id = ?;

-- name: GetAdminUserEmailEncryptedByID :one
SELECT email_encrypted FROM admin_users WHERE id = ?;
