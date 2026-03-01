package repository

import (
	"context"

	"user-svc/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PlayHistoryRepositoryImpl 播放历史仓储实现
type PlayHistoryRepositoryImpl struct {
	db *pgxpool.Pool
}

// NewPlayHistoryRepository 创建播放历史仓储
func NewPlayHistoryRepository(db *pgxpool.Pool) PlayHistoryRepository {
	return &PlayHistoryRepositoryImpl{db: db}
}

// Create 创建播放历史
func (r *PlayHistoryRepositoryImpl) Create(ctx context.Context, history *domain.PlayHistory) error {
	query := `
		INSERT INTO play_histories (id, user_id, song_id, song_name, singer_name, album_cover, duration, played_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query,
		history.ID,
		history.UserID,
		history.SongID,
		history.SongName,
		history.SingerName,
		history.AlbumCover,
		history.Duration,
		history.PlayedAt,
		history.CreatedAt,
	)
	return err
}

// GetByID 根据ID获取播放历史
func (r *PlayHistoryRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.PlayHistory, error) {
	query := `
		SELECT id, user_id, song_id, song_name, singer_name, album_cover, duration, played_at, created_at
		FROM play_histories
		WHERE id = $1
	`
	var history domain.PlayHistory
	err := r.db.QueryRow(ctx, query, id).Scan(
		&history.ID,
		&history.UserID,
		&history.SongID,
		&history.SongName,
		&history.SingerName,
		&history.AlbumCover,
		&history.Duration,
		&history.PlayedAt,
		&history.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &history, nil
}

// ListByUser 获取用户的播放历史列表
func (r *PlayHistoryRepositoryImpl) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.PlayHistory, error) {
	query := `
		SELECT id, user_id, song_id, song_name, singer_name, album_cover, duration, played_at, created_at
		FROM play_histories
		WHERE user_id = $1
		ORDER BY played_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []*domain.PlayHistory
	for rows.Next() {
		var history domain.PlayHistory
		err := rows.Scan(
			&history.ID,
			&history.UserID,
			&history.SongID,
			&history.SongName,
			&history.SingerName,
			&history.AlbumCover,
			&history.Duration,
			&history.PlayedAt,
			&history.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		histories = append(histories, &history)
	}
	return histories, rows.Err()
}

// Count 统计用户的播放历史数量
func (r *PlayHistoryRepositoryImpl) Count(ctx context.Context, userID string) (int64, error) {
	query := `SELECT COUNT(*) FROM play_histories WHERE user_id = $1`
	var count int64
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// Delete 删除播放历史
func (r *PlayHistoryRepositoryImpl) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM play_histories WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// DeleteOldest 删除最早的N条记录
func (r *PlayHistoryRepositoryImpl) DeleteOldest(ctx context.Context, userID string, count int) error {
	query := `
		DELETE FROM play_histories
		WHERE id IN (
			SELECT id FROM play_histories
			WHERE user_id = $1
			ORDER BY played_at ASC
			LIMIT $2
		)
	`
	_, err := r.db.Exec(ctx, query, userID, count)
	return err
}

// Cleanup 清理用户历史记录，保留最新的keepCount条
func (r *PlayHistoryRepositoryImpl) Cleanup(ctx context.Context, userID string, keepCount int) error {
	query := `
		DELETE FROM play_histories
		WHERE user_id = $1
		AND id NOT IN (
			SELECT id FROM play_histories
			WHERE user_id = $1
			ORDER BY played_at DESC
			LIMIT $2
		)
	`
	_, err := r.db.Exec(ctx, query, userID, keepCount)
	return err
}

// GetAllUserIDs 获取所有有播放历史的用户ID列表
func (r *PlayHistoryRepositoryImpl) GetAllUserIDs(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT user_id FROM play_histories`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}
	return userIDs, rows.Err()
}
