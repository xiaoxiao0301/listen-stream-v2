package upstream

import (
	"context"
	"fmt"
	"sync"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// FallbackManager 管理多个上游源的降级链
type FallbackManager struct {
	clients []ClientInterface
	names   []string
	logger  logger.Logger
	mu      sync.RWMutex
}

// NewFallbackManager 创建Fallback管理器
func NewFallbackManager(clients []ClientInterface, names []string, log logger.Logger) *FallbackManager {
	if len(clients) != len(names) {
		panic("clients and names length mismatch")
	}

	return &FallbackManager{
		clients: clients,
		names:   names,
		logger:  log,
	}
}

// GetSongURL 获取歌曲播放URL（实现 ClientInterface）
func (fm *FallbackManager) GetSongURL(ctx context.Context, songMid, songName string) (*SongURL, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}

	// 尝试从第一个客户端获取（通常是 QQ Music）
	return fm.clients[0].GetSongURL(ctx, songMid, songName)
}

/* Deprecated: GetSongURLWithFallback is no longer used
// GetSongURLWithFallback 获取歌曲播放URL，支持智能Fallback
// 流程：QQ Music → Joox → NetEase → Kugou
// 这是一个更高级的方法，handler 可以使用这个方法来实现fallback
func (fm *FallbackManager) GetSongURLWithFallback(ctx context.Context, songID, songName, artistName, quality string) (*SongURLResponse, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	var lastErr error

	// 尝试使用songID直接获取
	if songID != "" {
		for i, client := range fm.clients {
			sourceName := fm.names[i]
			
			fm.logger.Debug("Trying to get song URL from source",
				logger.String("source", sourceName),
				logger.String("song_id", songID),
			)

			urlResp, err := client.GetSongURL(ctx, songID, quality)
			if err == nil && urlResp != nil && urlResp.URL != "" {
				fm.logger.Info("Successfully got song URL",
					logger.String("source", sourceName),
					logger.String("song_id", songID),
				)
				return urlResp, nil
			}

			lastErr = err
			fm.logger.Warn("Failed to get song URL from source",
				logger.String("source", sourceName),
				logger.String("song_id", songID),
				logger.String("error", err.Error()),
			)
		}
	}

	// 如果直接获取失败，尝试搜索匹配
	if songName != "" {
		fm.logger.Info("Trying to match song by name",
			logger.String("song_name", songName),
			logger.String("artist_name", artistName),
		)

		urlResp, err := fm.searchAndMatch(ctx, songName, artistName, quality)
		if err == nil && urlResp != nil {
			return urlResp, nil
		}

		if lastErr == nil {
			lastErr = err
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all sources failed, last error: %w", lastErr)
	}

	return nil, ErrSongNotFound
}

// searchAndMatch 搜索并匹配歌曲
func (fm *FallbackManager) searchAndMatch(ctx context.Context, songName, artistName, quality string) (*SongURLResponse, error) {
	for i, client := range fm.clients {
		sourceName := fm.names[i]

		fm.logger.Debug("Searching song in source",
			logger.String("source", sourceName),
			logger.String("song_name", songName),
			logger.String("artist_name", artistName),
		)

		// 搜索歌曲
		searchResult, err := client.SearchSong(ctx, songName, artistName)
		if err != nil {
			fm.logger.Warn("Search failed in source",
				logger.String("source", sourceName),
				logger.String("error", err.Error()),
			)
			continue
		}

		if searchResult == nil || len(searchResult.Songs) == 0 {
			fm.logger.Debug("No results found in source",
				logger.String("source", sourceName),
			)
			continue
		}

		// 智能匹配：优先匹配歌名+歌手名
		matchedSong := fm.findBestMatch(searchResult.Songs, songName, artistName)
		if matchedSong == nil {
			fm.logger.Debug("No good match found in source",
				logger.String("source", sourceName),
			)
			continue
		}

		// 获取匹配歌曲的ID
		songID := fm.extractSongID(matchedSong)
		if songID == "" {
			fm.logger.Warn("Cannot extract song ID from matched song",
				logger.String("source", sourceName),
			)
			continue
		}

		fm.logger.Info("Found matched song",
			logger.String("source", sourceName),
			logger.String("song_id", songID),
			logger.String("song_name", getSongName(matchedSong)),
			logger.String("artist", getArtistName(matchedSong)),
		)

		// 获取播放URL
		urlResp, err := client.GetSongURL(ctx, songID, quality)
		if err != nil {
			fm.logger.Warn("Failed to get URL for matched song",
				logger.String("source", sourceName),
				logger.String("song_id", songID),
				logger.String("error", err.Error()),
			)
			continue
		}

		if urlResp != nil && urlResp.URL != "" {
			fm.logger.Info("Successfully got song URL via search and match",
				logger.String("source", sourceName),
				logger.String("song_id", songID),
			)
			return urlResp, nil
		}
	}

	return nil, ErrSongNotFound
}

// findBestMatch 找到最佳匹配的歌曲
// 优先级：歌名+歌手名完全匹配 > 歌名完全匹配 > 歌名模糊匹配
func (fm *FallbackManager) findBestMatch(songs []Song, targetName, targetArtist string) *Song {
	if len(songs) == 0 {
		return nil
	}

	targetNameLower := strings.ToLower(strings.TrimSpace(targetName))
	targetArtistLower := strings.ToLower(strings.TrimSpace(targetArtist))

	var exactMatch *Song
	var nameMatch *Song
	var fuzzyMatch *Song

	for i := range songs {
		song := &songs[i]
		songName := strings.ToLower(strings.TrimSpace(getSongName(song)))
		artistName := strings.ToLower(strings.TrimSpace(getArtistName(song)))

		// 检查歌名+歌手名完全匹配
		if targetArtistLower != "" && songName == targetNameLower && strings.Contains(artistName, targetArtistLower) {
			exactMatch = song
			break
		}

		// 检查歌名完全匹配
		if songName == targetNameLower && nameMatch == nil {
			nameMatch = song
		}

		// 检查歌名模糊匹配
		if strings.Contains(songName, targetNameLower) && fuzzyMatch == nil {
			fuzzyMatch = song
		}
	}

	// 返回最佳匹配
	if exactMatch != nil {
		return exactMatch
	}
	if nameMatch != nil {
		return nameMatch
	}
	if fuzzyMatch != nil {
		return fuzzyMatch
	}

	// 如果没有找到合适的匹配，返回第一个结果
	return &songs[0]
}

// extractSongID 从Song结构中提取ID
func (fm *FallbackManager) extractSongID(song *Song) string {
	// 优先使用standardized ID
	if song.ID != "" {
		return song.ID
	}
	// Fallback to QQ Music style IDs
	if song.SongMid != "" {
		return song.SongMid
	}
	if song.SongID != "" {
		return song.SongID
	}
	return ""
}
*/

