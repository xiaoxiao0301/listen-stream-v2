-- name: CreateDevice :one
INSERT INTO devices (
    id, user_id, device_name, fingerprint, platform, 
    app_version, last_ip, last_login_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetDeviceByID :one
SELECT * FROM devices
WHERE id = $1 LIMIT 1;

-- name: GetDeviceByFingerprint :one
SELECT * FROM devices
WHERE user_id = $1 AND fingerprint = $2 LIMIT 1;

-- name: ListDevicesByUserID :many
SELECT * FROM devices
WHERE user_id = $1
ORDER BY last_login_at DESC;

-- name: CountDevicesByUserID :one
SELECT COUNT(*) FROM devices
WHERE user_id = $1;

-- name: UpdateDeviceLoginInfo :exec
UPDATE devices
SET last_ip = $2, last_login_at = $3
WHERE id = $1;

-- name: DeleteDevice :exec
DELETE FROM devices
WHERE id = $1;

-- name: DeleteDevicesByUserID :exec
DELETE FROM devices
WHERE user_id = $1;

-- name: DeleteInactiveDevices :exec
DELETE FROM devices
WHERE last_login_at < $1;
