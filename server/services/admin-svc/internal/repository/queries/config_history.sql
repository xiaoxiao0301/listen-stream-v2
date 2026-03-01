-- name: CreateConfigHistory :one
INSERT INTO config_histories (
    id, config_key, old_value, new_value, version,
    admin_id, admin_name, reason, rollbackable, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetConfigHistory :one
SELECT * FROM config_histories
WHERE id = $1 LIMIT 1;

-- name: ListConfigHistories :many
SELECT * FROM config_histories
WHERE config_key = COALESCE(sqlc.narg('config_key'), config_key)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetLatestConfigVersion :one
SELECT * FROM config_histories
WHERE config_key = $1
ORDER BY version DESC
LIMIT 1;

-- name: DeleteOldConfigHistories :exec
DELETE FROM config_histories
WHERE created_at < $1;
