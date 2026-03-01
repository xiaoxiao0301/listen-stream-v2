package service

import (
	"context"
	"time"

	"user-svc/internal/domain"
	"user-svc/internal/repository"

	"github.com/google/uuid"
)

const (
	// MaxHistoryPerUser 每个用户最多保留的历史记录数
	MaxHistoryPerUser = 500
)

// PlayHistoryService 播放历史服务
type PlayHistoryService struct {
	repo repository.PlayHistoryRepository
}

// NewPlayHistoryService 创建播放历史服务
func NewPlayHistoryService(repo repository.PlayHistoryRepository) *PlayHistoryService {
	return &PlayHistoryService{
		repo: repo,
	}
}

// AddPlayHistory 添加播放历史
func (s *PlayHistoryService) AddPlayHistory(ctx context.Context, userID, songID, songName, singerName, albumCover string, duration int) (*domain.PlayHistory, error) {
	history := &domain.PlayHistory{
		ID:         uuid.New().String(),
		UserID:     userID,
		SongID:     songID,
		SongName:   songName,
		SingerName: singerName,
		AlbumCover: albumCover,
		Duration:   duration,
		PlayedAt:   time.Now(),
		CreatedAt:  time.Now(),
	}

	// 验证数据
	if err := history.Validate(); err != nil {
		return nil, err
	}

	// 保存历史记录
	if err := s.repo.Create(ctx, history); err != nil {
		return nil, err
	}

	// 检查是否超出限制，如果超出则清理旧记录
	count, err := s.repo.Count(ctx, userID)
	if err != nil {
		// 清理失败不影响主流程，只记录日志
		return history, nil
	}

	if count > MaxHistoryPerUser {
		// 异步清理，不阻塞主流程
		go func() {
			cleanCtx := context.Background()
			deleteCount := int(count - MaxHistoryPerUser)
			_ = s.repo.DeleteOldest(cleanCtx, userID, deleteCount)
		}()
	}

	return history, nil
}

// GetPlayHistory 获取播放历史详情
func (s *PlayHistoryService) GetPlayHistory(ctx context.Context, userID, historyID string) (*domain.PlayHistory, error) {
	history, err := s.repo.GetByID(ctx, historyID)
	if err != nil {
		return nil, err
	}
	if history == nil {
		return nil, domain.ErrHistoryNotFound
	}

	// 检查权限
	if history.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	return history, nil
}

// ListPlayHistories 获取用户播放历史列表
func (s *PlayHistoryService) ListPlayHistories(ctx context.Context, userID string, limit, offset int) ([]*domain.PlayHistory, int64, error) {
	// 获取列表
	histories, err := s.repo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// 获取总数
	total, err := s.repo.Count(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}

// DeletePlayHistory 删除播放历史
func (s *PlayHistoryService) DeletePlayHistory(ctx context.Context, userID, historyID string) error {
	// 获取历史记录
	history, err := s.repo.GetByID(ctx, historyID)
	if err != nil {
		return err
	}
	if history == nil {
		return domain.ErrHistoryNotFound
	}

	// 检查权限
	if history.UserID != userID {
		return domain.ErrUnauthorized
	}

	// 删除
	return s.repo.Delete(ctx, historyID)
}

// CleanupUserHistory 清理用户历史记录（保留最新的N条）
func (s *PlayHistoryService) CleanupUserHistory(ctx context.Context, userID string, keepCount int) error {
	return s.repo.Cleanup(ctx, userID, keepCount)
}

// AddHistory 添加播放历史（别名方法）
func (s *PlayHistoryService) AddHistory(ctx context.Context, userID, songID, songName, singerName, albumCover string, duration int) (*domain.PlayHistory, error) {
	return s.AddPlayHistory(ctx, userID, songID, songName, singerName, albumCover, duration)
}

// GetHistory 获取播放历史列表（别名方法）
func (s *PlayHistoryService) GetHistory(ctx context.Context, userID string, page, pageSize int) ([]*domain.PlayHistory, int64, error) {
	offset := (page - 1) * pageSize
	return s.ListPlayHistories(ctx, userID, pageSize, offset)
}

// DeleteHistory 删除播放历史（别名方法）
func (s *PlayHistoryService) DeleteHistory(ctx context.Context, historyID, userID string) error {
	return s.DeletePlayHistory(ctx, userID, historyID)
}
