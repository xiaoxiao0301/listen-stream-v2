package service

import (
	"context"
	"time"

	"user-svc/internal/domain"
	"user-svc/internal/repository"

	"github.com/google/uuid"
)

// FavoriteService 收藏服务
type FavoriteService struct {
	repo repository.FavoriteRepository
}

// NewFavoriteService 创建收藏服务
func NewFavoriteService(repo repository.FavoriteRepository) *FavoriteService {
	return &FavoriteService{
		repo: repo,
	}
}

// AddFavorite 添加收藏
func (s *FavoriteService) AddFavorite(ctx context.Context, userID, songID, songName, singerName string) (*domain.Favorite, error) {
	// 检查是否已存在
	exists, err := s.repo.Exists(ctx, userID, songID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrFavoriteAlreadyExists
	}

	// 创建收藏
	favorite := &domain.Favorite{
		ID:         uuid.New().String(),
		UserID:     userID,
		SongID:     songID,
		SongName:   songName,
		SingerName: singerName,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.Create(ctx, favorite); err != nil {
		return nil, err
	}

	return favorite, nil
}

// RemoveFavorite 移除收藏（软删除）
func (s *FavoriteService) RemoveFavorite(ctx context.Context, userID, favoriteID string) error {
	// 获取收藏
	favorite, err := s.repo.GetByID(ctx, favoriteID)
	if err != nil {
		return err
	}
	if favorite == nil {
		return domain.ErrFavoriteNotFound
	}

	// 检查权限
	if favorite.UserID != userID {
		return domain.ErrUnauthorized
	}

	// 软删除
	return s.repo.SoftDelete(ctx, favoriteID)
}

// GetFavorite 获取收藏详情
func (s *FavoriteService) GetFavorite(ctx context.Context, userID, favoriteID string) (*domain.Favorite, error) {
	favorite, err := s.repo.GetByID(ctx, favoriteID)
	if err != nil {
		return nil, err
	}
	if favorite == nil {
		return nil, domain.ErrFavoriteNotFound
	}

	// 检查权限
	if favorite.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	return favorite, nil
}

// ListFavorites 获取用户收藏列表
func (s *FavoriteService) ListFavorites(ctx context.Context, userID string, limit, offset int) ([]*domain.Favorite, int64, error) {
	// 获取列表
	favorites, err := s.repo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// 获取总数
	total, err := s.repo.Count(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return favorites, total, nil
}

// CheckFavorite 检查歌曲是否已收藏
func (s *FavoriteService) CheckFavorite(ctx context.Context, userID, songID string) (bool, error) {
	return s.repo.Exists(ctx, userID, songID)
}

// GetFavorites 获取用户收藏列表（别名方法）
func (s *FavoriteService) GetFavorites(ctx context.Context, userID string, page, pageSize int) ([]*domain.Favorite, int64, error) {
	offset := (page - 1) * pageSize
	return s.ListFavorites(ctx, userID, pageSize, offset)
}

// IsFavorite 检查歌曲是否已收藏（别名方法）
func (s *FavoriteService) IsFavorite(ctx context.Context, userID, songID string) (bool, error) {
	return s.CheckFavorite(ctx, userID, songID)
}
