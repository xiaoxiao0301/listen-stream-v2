-- name: CreateOperationLog :one
INSERT INTO operation_logs (
    id, admin_id, admin_name, operation, resource, resource_id,
    action, details, request_id, ip, user_agent, status, error_msg, duration, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
) RETURNING *;

-- name: GetOperationLog :one
SELECT * FROM operation_logs
WHERE id = $1 LIMIT 1;

-- name: ListOperationLogs :many
SELECT * FROM operation_logs
WHERE 
    admin_id = COALESCE(sqlc.narg('admin_id'), admin_id)
    AND operation = COALESCE(sqlc.narg('operation'), operation)
    AND resource = COALESCE(sqlc.narg('resource'), resource)
    AND status = COALESCE(sqlc.narg('status'), status)
    AND created_at >= COALESCE(sqlc.narg('start_date'), created_at)
    AND created_at <= COALESCE(sqlc.narg('end_date'), created_at)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountOperationLogs :one
SELECT COUNT(*) FROM operation_logs
WHERE
    admin_id = COALESCE(sqlc.narg('admin_id'), admin_id)
    AND operation = COALESCE(sqlc.narg('operation'), operation)
    AND resource = COALESCE(sqlc.narg('resource'), resource)
    AND status = COALESCE(sqlc.narg('status'), status)
    AND created_at >= COALESCE(sqlc.narg('start_date'), created_at)
    AND created_at <= COALESCE(sqlc.narg('end_date'), created_at);

-- name: ListOperationLogsByRequestID :many
SELECT * FROM operation_logs
WHERE request_id = $1
ORDER BY created_at ASC;

-- name: DeleteOldOperationLogs :exec
DELETE FROM operation_logs
WHERE created_at < $1;
