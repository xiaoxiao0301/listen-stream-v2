-- name: CreateFavorite :one
INSERT INTO favorites (
    id, user_id, song_id, song_name, singer_name, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetFavorite :one
SELECT * FROM favorites
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetFavoriteByUserAndSong :one
SELECT * FROM favorites
WHERE user_id = $1 AND song_id = $2 AND deleted_at IS NULL;

-- name: ListFavoritesByUser :many
SELECT * FROM favorites
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountFavoritesByUser :one
SELECT COUNT(*) FROM favorites
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteFavorite :exec
UPDATE favorites
SET deleted_at = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: RestoreFavorite :exec
UPDATE favorites
SET deleted_at = NULL
WHERE id = $1 AND deleted_at IS NOT NULL;

-- name: HardDeleteFavorite :exec
DELETE FROM favorites WHERE id = $1;

-- name: CheckFavoriteExists :one
SELECT EXISTS(
    SELECT 1 FROM favorites
    WHERE user_id = $1 AND song_id = $2 AND deleted_at IS NULL
);
