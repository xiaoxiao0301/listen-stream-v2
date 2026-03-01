package repository

import (
	"context"
	"time"

	"user-svc/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PlaylistRepositoryImpl 歌单仓储实现
type PlaylistRepositoryImpl struct {
	db *pgxpool.Pool
}

// NewPlaylistRepository 创建歌单仓储
func NewPlaylistRepository(db *pgxpool.Pool) PlaylistRepository {
	return &PlaylistRepositoryImpl{db: db}
}

// Create 创建歌单
func (r *PlaylistRepositoryImpl) Create(ctx context.Context, playlist *domain.UserPlaylist) error {
	query := `
		INSERT INTO user_playlists (id, user_id, name, description, cover_url, song_count, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query,
		playlist.ID,
		playlist.UserID,
		playlist.Name,
		playlist.Description,
		playlist.CoverURL,
		playlist.SongCount,
		playlist.IsPublic,
		playlist.CreatedAt,
		playlist.UpdatedAt,
	)
	return err
}

// GetByID 根据ID获取歌单
func (r *PlaylistRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.UserPlaylist, error) {
	query := `
		SELECT id, user_id, name, description, cover_url, song_count, is_public, deleted_at, created_at, updated_at
		FROM user_playlists
		WHERE id = $1 AND deleted_at IS NULL
	`
	var playlist domain.UserPlaylist
	err := r.db.QueryRow(ctx, query, id).Scan(
		&playlist.ID,
		&playlist.UserID,
		&playlist.Name,
		&playlist.Description,
		&playlist.CoverURL,
		&playlist.SongCount,
		&playlist.IsPublic,
		&playlist.DeletedAt,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}

// ListByUser 获取用户的歌单列表
func (r *PlaylistRepositoryImpl) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.UserPlaylist, error) {
	query := `
		SELECT id, user_id, name, description, cover_url, song_count, is_public, deleted_at, created_at, updated_at
		FROM user_playlists
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []*domain.UserPlaylist
	for rows.Next() {
		var playlist domain.UserPlaylist
		err := rows.Scan(
			&playlist.ID,
			&playlist.UserID,
			&playlist.Name,
			&playlist.Description,
			&playlist.CoverURL,
			&playlist.SongCount,
			&playlist.IsPublic,
			&playlist.DeletedAt,
			&playlist.CreatedAt,
			&playlist.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		playlists = append(playlists, &playlist)
	}
	return playlists, rows.Err()
}

// ListPublic 获取公开歌单列表
func (r *PlaylistRepositoryImpl) ListPublic(ctx context.Context, limit, offset int) ([]*domain.UserPlaylist, error) {
	query := `
		SELECT id, user_id, name, description, cover_url, song_count, is_public, deleted_at, created_at, updated_at
		FROM user_playlists
		WHERE is_public = TRUE AND deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []*domain.UserPlaylist
	for rows.Next() {
		var playlist domain.UserPlaylist
		err := rows.Scan(
			&playlist.ID,
			&playlist.UserID,
			&playlist.Name,
			&playlist.Description,
			&playlist.CoverURL,
			&playlist.SongCount,
			&playlist.IsPublic,
			&playlist.DeletedAt,
			&playlist.CreatedAt,
			&playlist.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		playlists = append(playlists, &playlist)
	}
	return playlists, rows.Err()
}

// Count 统计用户的歌单数量
func (r *PlaylistRepositoryImpl) Count(ctx context.Context, userID string) (int64, error) {
	query := `SELECT COUNT(*) FROM user_playlists WHERE user_id = $1 AND deleted_at IS NULL`
	var count int64
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// Update 更新歌单
func (r *PlaylistRepositoryImpl) Update(ctx context.Context, playlist *domain.UserPlaylist) error {
	query := `
		UPDATE user_playlists
		SET name = $2, description = $3, cover_url = $4, is_public = $5, updated_at = $6
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.Exec(ctx, query,
		playlist.ID,
		playlist.Name,
		playlist.Description,
		playlist.CoverURL,
		playlist.IsPublic,
		playlist.UpdatedAt,
	)
	return err
}

// IncrementSongCount 增加歌曲数量
func (r *PlaylistRepositoryImpl) IncrementSongCount(ctx context.Context, playlistID string) error {
	query := `
		UPDATE user_playlists
		SET song_count = song_count + 1, updated_at = $2
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, playlistID, time.Now())
	return err
}

// DecrementSongCount 减少歌曲数量
func (r *PlaylistRepositoryImpl) DecrementSongCount(ctx context.Context, playlistID string) error {
	query := `
		UPDATE user_playlists
		SET song_count = song_count - 1, updated_at = $2
		WHERE id = $1 AND song_count > 0
	`
	_, err := r.db.Exec(ctx, query, playlistID, time.Now())
	return err
}

// SoftDelete 软删除歌单
func (r *PlaylistRepositoryImpl) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE user_playlists SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, id, time.Now())
	return err
}

// Restore 恢复歌单
func (r *PlaylistRepositoryImpl) Restore(ctx context.Context, id string) error {
	query := `UPDATE user_playlists SET deleted_at = NULL WHERE id = $1 AND deleted_at IS NOT NULL`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// HardDelete 硬删除歌单
func (r *PlaylistRepositoryImpl) HardDelete(ctx context.Context, id string) error {
	query := `DELETE FROM user_playlists WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
