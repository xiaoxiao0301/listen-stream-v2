-- name: CreateAdminUser :one
INSERT INTO admin_users (
    id, username, password_hash, email, role, status,
    totp_secret, totp_enabled, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetAdminUser :one
SELECT * FROM admin_users
WHERE id = $1 LIMIT 1;

-- name: GetAdminUserByUsername :one
SELECT * FROM admin_users
WHERE username = $1 LIMIT 1;

-- name: GetAdminUserByEmail :one
SELECT * FROM admin_users
WHERE email = $1 LIMIT 1;

-- name: ListAdminUsers :many
SELECT * FROM admin_users
WHERE status = COALESCE(sqlc.narg('status'), status)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateAdminUser :one
UPDATE admin_users
SET
    email = COALESCE(sqlc.narg('email'), email),
    role = COALESCE(sqlc.narg('role'), role),
    status = COALESCE(sqlc.narg('status'), status),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash),
    totp_secret = COALESCE(sqlc.narg('totp_secret'), totp_secret),
    totp_enabled = COALESCE(sqlc.narg('totp_enabled'), totp_enabled),
    updated_at = $1
WHERE id = $2
RETURNING *;

-- name: UpdateLastLogin :exec
UPDATE admin_users
SET last_login_at = $1, last_login_ip = $2, updated_at = $3
WHERE id = $4;

-- name: DeleteAdminUser :exec
DELETE FROM admin_users
WHERE id = $1;

-- name: CountAdminUsers :one
SELECT COUNT(*) FROM admin_users
WHERE status = COALESCE(sqlc.narg('status'), status);
