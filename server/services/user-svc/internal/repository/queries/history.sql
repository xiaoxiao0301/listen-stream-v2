-- name: CreatePlayHistory :one
INSERT INTO play_histories (
    id, user_id, song_id, song_name, singer_name, album_cover, duration, played_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetPlayHistory :one
SELECT * FROM play_histories WHERE id = $1;

-- name: ListPlayHistoriesByUser :many
SELECT * FROM play_histories
WHERE user_id = $1
ORDER BY played_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPlayHistoriesByUser :one
SELECT COUNT(*) FROM play_histories
WHERE user_id = $1;

-- name: DeletePlayHistory :exec
DELETE FROM play_histories WHERE id = $1;

-- name: DeleteOldestPlayHistories :exec
DELETE FROM play_histories
WHERE id IN (
    SELECT id FROM play_histories
    WHERE user_id = $1
    ORDER BY played_at ASC
    LIMIT $2
);

-- name: CleanupOldPlayHistories :exec
DELETE FROM play_histories
WHERE user_id = $1
AND id NOT IN (
    SELECT id FROM play_histories
    WHERE user_id = $1
    ORDER BY played_at DESC
    LIMIT $2
);
