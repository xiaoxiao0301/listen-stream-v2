-- name: UpsertDailyStats :one
INSERT INTO daily_stats (
    date, total_users, new_users, active_users,
    total_requests, success_requests, failed_requests, error_rate,
    avg_response_time, total_favorites, total_playlists, total_plays, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
ON CONFLICT (date) DO UPDATE SET
    total_users = EXCLUDED.total_users,
    new_users = EXCLUDED.new_users,
    active_users = EXCLUDED.active_users,
    total_requests = EXCLUDED.total_requests,
    success_requests = EXCLUDED.success_requests,
    failed_requests = EXCLUDED.failed_requests,
    error_rate = EXCLUDED.error_rate,
    avg_response_time = EXCLUDED.avg_response_time,
    total_favorites = EXCLUDED.total_favorites,
    total_playlists = EXCLUDED.total_playlists,
    total_plays = EXCLUDED.total_plays
RETURNING *;

-- name: GetDailyStats :one
SELECT * FROM daily_stats
WHERE date = $1 LIMIT 1;

-- name: ListDailyStats :many
SELECT * FROM daily_stats
WHERE date >= $1 AND date <= $2
ORDER BY date DESC;

-- name: DeleteOldDailyStats :exec
DELETE FROM daily_stats
WHERE date < $1;
