package client

import (
	"context"
	"fmt"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
	userv1 "github.com/xiaoxiao0301/listen-stream-v2/server/shared/proto/user/v1"
	"google.golang.org/grpc"
)

// UserClient user-svc gRPC客户端
type UserClient struct {
	client  userv1.UserServiceClient
	address string
	log     logger.Logger
}

// NewUserClient 创建user客户端
func NewUserClient(conn *grpc.ClientConn, address string, log logger.Logger) *UserClient {
	return &UserClient{
		client:  userv1.NewUserServiceClient(conn),
		address: address,
		log:     log,
	}
}

// AddFavorite 添加收藏
func (c *UserClient) AddFavorite(ctx context.Context, userID, targetID string, favType userv1.FavoriteType, metadata *userv1.FavoriteMetadata) (*userv1.AddFavoriteResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &userv1.AddFavoriteRequest{
		UserId:   userID,
		Type:     favType,
		TargetId: targetID,
		Metadata: metadata,
	}

	resp, err := c.client.AddFavorite(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to add favorite via gRPC")
		return nil, fmt.Errorf("add favorite failed: %w", err)
	}

	return resp, nil
}

// RemoveFavorite 移除收藏
func (c *UserClient) RemoveFavorite(ctx context.Context, userID, favoriteID string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &userv1.RemoveFavoriteRequest{
		UserId:     userID,
		FavoriteId: favoriteID,
	}

	resp, err := c.client.RemoveFavorite(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to remove favorite via gRPC")
		return fmt.Errorf("remove favorite failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("remove favorite unsuccessful")
	}

	return nil
}

// ListFavorites 获取收藏列表
func (c *UserClient) ListFavorites(ctx context.Context, userID string, favType userv1.FavoriteType, page, size int32) (*userv1.ListFavoritesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &userv1.ListFavoritesRequest{
		UserId:   userID,
		Type:     favType,
		Page:     page,
		PageSize: size,
	}

	resp, err := c.client.ListFavorites(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to list favorites via gRPC")
		return nil, fmt.Errorf("list favorites failed: %w", err)
	}

	return resp, nil
}

// AddPlayHistory 添加播放历史
func (c *UserClient) AddPlayHistory(ctx context.Context, userID, songID, songName, artistName, albumName string, duration int32) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &userv1.AddPlayHistoryRequest{
		UserId:     userID,
		SongId:     songID,
		SongName:   songName,
		ArtistName: artistName,
		AlbumName:  albumName,
		Duration:   duration,
	}

	_, err := c.client.AddPlayHistory(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to add play history via gRPC")
		return fmt.Errorf("add play history failed: %w", err)
	}

	return nil
}

// ListPlayHistory 获取播放历史
func (c *UserClient) ListPlayHistory(ctx context.Context, userID string, page, size int32) (*userv1.ListPlayHistoryResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &userv1.ListPlayHistoryRequest{
		UserId:   userID,
		Page:     page,
		PageSize: size,
	}

	resp, err := c.client.ListPlayHistory(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to list play history via gRPC")
		return nil, fmt.Errorf("list play history failed: %w", err)
	}

	return resp, nil
}

// CreatePlaylist 创建歌单
func (c *UserClient) CreatePlaylist(ctx context.Context, userID, name, cover string, isPublic bool) (*userv1.CreatePlaylistResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &userv1.CreatePlaylistRequest{
		UserId:   userID,
		Name:     name,
		CoverUrl: cover,
		IsPublic: isPublic,
	}

	resp, err := c.client.CreatePlaylist(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to create playlist via gRPC")
		return nil, fmt.Errorf("create playlist failed: %w", err)
	}

	return resp, nil
}

// UpdatePlaylist 更新歌单
func (c *UserClient) UpdatePlaylist(ctx context.Context, userID, playlistID, name, cover string, isPublic *bool) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &userv1.UpdatePlaylistRequest{
		UserId:     userID,
		PlaylistId: playlistID,
		Name:       name,
		CoverUrl:   cover,
	}

	if isPublic != nil {
		req.IsPublic = isPublic
	}

	_, err := c.client.UpdatePlaylist(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to update playlist via gRPC")
		return fmt.Errorf("update playlist failed: %w", err)
	}

	return nil
}

// DeletePlaylist 删除歌单
func (c *UserClient) DeletePlaylist(ctx context.Context, userID, playlistID string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &userv1.DeletePlaylistRequest{
		UserId:     userID,
		PlaylistId: playlistID,
	}

	_, err := c.client.DeletePlaylist(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to delete playlist via gRPC")
		return fmt.Errorf("delete playlist failed: %w", err)
	}

	return nil
}

// ListPlaylists 获取歌单列表
func (c *UserClient) ListPlaylists(ctx context.Context, userID string) (*userv1.ListPlaylistsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &userv1.ListPlaylistsRequest{
		UserId: userID,
	}

	resp, err := c.client.ListPlaylists(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to list playlists via gRPC")
		return nil, fmt.Errorf("list playlists failed: %w", err)
	}

	return resp, nil
}

// AddSongToPlaylist 添加歌曲到歌单
func (c *UserClient) AddSongToPlaylist(ctx context.Context, userID, playlistID, songID string, position int32) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &userv1.AddSongToPlaylistRequest{
		UserId:     userID,
		PlaylistId: playlistID,
		SongId:     songID,
		Position:   position,
	}

	_, err := c.client.AddSongToPlaylist(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to add song to playlist via gRPC")
		return fmt.Errorf("add song to playlist failed: %w", err)
	}

	return nil
}

// RemoveSongFromPlaylist 从歌单移除歌曲
func (c *UserClient) RemoveSongFromPlaylist(ctx context.Context, userID, playlistID, songID string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &userv1.RemoveSongFromPlaylistRequest{
		UserId:     userID,
		PlaylistId: playlistID,
		SongId:     songID,
	}

	_, err := c.client.RemoveSongFromPlaylist(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("user_id", userID),
		).Error("Failed to remove song from playlist via gRPC")
		return fmt.Errorf("remove song from playlist failed: %w", err)
	}

	return nil
}

// GetPlaylistSongs 获取歌单歌曲列表
func (c *UserClient) GetPlaylistSongs(ctx context.Context, playlistID string) (*userv1.GetPlaylistSongsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &userv1.GetPlaylistSongsRequest{
		PlaylistId: playlistID,
	}

	resp, err := c.client.GetPlaylistSongs(ctx, req)
	if err != nil {
		c.log.WithFields(
			logger.String("error", err.Error()),
			logger.String("playlist_id", playlistID),
		).Error("Failed to get playlist songs via gRPC")
		return nil, fmt.Errorf("get playlist songs failed: %w", err)
	}

	return resp, nil
}
