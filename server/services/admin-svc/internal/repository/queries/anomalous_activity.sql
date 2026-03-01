-- name: CreateAnomalousActivity :one
INSERT INTO anomalous_activities (
    id, type, severity, description, admin_id, admin_name,
    details, resolved, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetAnomalousActivity :one
SELECT * FROM anomalous_activities
WHERE id = $1 LIMIT 1;

-- name: ListAnomalousActivities :many
SELECT * FROM anomalous_activities
WHERE
    admin_id = COALESCE(sqlc.narg('admin_id'), admin_id)
    AND type = COALESCE(sqlc.narg('type'), type)
    AND severity = COALESCE(sqlc.narg('severity'), severity)
    AND resolved = COALESCE(sqlc.narg('resolved'), resolved)
    AND created_at >= COALESCE(sqlc.narg('start_date'), created_at)
    AND created_at <= COALESCE(sqlc.narg('end_date'), created_at)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAnomalousActivities :one
SELECT COUNT(*) FROM anomalous_activities
WHERE
    admin_id = COALESCE(sqlc.narg('admin_id'), admin_id)
    AND type = COALESCE(sqlc.narg('type'), type)
    AND severity = COALESCE(sqlc.narg('severity'), severity)
    AND resolved = COALESCE(sqlc.narg('resolved'), resolved);

-- name: ResolveAnomalousActivity :one
UPDATE anomalous_activities
SET resolved = true, resolved_by = $1, resolved_at = $2
WHERE id = $3
RETURNING *;

-- name: DeleteOldAnomalousActivities :exec
DELETE FROM anomalous_activities
WHERE created_at < $1 AND resolved = true;