// HealthCheckAll 检查所有源的健康状态
func (fm *FallbackManager) HealthCheckAll(ctx context.Context) map[string]error {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	results := make(map[string]error)

	for i, client := range fm.clients {
		sourceName := fm.names[i]
		err := client.HealthCheck(ctx)
		results[sourceName] = err

		if err != nil {
			fm.logger.Warn("Health check failed",
				logger.String("source", sourceName),
				logger.String("error", err.Error()),
			)
		} else {
			fm.logger.Debug("Health check passed",
				logger.String("source", sourceName),
			)
		}
	}

	return results
}

// GetHealthySourceCount 获取健康源的数量
func (fm *FallbackManager) GetHealthySourceCount(ctx context.Context) int {
	results := fm.HealthCheckAll(ctx)
	count := 0
	for _, err := range results {
		if err == nil {
			count++
		}
	}
	return count
}

// Helper functions to handle different Song struct field naming

func getSongName(song *Song) string {
	if song.Name != "" {
		return song.Name
	}
	return song.SongName
}

func getArtistName(song *Song) string {
	if song.Artist != "" {
		return song.Artist
	}
	return song.SingerName
}

// Delegate methods to the primary client (first in the list)
// These methods don't need fallback logic

func (fm *FallbackManager) SearchSong(ctx context.Context, songName, artistName string) (*SearchResult, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].SearchSong(ctx, songName, artistName)
}

func (fm *FallbackManager) GetBanners(ctx context.Context) ([]Banner, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetBanners(ctx)
}

func (fm *FallbackManager) HealthCheck(ctx context.Context) error {
	if len(fm.clients) == 0 {
		return fmt.Errorf("no clients available")
	}
	return fm.clients[0].HealthCheck(ctx)
}

