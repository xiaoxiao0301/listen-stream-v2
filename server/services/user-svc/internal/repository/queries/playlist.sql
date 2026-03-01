-- name: CreateUserPlaylist :one
INSERT INTO user_playlists (
    id, user_id, name, description, cover_url, song_count, is_public, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetUserPlaylist :one
SELECT * FROM user_playlists
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListUserPlaylistsByUser :many
SELECT * FROM user_playlists
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPublicPlaylists :many
SELECT * FROM user_playlists
WHERE is_public = TRUE AND deleted_at IS NULL
ORDER BY updated_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUserPlaylistsByUser :one
SELECT COUNT(*) FROM user_playlists
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: UpdateUserPlaylist :exec
UPDATE user_playlists
SET name = $2, description = $3, cover_url = $4, is_public = $5, updated_at = $6
WHERE id = $1 AND deleted_at IS NULL;

-- name: IncrementPlaylistSongCount :exec
UPDATE user_playlists
SET song_count = song_count + 1, updated_at = $2
WHERE id = $1;

-- name: DecrementPlaylistSongCount :exec
UPDATE user_playlists
SET song_count = song_count - 1, updated_at = $2
WHERE id = $1 AND song_count > 0;

-- name: SoftDeleteUserPlaylist :exec
UPDATE user_playlists
SET deleted_at = $2
WHERE id = $1 AND deleted_at IS NULL;

-- name: RestoreUserPlaylist :exec
UPDATE user_playlists
SET deleted_at = NULL
WHERE id = $1 AND deleted_at IS NOT NULL;

-- name: HardDeleteUserPlaylist :exec
DELETE FROM user_playlists WHERE id = $1;
