package repository

import (
	"context"

	"user-svc/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PlaylistSongRepositoryImpl 歌单歌曲仓储实现
type PlaylistSongRepositoryImpl struct {
	db *pgxpool.Pool
}

// NewPlaylistSongRepository 创建歌单歌曲仓储
func NewPlaylistSongRepository(db *pgxpool.Pool) PlaylistSongRepository {
	return &PlaylistSongRepositoryImpl{db: db}
}

// Add 添加歌曲到歌单
func (r *PlaylistSongRepositoryImpl) Add(ctx context.Context, ps *domain.PlaylistSong) error {
	query := `
		INSERT INTO playlist_songs (playlist_id, song_id, song_name, singer_name, position, added_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		ps.PlaylistID,
		ps.SongID,
		ps.SongName,
		ps.SingerName,
		ps.Position,
		ps.AddedAt,
	)
	return err
}

// Get 获取歌单中的歌曲
func (r *PlaylistSongRepositoryImpl) Get(ctx context.Context, playlistID, songID string) (*domain.PlaylistSong, error) {
	query := `
		SELECT playlist_id, song_id, song_name, singer_name, position, added_at
		FROM playlist_songs
		WHERE playlist_id = $1 AND song_id = $2
	`
	var ps domain.PlaylistSong
	err := r.db.QueryRow(ctx, query, playlistID, songID).Scan(
		&ps.PlaylistID,
		&ps.SongID,
		&ps.SongName,
		&ps.SingerName,
		&ps.Position,
		&ps.AddedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ps, nil
}

// List 获取歌单的所有歌曲
func (r *PlaylistSongRepositoryImpl) List(ctx context.Context, playlistID string) ([]*domain.PlaylistSong, error) {
	query := `
		SELECT playlist_id, song_id, song_name, singer_name, position, added_at
		FROM playlist_songs
		WHERE playlist_id = $1
		ORDER BY position ASC
	`
	rows, err := r.db.Query(ctx, query, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []*domain.PlaylistSong
	for rows.Next() {
		var ps domain.PlaylistSong
		err := rows.Scan(
			&ps.PlaylistID,
			&ps.SongID,
			&ps.SongName,
			&ps.SingerName,
			&ps.Position,
			&ps.AddedAt,
		)
		if err != nil {
			return nil, err
		}
		songs = append(songs, &ps)
	}
	return songs, rows.Err()
}

// Count 统计歌单的歌曲数量
func (r *PlaylistSongRepositoryImpl) Count(ctx context.Context, playlistID string) (int64, error) {
	query := `SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = $1`
	var count int64
	err := r.db.QueryRow(ctx, query, playlistID).Scan(&count)
	return count, err
}

// Remove 从歌单移除歌曲
func (r *PlaylistSongRepositoryImpl) Remove(ctx context.Context, playlistID, songID string) error {
	query := `DELETE FROM playlist_songs WHERE playlist_id = $1 AND song_id = $2`
	_, err := r.db.Exec(ctx, query, playlistID, songID)
	return err
}

// UpdatePosition 更新歌曲位置
func (r *PlaylistSongRepositoryImpl) UpdatePosition(ctx context.Context, playlistID, songID string, position int) error {
	query := `UPDATE playlist_songs SET position = $3 WHERE playlist_id = $1 AND song_id = $2`
	_, err := r.db.Exec(ctx, query, playlistID, songID, position)
	return err
}

// Exists 检查歌曲是否在歌单中
func (r *PlaylistSongRepositoryImpl) Exists(ctx context.Context, playlistID, songID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM playlist_songs
			WHERE playlist_id = $1 AND song_id = $2
		)
	`
	var exists bool
	err := r.db.QueryRow(ctx, query, playlistID, songID).Scan(&exists)
	return exists, err
}

// GetMaxPosition 获取最大位置
func (r *PlaylistSongRepositoryImpl) GetMaxPosition(ctx context.Context, playlistID string) (int, error) {
	query := `SELECT COALESCE(MAX(position), -1) FROM playlist_songs WHERE playlist_id = $1`
	var maxPos int
	err := r.db.QueryRow(ctx, query, playlistID).Scan(&maxPos)
	return maxPos, err
}

// DeleteAll 删除歌单的所有歌曲
func (r *PlaylistSongRepositoryImpl) DeleteAll(ctx context.Context, playlistID string) error {
	query := `DELETE FROM playlist_songs WHERE playlist_id = $1`
	_, err := r.db.Exec(ctx, query, playlistID)
	return err
}
