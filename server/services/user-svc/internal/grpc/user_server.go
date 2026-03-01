package grpc

import (
	"context"

	"user-svc/internal/domain"
	"user-svc/internal/service"

	userv1 "github.com/listen-stream/server/shared/proto/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserServer 用户服务gRPC实现
type UserServer struct {
	userv1.UnimplementedUserServiceServer
	favoriteService *service.FavoriteService
	historyService  *service.PlayHistoryService
	playlistService *service.PlaylistService
}

// NewUserServer 创建用户服务gRPC服务器
func NewUserServer(
	favoriteService *service.FavoriteService,
	historyService *service.PlayHistoryService,
	playlistService *service.PlaylistService,
) *UserServer {
	return &UserServer{
		favoriteService: favoriteService,
		historyService:  historyService,
		playlistService: playlistService,
	}
}

// AddFavorite 添加收藏
// 注意：当前实现仅支持歌曲收藏(FAVORITE_TYPE_SONG)，未来需扩展支持album/artist/mv
func (s *UserServer) AddFavorite(ctx context.Context, req *userv1.AddFavoriteRequest) (*userv1.AddFavoriteResponse, error) {
	// 验证类型（当前仅支持歌曲）
	if req.Type != userv1.FavoriteType_FAVORITE_TYPE_SONG {
		return nil, status.Errorf(codes.Unimplemented, "only song favorites are currently supported")
	}

	// 提取元数据
	songName := ""
	singerName := ""
	if req.Metadata != nil {
		songName = req.Metadata.Name
		singerName = req.Metadata.Artist
	}

	// 调用服务层
	favorite, err := s.favoriteService.AddFavorite(ctx, req.UserId, req.TargetId, songName, singerName)
	if err != nil {
		if err == domain.ErrFavoriteAlreadyExists {
			return nil, status.Errorf(codes.AlreadyExists, "already favorited")
		}
		return nil, status.Errorf(codes.Internal, "failed to add favorite: %v", err)
	}

	return &userv1.AddFavoriteResponse{
		FavoriteId: favorite.ID,
		CreatedAt:  timestamppb.New(favorite.CreatedAt),
	}, nil
}

// RemoveFavorite 移除收藏
func (s *UserServer) RemoveFavorite(ctx context.Context, req *userv1.RemoveFavoriteRequest) (*userv1.RemoveFavoriteResponse, error) {
	err := s.favoriteService.RemoveFavorite(ctx, req.UserId, req.FavoriteId)
	if err != nil {
		if err == domain.ErrFavoriteNotFound {
			return nil, status.Errorf(codes.NotFound, "favorite not found")
		}
		if err == domain.ErrUnauthorized {
			return nil, status.Errorf(codes.PermissionDenied, "not authorized")
		}
		return nil, status.Errorf(codes.Internal, "failed to remove favorite: %v", err)
	}

	return &userv1.RemoveFavoriteResponse{
		Success: true,
	}, nil
}

// ListFavorites 获取收藏列表
func (s *UserServer) ListFavorites(ctx context.Context, req *userv1.ListFavoritesRequest) (*userv1.ListFavoritesResponse, error) {
	// 验证类型过滤（当前仅支持SONG或不过滤）
	if req.Type != userv1.FavoriteType_FAVORITE_TYPE_UNSPECIFIED && req.Type != userv1.FavoriteType_FAVORITE_TYPE_SONG {
		return &userv1.ListFavoritesResponse{
			Favorites: []*userv1.Favorite{},
			Total:     0,
			Page:      req.Page,
			PageSize:  req.PageSize,
		}, nil
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	favorites, total, err := s.favoriteService.GetFavorites(ctx, req.UserId, int(page), int(pageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list favorites: %v", err)
	}

	// 转换为proto消息
	pbFavorites := make([]*userv1.Favorite, 0, len(favorites))
	for _, f := range favorites {
		pbFavorites = append(pbFavorites, &userv1.Favorite{
			Id:       f.ID,
			UserId:   f.UserID,
			Type:     userv1.FavoriteType_FAVORITE_TYPE_SONG,
			TargetId: f.SongID,
			Metadata: &userv1.FavoriteMetadata{
				Name:   f.SongName,
				Artist: f.SingerName,
			},
			CreatedAt: timestamppb.New(f.CreatedAt),
		})
	}

	return &userv1.ListFavoritesResponse{
		Favorites: pbFavorites,
		Total:     int32(total),
		Page:      page,
		PageSize:  pageSize,
	}, nil
}

// AddPlayHistory 添加播放历史
func (s *UserServer) AddPlayHistory(ctx context.Context, req *userv1.AddPlayHistoryRequest) (*userv1.AddPlayHistoryResponse, error) {
	// 调用服务层（注意：服务层方法签名与proto稍有不同）
	history, err := s.historyService.AddHistory(
		ctx,
		req.UserId,
		req.SongId,
		req.SongName,
		req.ArtistName,
		"", // albumCover - proto中没有此字段，使用空字符串
		int(req.Duration),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add play history: %v", err)
	}

	return &userv1.AddPlayHistoryResponse{
		HistoryId:        history.ID,
		PlayedAt:         timestamppb.New(history.PlayedAt),
		EvictedOldHistory: false, // TODO: 实现清理检测
	}, nil
}

// ListPlayHistory 获取播放历史列表
func (s *UserServer) ListPlayHistory(ctx context.Context, req *userv1.ListPlayHistoryRequest) (*userv1.ListPlayHistoryResponse, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	histories, total, err := s.historyService.GetHistory(ctx, req.UserId, int(page), int(pageSize))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list play history: %v", err)
	}

	// 转换为proto消息
	pbHistories := make([]*userv1.PlayHistory, 0, len(histories))
	for _, h := range histories {
		pbHistories = append(pbHistories, &userv1.PlayHistory{
			Id:         h.ID,
			UserId:     h.UserID,
			SongId:     h.SongID,
			SongName:   h.SongName,
			ArtistName: h.SingerName,
			AlbumName:  "",           // domain层没有album_name字段
			Duration:   int32(h.Duration),
			PlayedAt:   timestamppb.New(h.PlayedAt),
		})
	}

	return &userv1.ListPlayHistoryResponse{
		History:  pbHistories,
		Total:    int32(total),
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// CreatePlaylist 创建歌单
func (s *UserServer) CreatePlaylist(ctx context.Context, req *userv1.CreatePlaylistRequest) (*userv1.CreatePlaylistResponse, error) {
	playlist, err := s.playlistService.CreatePlaylist(
		ctx,
		req.UserId,
		req.Name,
		"",          // description - proto中没有此字段
		req.CoverUrl,
		req.IsPublic,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create playlist: %v", err)
	}

	return &userv1.CreatePlaylistResponse{
		Playlist: domainPlaylistToProto(playlist),
	}, nil
}

// UpdatePlaylist 更新歌单
func (s *UserServer) UpdatePlaylist(ctx context.Context, req *userv1.UpdatePlaylistRequest) (*userv1.UpdatePlaylistResponse, error) {
	isPublic := false
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	playlist, err := s.playlistService.UpdatePlaylist(
		ctx,
		req.PlaylistId,
		req.UserId,
		req.Name,
		"",          // description
		req.CoverUrl,
		isPublic,
	)
	if err != nil {
		if err == domain.ErrUnauthorized {
			return nil, status.Errorf(codes.PermissionDenied, "not authorized")
		}
		return nil, status.Errorf(codes.Internal, "failed to update playlist: %v", err)
	}

	return &userv1.UpdatePlaylistResponse{
		Playlist: domainPlaylistToProto(playlist),
	}, nil
}

// DeletePlaylist 删除歌单
func (s *UserServer) DeletePlaylist(ctx context.Context, req *userv1.DeletePlaylistRequest) (*userv1.DeletePlaylistResponse, error) {
	err := s.playlistService.DeletePlaylist(ctx, req.PlaylistId, req.UserId)
	if err != nil {
		if err == domain.ErrUnauthorized {
			return nil, status.Errorf(codes.PermissionDenied, "not authorized")
		}
		return nil, status.Errorf(codes.Internal, "failed to delete playlist: %v", err)
	}

	return &userv1.DeletePlaylistResponse{
		Success: true,
	}, nil
}

// ListPlaylists 获取用户歌单列表
func (s *UserServer) ListPlaylists(ctx context.Context, req *userv1.ListPlaylistsRequest) (*userv1.ListPlaylistsResponse, error) {
	// 获取所有歌单（不分页）
	playlists, _, err := s.playlistService.GetUserPlaylists(ctx, req.UserId, 1, 1000)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list playlists: %v", err)
	}

	// 转换为proto消息
	pbPlaylists := make([]*userv1.Playlist, 0, len(playlists))
	for _, p := range playlists {
		pbPlaylists = append(pbPlaylists, domainPlaylistToProto(p))
	}

	return &userv1.ListPlaylistsResponse{
		Playlists: pbPlaylists,
	}, nil
}

// AddSongToPlaylist 添加歌曲到歌单
func (s *UserServer) AddSongToPlaylist(ctx context.Context, req *userv1.AddSongToPlaylistRequest) (*userv1.AddSongToPlaylistResponse, error) {
	// 注意：proto要求传入song_name和singer_name，但当前缺失，使用空字符串
	err := s.playlistService.AddSongToPlaylist(
		ctx,
		req.PlaylistId,
		req.UserId,
		req.SongId,
		"",  // songName - proto中没有此字段
		"",  // singerName - proto中没有此字段
	)
	if err != nil {
		if err == domain.ErrUnauthorized {
			return nil, status.Errorf(codes.PermissionDenied, "not authorized")
		}
		if err == domain.ErrSongAlreadyInPlaylist {
			return nil, status.Errorf(codes.AlreadyExists, "song already in playlist")
		}
		return nil, status.Errorf(codes.Internal, "failed to add song to playlist: %v", err)
	}

	// 获取更新后的歌单以返回song_count
	playlist, err := s.playlistService.GetPlaylist(ctx, req.PlaylistId)
	if err != nil {
		return &userv1.AddSongToPlaylistResponse{
			Success:   true,
			SongCount: 0,
		}, nil
	}

	return &userv1.AddSongToPlaylistResponse{
		Success:   true,
		SongCount: int32(playlist.SongCount),
	}, nil
}

// RemoveSongFromPlaylist 从歌单移除歌曲
func (s *UserServer) RemoveSongFromPlaylist(ctx context.Context, req *userv1.RemoveSongFromPlaylistRequest) (*userv1.RemoveSongFromPlaylistResponse, error) {
	err := s.playlistService.RemoveSongFromPlaylist(ctx, req.PlaylistId, req.UserId, req.SongId)
	if err != nil {
		if err == domain.ErrUnauthorized {
			return nil, status.Errorf(codes.PermissionDenied, "not authorized")
		}
		return nil, status.Errorf(codes.Internal, "failed to remove song from playlist: %v", err)
	}

	// 获取更新后的歌单以返回song_count
	playlist, err := s.playlistService.GetPlaylist(ctx, req.PlaylistId)
	if err != nil {
		return &userv1.RemoveSongFromPlaylistResponse{
			Success:   true,
			SongCount: 0,
		}, nil
	}

	return &userv1.RemoveSongFromPlaylistResponse{
		Success:   true,
		SongCount: int32(playlist.SongCount),
	}, nil
}

// GetPlaylistSongs 获取歌单歌曲列表
func (s *UserServer) GetPlaylistSongs(ctx context.Context, req *userv1.GetPlaylistSongsRequest) (*userv1.GetPlaylistSongsResponse, error) {
	songs, err := s.playlistService.GetPlaylistSongs(ctx, req.PlaylistId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get playlist songs: %v", err)
	}

	// 转换为proto消息
	pbSongs := make([]*userv1.PlaylistSong, 0, len(songs))
	for _, s := range songs {
		pbSongs = append(pbSongs, &userv1.PlaylistSong{
			PlaylistId: s.PlaylistID,
			SongId:     s.SongID,
			Position:   int32(s.Position),
			AddedAt:    timestamppb.New(s.AddedAt),
		})
	}

	return &userv1.GetPlaylistSongsResponse{
		Songs: pbSongs,
	}, nil
}

// domainPlaylistToProto 将domain歌单转换为proto消息
func domainPlaylistToProto(p *domain.UserPlaylist) *userv1.Playlist {
	return &userv1.Playlist{
		Id:        p.ID,
		UserId:    p.UserID,
		Name:      p.Name,
		CoverUrl:  p.CoverURL,
		IsPublic:  p.IsPublic,
		SongCount: int32(p.SongCount),
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}
}
