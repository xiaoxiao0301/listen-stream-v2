package repository

import (
	"context"
	"time"

	"user-svc/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FavoriteRepositoryImpl 收藏仓储实现
type FavoriteRepositoryImpl struct {
	db *pgxpool.Pool
}

// NewFavoriteRepository 创建收藏仓储
func NewFavoriteRepository(db *pgxpool.Pool) FavoriteRepository {
	return &FavoriteRepositoryImpl{db: db}
}

// Create 创建收藏
func (r *FavoriteRepositoryImpl) Create(ctx context.Context, favorite *domain.Favorite) error {
	query := `
		INSERT INTO favorites (id, user_id, song_id, song_name, singer_name, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		favorite.ID,
		favorite.UserID,
		favorite.SongID,
		favorite.SongName,
		favorite.SingerName,
		favorite.CreatedAt,
	)
	return err
}

// GetByID 根据ID获取收藏
func (r *FavoriteRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.Favorite, error) {
	query := `
		SELECT id, user_id, song_id, song_name, singer_name, deleted_at, created_at
		FROM favorites
		WHERE id = $1 AND deleted_at IS NULL
	`
	var favorite domain.Favorite
	err := r.db.QueryRow(ctx, query, id).Scan(
		&favorite.ID,
		&favorite.UserID,
		&favorite.SongID,
		&favorite.SongName,
		&favorite.SingerName,
		&favorite.DeletedAt,
		&favorite.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &favorite, nil
}

// GetByUserAndSong 根据用户和歌曲ID获取收藏
func (r *FavoriteRepositoryImpl) GetByUserAndSong(ctx context.Context, userID, songID string) (*domain.Favorite, error) {
	query := `
		SELECT id, user_id, song_id, song_name, singer_name, deleted_at, created_at
		FROM favorites
		WHERE user_id = $1 AND song_id = $2 AND deleted_at IS NULL
	`
	var favorite domain.Favorite
	err := r.db.QueryRow(ctx, query, userID, songID).Scan(
		&favorite.ID,
		&favorite.UserID,
		&favorite.SongID,
		&favorite.SongName,
		&favorite.SingerName,
		&favorite.DeletedAt,
		&favorite.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &favorite, nil
}

// ListByUser 获取用户的收藏列表
func (r *FavoriteRepositoryImpl) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Favorite, error) {
	query := `
		SELECT id, user_id, song_id, song_name, singer_name, deleted_at, created_at
		FROM favorites
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var favorites []*domain.Favorite
	for rows.Next() {
		var favorite domain.Favorite
		err := rows.Scan(
			&favorite.ID,
			&favorite.UserID,
			&favorite.SongID,
			&favorite.SongName,
			&favorite.SingerName,
			&favorite.DeletedAt,
			&favorite.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		favorites = append(favorites, &favorite)
	}
	return favorites, rows.Err()
}

// Count 统计用户的收藏数量
func (r *FavoriteRepositoryImpl) Count(ctx context.Context, userID string) (int64, error) {
	query := `SELECT COUNT(*) FROM favorites WHERE user_id = $1 AND deleted_at IS NULL`
	var count int64
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// SoftDelete 软删除收藏
func (r *FavoriteRepositoryImpl) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE favorites SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, id, time.Now())
	return err
}

// Restore 恢复收藏
func (r *FavoriteRepositoryImpl) Restore(ctx context.Context, id string) error {
	query := `UPDATE favorites SET deleted_at = NULL WHERE id = $1 AND deleted_at IS NOT NULL`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// HardDelete 硬删除收藏
func (r *FavoriteRepositoryImpl) HardDelete(ctx context.Context, id string) error {
	query := `DELETE FROM favorites WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// Exists 检查收藏是否存在
func (r *FavoriteRepositoryImpl) Exists(ctx context.Context, userID, songID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM favorites
			WHERE user_id = $1 AND song_id = $2 AND deleted_at IS NULL
		)
	`
	var exists bool
	err := r.db.QueryRow(ctx, query, userID, songID).Scan(&exists)
	return exists, err
}
