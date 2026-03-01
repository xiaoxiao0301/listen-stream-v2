package service

import (
	"context"
	"time"

	"user-svc/internal/domain"
	"user-svc/internal/repository"

	"github.com/google/uuid"
)

// PlaylistService 歌单服务
type PlaylistService struct {
	playlistRepo     repository.PlaylistRepository
	playlistSongRepo repository.PlaylistSongRepository
}

// NewPlaylistService 创建歌单服务
func NewPlaylistService(playlistRepo repository.PlaylistRepository, playlistSongRepo repository.PlaylistSongRepository) *PlaylistService {
	return &PlaylistService{
		playlistRepo:     playlistRepo,
		playlistSongRepo: playlistSongRepo,
	}
}

// CreatePlaylist 创建歌单
func (s *PlaylistService) CreatePlaylist(ctx context.Context, userID, name, description, coverURL string, isPublic bool) (*domain.UserPlaylist, error) {
	if err := domain.ValidatePlaylistName(name); err != nil {
		return nil, err
	}

	playlist := &domain.UserPlaylist{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        name,
		Description: description,
		CoverURL:    coverURL,
		IsPublic:    isPublic,
		SongCount:   0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.playlistRepo.Create(ctx, playlist); err != nil {
		return nil, err
	}

	return playlist, nil
}

// GetPlaylist 获取歌单详情
func (s *PlaylistService) GetPlaylist(ctx context.Context, playlistID string) (*domain.UserPlaylist, error) {
	return s.playlistRepo.GetByID(ctx, playlistID)
}

// GetUserPlaylists 获取用户的歌单列表
func (s *PlaylistService) GetUserPlaylists(ctx context.Context, userID string, page, pageSize int) ([]*domain.UserPlaylist, int64, error) {
	offset := (page - 1) * pageSize
	playlists, err := s.playlistRepo.ListByUser(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.playlistRepo.Count(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return playlists, total, nil
}

// UpdatePlaylist 更新歌单信息
func (s *PlaylistService) UpdatePlaylist(ctx context.Context, playlistID, userID, name, description, coverURL string, isPublic bool) (*domain.UserPlaylist, error) {
	playlist, err := s.playlistRepo.GetByID(ctx, playlistID)
	if err != nil {
		return nil, err
	}

	if playlist.UserID != userID {
		return nil, domain.ErrUnauthorized
	}

	if name != "" {
		if err := domain.ValidatePlaylistName(name); err != nil {
			return nil, err
		}
		playlist.SetName(name)
	}

	if description != "" {
		playlist.Description = description
	}

	if coverURL != "" {
		playlist.CoverURL = coverURL
	}

	playlist.IsPublic = isPublic
	playlist.UpdatedAt = time.Now()

	if err := s.playlistRepo.Update(ctx, playlist); err != nil {
		return nil, err
	}

	return playlist, nil
}

// DeletePlaylist 删除歌单
func (s *PlaylistService) DeletePlaylist(ctx context.Context, playlistID, userID string) error {
	playlist, err := s.playlistRepo.GetByID(ctx, playlistID)
	if err != nil {
		return err
	}

	if playlist.UserID != userID {
		return domain.ErrUnauthorized
	}

	return s.playlistRepo.SoftDelete(ctx, playlistID)
}

// AddSongToPlaylist 添加歌曲到歌单
func (s *PlaylistService) AddSongToPlaylist(ctx context.Context, playlistID, userID, songID, songName, singerName string) error {
	playlist, err := s.playlistRepo.GetByID(ctx, playlistID)
	if err != nil {
		return err
	}

	if playlist.UserID != userID {
		return domain.ErrUnauthorized
	}

	// 检查歌曲是否已在歌单中
	exists, err := s.playlistSongRepo.Exists(ctx, playlistID, songID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrSongAlreadyInPlaylist
	}

	// 获取当前最大位置
	maxPos, err := s.playlistSongRepo.GetMaxPosition(ctx, playlistID)
	if err != nil {
		return err
	}

	playlistSong := &domain.PlaylistSong{
		PlaylistID: playlistID,
		SongID:     songID,
		SongName:   songName,
		SingerName: singerName,
		Position:   maxPos + 1,
		AddedAt:    time.Now(),
	}

	if err := s.playlistSongRepo.Add(ctx, playlistSong); err != nil {
		return err
	}

	// 增加歌单歌曲数量
	return s.playlistRepo.IncrementSongCount(ctx, playlistID)
}

// RemoveSongFromPlaylist 从歌单移除歌曲
func (s *PlaylistService) RemoveSongFromPlaylist(ctx context.Context, playlistID, userID, songID string) error {
	playlist, err := s.playlistRepo.GetByID(ctx, playlistID)
	if err != nil {
		return err
	}

	if playlist.UserID != userID {
		return domain.ErrUnauthorized
	}

	if err := s.playlistSongRepo.Remove(ctx, playlistID, songID); err != nil {
		return err
	}

	// 减少歌单歌曲数量
	return s.playlistRepo.DecrementSongCount(ctx, playlistID)
}

// GetPlaylistSongs 获取歌单的所有歌曲
func (s *PlaylistService) GetPlaylistSongs(ctx context.Context, playlistID string) ([]*domain.PlaylistSong, error) {
	return s.playlistSongRepo.List(ctx, playlistID)
}