func (fm *FallbackManager) GetDailyRecommendPlaylists(ctx context.Context) ([]Playlist, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetDailyRecommendPlaylists(ctx)
}

func (fm *FallbackManager) GetRecommendPlaylists(ctx context.Context) ([]Playlist, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetRecommendPlaylists(ctx)
}

func (fm *FallbackManager) GetNewSongs(ctx context.Context, typ int) ([]Song, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetNewSongs(ctx, typ)
}

func (fm *FallbackManager) GetNewAlbums(ctx context.Context, typ int) ([]Album, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetNewAlbums(ctx, typ)
}

func (fm *FallbackManager) GetPlaylistCategories(ctx context.Context) ([]PlaylistCategory, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetPlaylistCategories(ctx)
}

func (fm *FallbackManager) GetPlaylistsByCategory(ctx context.Context, req GetPlaylistsByCategoryRequest) (*PageResponse, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetPlaylistsByCategory(ctx, req)
}

func (fm *FallbackManager) GetPlaylistDetail(ctx context.Context, dissID string) (*PlaylistDetail, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetPlaylistDetail(ctx, dissID)
}

func (fm *FallbackManager) GetSongDetail(ctx context.Context, songMid string) (*SongDetail, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetSongDetail(ctx, songMid)
}

func (fm *FallbackManager) GetLyric(ctx context.Context, songMid string) (*Lyric, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetLyric(ctx, songMid)
}

func (fm *FallbackManager) GetAlbumDetail(ctx context.Context, albumMid string) (*Album, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetAlbumDetail(ctx, albumMid)
}

func (fm *FallbackManager) GetAlbumSongs(ctx context.Context, albumMid string) ([]Song, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetAlbumSongs(ctx, albumMid)
}

func (fm *FallbackManager) GetHotKeys(ctx context.Context) ([]HotKey, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetHotKeys(ctx)
}

func (fm *FallbackManager) SearchSongs(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].SearchSongs(ctx, keyword, page, size)
}

func (fm *FallbackManager) SearchSingers(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].SearchSingers(ctx, keyword, page, size)
}

func (fm *FallbackManager) SearchAlbums(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].SearchAlbums(ctx, keyword, page, size)
}

func (fm *FallbackManager) SearchMVs(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].SearchMVs(ctx, keyword, page, size)
}

func (fm *FallbackManager) GetSingerCategories(ctx context.Context) ([]CategoryFilter, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetSingerCategories(ctx)
}

func (fm *FallbackManager) GetSingerList(ctx context.Context, req GetSingerListRequest) (*PageResponse, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetSingerList(ctx, req)
}

func (fm *FallbackManager) GetSingerDetail(ctx context.Context, singerMid string, page int) (*PageResponse, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetSingerDetail(ctx, singerMid, page)
}

func (fm *FallbackManager) GetSingerAlbums(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetSingerAlbums(ctx, singerMid, page, size)
}

func (fm *FallbackManager) GetSingerMVs(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetSingerMVs(ctx, singerMid, page, size)
}

func (fm *FallbackManager) GetSingerSongs(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetSingerSongs(ctx, singerMid, page, size)
}

func (fm *FallbackManager) GetRankingList(ctx context.Context) ([]RankingCategory, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetRankingList(ctx)
}

func (fm *FallbackManager) GetRankingDetail(ctx context.Context, topID, page, size int, period string) (*PageResponse, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetRankingDetail(ctx, topID, page, size, period)
}

func (fm *FallbackManager) GetRadioCategories(ctx context.Context) ([]Radio, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetRadioCategories(ctx)
}

func (fm *FallbackManager) GetRadioSongs(ctx context.Context, radioID int) ([]Song, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetRadioSongs(ctx, radioID)
}

func (fm *FallbackManager) GetMVCategories(ctx context.Context) (*MVCategory, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetMVCategories(ctx)
}

func (fm *FallbackManager) GetMVList(ctx context.Context, area, version, page, size int) (*PageResponse, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetMVList(ctx, area, version, page, size)
}

func (fm *FallbackManager) GetMVDetail(ctx context.Context, vid string) (*MVDetail, error) {
	if len(fm.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}
	return fm.clients[0].GetMVDetail(ctx, vid)
}
