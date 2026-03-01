package repository

import (
	"context"

	"user-svc/internal/domain"
)

// FavoriteRepository 收藏仓储接口
type FavoriteRepository interface {
	Create(ctx context.Context, favorite *domain.Favorite) error
	GetByID(ctx context.Context, id string) (*domain.Favorite, error)
	GetByUserAndSong(ctx context.Context, userID, songID string) (*domain.Favorite, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Favorite, error)
	Count(ctx context.Context, userID string) (int64, error)
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	HardDelete(ctx context.Context, id string) error
	Exists(ctx context.Context, userID, songID string) (bool, error)
}

// PlayHistoryRepository 播放历史仓储接口
type PlayHistoryRepository interface {
	Create(ctx context.Context, history *domain.PlayHistory) error
	GetByID(ctx context.Context, id string) (*domain.PlayHistory, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.PlayHistory, error)
	Count(ctx context.Context, userID string) (int64, error)
	Delete(ctx context.Context, id string) error
	DeleteOldest(ctx context.Context, userID string, count int) error
	Cleanup(ctx context.Context, userID string, keepCount int) error
	GetAllUserIDs(ctx context.Context) ([]string, error)
}

// PlaylistRepository 歌单仓储接口
type PlaylistRepository interface {
	Create(ctx context.Context, playlist *domain.UserPlaylist) error
	GetByID(ctx context.Context, id string) (*domain.UserPlaylist, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.UserPlaylist, error)
	ListPublic(ctx context.Context, limit, offset int) ([]*domain.UserPlaylist, error)
	Count(ctx context.Context, userID string) (int64, error)
	Update(ctx context.Context, playlist *domain.UserPlaylist) error
	IncrementSongCount(ctx context.Context, playlistID string) error
	DecrementSongCount(ctx context.Context, playlistID string) error
	SoftDelete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	HardDelete(ctx context.Context, id string) error
}

// PlaylistSongRepository 歌单歌曲仓储接口
type PlaylistSongRepository interface {
	Add(ctx context.Context, ps *domain.PlaylistSong) error
	Get(ctx context.Context, playlistID, songID string) (*domain.PlaylistSong, error)
	List(ctx context.Context, playlistID string) ([]*domain.PlaylistSong, error)
	Count(ctx context.Context, playlistID string) (int64, error)
	Remove(ctx context.Context, playlistID, songID string) error
	UpdatePosition(ctx context.Context, playlistID, songID string, position int) error
	Exists(ctx context.Context, playlistID, songID string) (bool, error)
	GetMaxPosition(ctx context.Context, playlistID string) (int, error)
	DeleteAll(ctx context.Context, playlistID string) error
}
