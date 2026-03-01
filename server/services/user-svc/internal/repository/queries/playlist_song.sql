-- name: AddSongToPlaylist :one
INSERT INTO playlist_songs (
    playlist_id, song_id, song_name, singer_name, position, added_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetPlaylistSong :one
SELECT * FROM playlist_songs
WHERE playlist_id = $1 AND song_id = $2;

-- name: ListPlaylistSongs :many
SELECT * FROM playlist_songs
WHERE playlist_id = $1
ORDER BY position ASC;

-- name: CountPlaylistSongs :one
SELECT COUNT(*) FROM playlist_songs
WHERE playlist_id = $1;

-- name: RemoveSongFromPlaylist :exec
DELETE FROM playlist_songs
WHERE playlist_id = $1 AND song_id = $2;

-- name: UpdateSongPosition :exec
UPDATE playlist_songs
SET position = $3
WHERE playlist_id = $1 AND song_id = $2;

-- name: CheckSongInPlaylist :one
SELECT EXISTS(
    SELECT 1 FROM playlist_songs
    WHERE playlist_id = $1 AND song_id = $2
);

-- name: GetMaxPosition :one
SELECT COALESCE(MAX(position), -1) FROM playlist_songs
WHERE playlist_id = $1;

-- name: DeletePlaylistSongs :exec
DELETE FROM playlist_songs WHERE playlist_id = $1;
