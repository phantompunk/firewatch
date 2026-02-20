-- name: CountAdminUsers :one
SELECT COUNT(*) FROM admin_users;

-- name: CreateAdminUser :exec
INSERT INTO admin_users (id, email, password_hash, role)
VALUES ($1, $2, $3, $4);

-- name: GetAdminUserByEmail :one
SELECT id, email, password_hash, role, status, created_at, last_login_at
FROM admin_users
WHERE email = $1;

-- name: GetAdminUserByID :one
SELECT id, email, role, status, created_at, last_login_at
FROM admin_users
WHERE id = $1;

-- name: ListAdminUsers :many
SELECT id, email, role, status, created_at, last_login_at
FROM admin_users
ORDER BY created_at;

-- name: UpdateAdminUserRoleAndStatus :exec
UPDATE admin_users SET role = $1, status = $2 WHERE id = $3;

-- name: UpdateAdminUserPassword :exec
UPDATE admin_users SET password_hash = $1 WHERE id = $2;

-- name: UpdateAdminUserLastLogin :exec
UPDATE admin_users SET last_login_at = NOW() WHERE id = $1;

-- name: CountActiveSuperAdmins :one
SELECT COUNT(*) FROM admin_users
WHERE role = 'super_admin' AND status = 'active';

-- name: GetAdminUserRoleByID :one
SELECT role FROM admin_users WHERE id = $1;

-- name: DeleteAdminUser :exec
DELETE FROM admin_users WHERE id = $1;
