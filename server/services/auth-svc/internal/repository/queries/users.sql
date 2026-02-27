-- name: CreateUser :one
INSERT INTO users (
    id, phone, token_version, is_active, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByPhone :one
SELECT * FROM users
WHERE phone = $1 LIMIT 1;

-- name: UpdateUserTokenVersion :exec
UPDATE users
SET token_version = $2, updated_at = $3
WHERE id = $1;

-- name: UpdateUserActive :exec
UPDATE users
SET is_active = $2, updated_at = $3
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CountActiveUsers :one
SELECT COUNT(*) FROM users
WHERE is_active = true;
