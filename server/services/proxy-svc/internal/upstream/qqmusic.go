package upstream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// QQMusicClient QQ音乐API客户端
type QQMusicClient struct {
	*Client
	cookie      string
	fallbackURL string
}

// NewQQMusicClient 创建QQ音乐客户端
func NewQQMusicClient(config UpstreamConfig, log logger.Logger) *QQMusicClient {
	clientConfig := ClientConfig{
		Name:            config.Name,
		BaseURL:         config.BaseURL,
		Timeout:         config.Timeout,
		MaxRetries:      config.MaxRetries,
		RateLimit:       config.RateLimit,
		BreakerSettings: config.BreakerSettings,
	}

	return &QQMusicClient{
		Client:      NewClient(clientConfig, log),
		cookie:      config.Cookie,
		fallbackURL: config.FallbackURL,
	}
}

// buildHeaders 构建请求头
func (c *QQMusicClient) buildHeaders() map[string]string {
	headers := make(map[string]string)
	if c.cookie != "" {
		headers["Cookie"] = c.cookie
	}
	return headers
}

// ===== Home模块 =====

// GetBanners 获取轮播图
func (c *QQMusicClient) GetBanners(ctx context.Context) ([]Banner, error) {
	data, err := c.Get(ctx, "/recommend/banner", nil)
	if err != nil {
		return nil, fmt.Errorf("get banners failed: %w", err)
	}

	var response struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Data    []Banner `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return response.Data, nil
}

// GetDailyRecommendPlaylists 获取每日推荐歌单 (需要Cookie)
func (c *QQMusicClient) GetDailyRecommendPlaylists(ctx context.Context) ([]Playlist, error) {
	headers := c.buildHeaders()
	data, err := c.Get(ctx, "/recommend/daily", headers)
	if err != nil {
		return nil, fmt.Errorf("get daily recommend playlists failed: %w", err)
	}

	var response struct {
		Code    int        `json:"code"`
		Message string     `json:"message"`
		Data    []Playlist `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return response.Data, nil
}

// GetRecommendPlaylists 获取推荐歌单
func (c *QQMusicClient) GetRecommendPlaylists(ctx context.Context) ([]Playlist, error) {
	data, err := c.Get(ctx, "/recommend/playlist", nil)
	if err != nil {
		return nil, fmt.Errorf("get recommend playlists failed: %w", err)
	}

	var response struct {
		Code    int        `json:"code"`
		Message string     `json:"message"`
		Data    []Playlist `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return response.Data, nil
}

// GetNewSongs 获取新歌推荐
// typ: 1-内地, 2-欧美, 3-日本, 4-韩国, 5-最新, 6-港台
func (c *QQMusicClient) GetNewSongs(ctx context.Context, typ int) ([]Song, error) {
	path := fmt.Sprintf("/recommend/new/songs?type=%d", typ)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get new songs failed: %w", err)
	}

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    []Song `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return response.Data, nil
}

// GetNewAlbums 获取新专辑推荐
// typ: 1-内地, 2-港台, 3-欧美, 4-韩国, 5-日本, 6-其他
func (c *QQMusicClient) GetNewAlbums(ctx context.Context, typ int) ([]Album, error) {
	path := fmt.Sprintf("/recommend/new/albums?type=%d", typ)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get new albums failed: %w", err)
	}

	var response struct {
		Code    int     `json:"code"`
		Message string  `json:"message"`
		Data    []Album `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return response.Data, nil
}

// ===== Playlist模块 =====

// GetPlaylistCategories 获取歌单分类
func (c *QQMusicClient) GetPlaylistCategories(ctx context.Context) ([]PlaylistCategory, error) {
	data, err := c.Get(ctx, "/playlist/category", nil)
	if err != nil {
		return nil, fmt.Errorf("get playlist categories failed: %w", err)
	}

	var response struct {
		Code    int                `json:"code"`
		Message string             `json:"message"`
		Data    []PlaylistCategory `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return response.Data, nil
}

// GetPlaylistsByCategoryRequest 按分类获取歌单请求参数
type GetPlaylistsByCategoryRequest struct {
	Number int // 页码
	Size   int // 每页数量
	Sort   int // 排序: 2-最新, 5-推荐
	ID     int // 分类ID
}

// GetPlaylistsByCategory 按分类获取歌单列表
func (c *QQMusicClient) GetPlaylistsByCategory(ctx context.Context, req GetPlaylistsByCategoryRequest) (*PageResponse, error) {
	path := fmt.Sprintf("/playlist/information?number=%d&size=%d&sort=%d&id=%d",
		req.Number, req.Size, req.Sort, req.ID)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get playlists by category failed: %w", err)
	}

	var response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    PageResponse `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// GetSongURL 获取歌曲播放URL (带Fallback)
func (c *QQMusicClient) GetSongURL(ctx context.Context, songMid, songName string) (*SongURL, error) {
	path := fmt.Sprintf("/song/url?id=%s", songMid)
	headers := c.buildHeaders()

	data, err := c.Get(ctx, path, headers)
	if err != nil {
		return nil, fmt.Errorf("get song url failed: %w", err)
	}

	var response struct {
		Code    int     `json:"code"`
		Message string  `json:"message"`
		Data    SongURL `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	// Code=1表示成功获取到URL
	if response.Code == 1 {
		return &response.Data, nil
	}

	// Code=0表示需要Fallback (VIP或地区限制)
	if response.Code == 0 {
		c.logger.Info("Song not available in QQ Music, trying fallback",
			logger.String("songMid", songMid),
			logger.String("songName", songName),
			logger.String("reason", response.Message),
		)

		// 使用Fallback机制
		return c.getSongURLWithFallback(ctx, songName)
	}

	return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
}

// getSongURLWithFallback 使用Fallback获取歌曲URL
func (c *QQMusicClient) getSongURLWithFallback(ctx context.Context, songName string) (*SongURL, error) {
	if c.fallbackURL == "" {
		return nil, ErrSongNotFound
	}

	// 创建Fallback客户端
	fallback := NewFallbackClient(c.fallbackURL, c.logger)

	// 步骤1: 搜索歌曲（尝试多个源：joox, netease, kugou）
	sources := []string{"joox", "netease", "kugou"}

	for _, source := range sources {
		c.logger.Debug("Trying fallback source",
			logger.String("source", source),
			logger.String("songName", songName),
		)

		songInfo, err := fallback.SearchSong(ctx, source, songName)
		if err != nil {
			c.logger.Warn("Fallback search failed",
				logger.String("source", source),
				logger.Error(err),
			)
			continue
		}

		// 步骤2: 获取歌曲URL
		songURL, err := fallback.GetSongURL(ctx, source, songInfo.ID)
		if err != nil {
			c.logger.Warn("Fallback get URL failed",
				logger.String("source", source),
				logger.String("id", songInfo.ID),
				logger.Error(err),
			)
			continue
		}

		// 检查URL是否有效
		if songURL.URL != "" {
			c.logger.Info("Successfully got song URL from fallback",
				logger.String("source", source),
				logger.String("songName", songName),
			)

			return &SongURL{
				URL:    songURL.URL,
				Source: source,
			}, nil
		}
	}

	// 所有源都失败
	c.logger.Error("All fallback sources failed",
		logger.String("songName", songName),
		logger.Int("sourcesCount", len(sources)),
	)

	return nil, ErrSongNotFound
}

// ===== Search模块 =====

// GetHotKeys 获取热搜关键词
func (c *QQMusicClient) GetHotKeys(ctx context.Context) ([]HotKey, error) {
	data, err := c.Get(ctx, "/search/hotkey", nil)
	if err != nil {
		return nil, fmt.Errorf("get hot keys failed: %w", err)
	}

	var response struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Data    []HotKey `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return response.Data, nil
}

// SearchSongs 搜索歌曲
func (c *QQMusicClient) SearchSongs(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	path := fmt.Sprintf("/search/?keyword=%s&type=0&page=%d&size=%d", url.QueryEscape(keyword), page, size)

	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("search songs failed: %w", err)
	}

	var response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    PageResponse `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// SearchSingers 搜索歌手
func (c *QQMusicClient) SearchSingers(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	path := fmt.Sprintf("/search/?keyword=%s&type=9&page=%d&size=%d", url.QueryEscape(keyword), page, size)

	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("search singers failed: %w", err)
	}

	var response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    PageResponse `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// SearchAlbums 搜索专辑
func (c *QQMusicClient) SearchAlbums(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	path := fmt.Sprintf("/search/?keyword=%s&type=8&page=%d&size=%d", url.QueryEscape(keyword), page, size)

	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("search albums failed: %w", err)
	}

	var response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    PageResponse `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// SearchMVs 搜索MV
func (c *QQMusicClient) SearchMVs(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	path := fmt.Sprintf("/search/?keyword=%s&type=12&page=%d&size=%d", url.QueryEscape(keyword), page, size)

	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("search mvs failed: %w", err)
	}

	var response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    PageResponse `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// ===== Lyric模块 =====

// ===== Singer模块 =====

// GetSingerCategories 获取歌手筛选分类
func (c *QQMusicClient) GetSingerCategories(ctx context.Context) ([]CategoryFilter, error) {
	data, err := c.Get(ctx, "/artist/category", nil)
	if err != nil {
		return nil, fmt.Errorf("get singer categories failed: %w", err)
	}

	var response struct {
		Code    int              `json:"code"`
		Message string           `json:"message"`
		Data    []CategoryFilter `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return response.Data, nil
}

// GetSingerListRequest 获取歌手列表请求参数
type GetSingerListRequest struct {
	Area  int // 地区
	Sex   int // 性别
	Genre int // 流派
	Index int // 索引
	Page  int // 页码
	Size  int // 每页数量
}

// GetSingerList 获取歌手列表
func (c *QQMusicClient) GetSingerList(ctx context.Context, req GetSingerListRequest) (*PageResponse, error) {
	path := fmt.Sprintf("/artist/list?area=%d&sex=%d&genre=%d&index=%d&page=%d&size=%d",
		req.Area, req.Sex, req.Genre, req.Index, req.Page, req.Size)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get singer list failed: %w", err)
	}

	var response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    PageResponse `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// GetSingerDetail 获取歌手详情和歌曲
func (c *QQMusicClient) GetSingerDetail(ctx context.Context, singerMid string, page int) (*PageResponse, error) {
	path := fmt.Sprintf("/artist/detail?id=%s&page=%d", singerMid, page)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get singer detail failed: %w", err)
	}

	var response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    PageResponse `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// GetSingerAlbums 获取歌手专辑列表
func (c *QQMusicClient) GetSingerAlbums(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	path := fmt.Sprintf("/artist/albums?id=%s&page=%d&size=%d", singerMid, page, size)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get singer albums failed: %w", err)
	}

	var response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    PageResponse `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// GetSingerMVs 获取歌手MV列表
func (c *QQMusicClient) GetSingerMVs(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	path := fmt.Sprintf("/artist/mvs?id=%s&page=%d&size=%d", singerMid, page, size)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get singer mvs failed: %w", err)
	}

	var response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    PageResponse `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// GetSingerSongs 获取歌手歌曲列表
func (c *QQMusicClient) GetSingerSongs(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	path := fmt.Sprintf("/artist/songs?id=%s&page=%d&size=%d", singerMid, page, size)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get singer songs failed: %w", err)
	}

	var response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    PageResponse `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// ===== Ranking模块 =====

// GetRankingList 获取榜单分类和前三首歌曲
func (c *QQMusicClient) GetRankingList(ctx context.Context) ([]RankingCategory, error) {
	data, err := c.Get(ctx, "/rankings/list", nil)
	if err != nil {
		return nil, fmt.Errorf("get ranking list failed: %w", err)
	}

	var response struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Data    []RankingCategory `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return response.Data, nil
}

// GetRankingDetail 获取榜单详情
func (c *QQMusicClient) GetRankingDetail(ctx context.Context, topID, page, size int, period string) (*PageResponse, error) {
	path := fmt.Sprintf("/rankings/detail?id=%d&page=%d&size=%d&period=%s", topID, page, size, url.QueryEscape(period))
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get ranking detail failed: %w", err)
	}

	var response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    PageResponse `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// ===== Radio模块 =====

// GetRadioCategories 获取电台分类列表
func (c *QQMusicClient) GetRadioCategories(ctx context.Context) ([]Radio, error) {
	data, err := c.Get(ctx, "/radio/category", nil)
	if err != nil {
		return nil, fmt.Errorf("get radio categories failed: %w", err)
	}

	var response struct {
		Code    int     `json:"code"`
		Message string  `json:"message"`
		Data    []Radio `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return response.Data, nil
}

// GetRadioSongs 获取电台歌曲列表
func (c *QQMusicClient) GetRadioSongs(ctx context.Context, radioID int) ([]Song, error) {
	path := fmt.Sprintf("/radio/songlist?id=%d", radioID)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get radio songs failed: %w", err)
	}

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    []Song `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return response.Data, nil
}

// ===== MV模块 =====

// GetMVCategories 获取MV分类
func (c *QQMusicClient) GetMVCategories(ctx context.Context) (*MVCategory, error) {
	data, err := c.Get(ctx, "/mv/category", nil)
	if err != nil {
		return nil, fmt.Errorf("get mv categories failed: %w", err)
	}

	var response struct {
		Code    int        `json:"code"`
		Message string     `json:"message"`
		Data    MVCategory `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// GetMVList 获取MV列表
func (c *QQMusicClient) GetMVList(ctx context.Context, area, version, page, size int) (*PageResponse, error) {
	path := fmt.Sprintf("/mv/list?area=%d&version=%d&page=%d&size=%d", area, version, page, size)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get mv list failed: %w", err)
	}

	var response struct {
		Code    int          `json:"code"`
		Message string       `json:"message"`
		Data    PageResponse `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// GetMVDetail 获取MV详情
func (c *QQMusicClient) GetMVDetail(ctx context.Context, vid string) (*MVDetail, error) {
	path := fmt.Sprintf("/mv/detail?id=%s", vid)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get mv detail failed: %w", err)
	}

	var response struct {
		Code    int      `json:"code"`
		Message string   `json:"message"`
		Data    MVDetail `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// ===== Album模块 =====

// GetAlbumDetail 获取专辑详情
func (c *QQMusicClient) GetAlbumDetail(ctx context.Context, albumMid string) (*Album, error) {
	path := fmt.Sprintf("/album/detail?id=%s", albumMid)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get album detail failed: %w", err)
	}

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    Album  `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// GetAlbumSongs 获取专辑歌曲列表
func (c *QQMusicClient) GetAlbumSongs(ctx context.Context, albumMid string) ([]Song, error) {
	path := fmt.Sprintf("/album/songs?id=%s", albumMid)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get album songs failed: %w", err)
	}

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    []Song `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return response.Data, nil
}
// GetPlaylistDetail 获取歌单详细信息
func (c *QQMusicClient) GetPlaylistDetail(ctx context.Context, dissID string) (*PlaylistDetail, error) {
	path := fmt.Sprintf("/playlist/detail?diss_id=%s", dissID)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get playlist detail failed: %w", err)
	}

	var response struct {
		Code    int            `json:"code"`
		Message string         `json:"message"`
		Data    PlaylistDetail `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}
// ===== Song模块 =====

// GetSongDetail 获取歌曲详情
func (c *QQMusicClient) GetSongDetail(ctx context.Context, songMid string) (*SongDetail, error) {
	path := fmt.Sprintf("/song/detail?id=%s", songMid)
	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get song detail failed: %w", err)
	}

	var response struct {
		Code    int        `json:"code"`
		Message string     `json:"message"`
		Data    SongDetail `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}

// GetLyric 获取歌词
func (c *QQMusicClient) GetLyric(ctx context.Context, songMid string) (*Lyric, error) {
	path := fmt.Sprintf("/lyric?id=%s", songMid)

	data, err := c.Get(ctx, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get lyric failed: %w", err)
	}

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    Lyric  `json:"data"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	if response.Code != 1 {
		return nil, fmt.Errorf("api error: code=%d, msg=%s", response.Code, response.Message)
	}

	return &response.Data, nil
}
