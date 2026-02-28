package upstream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// JooxClient is the Joox Music API client
type JooxClient struct {
	config UpstreamConfig
	client *http.Client
	logger logger.Logger
}

// NewJooxClient creates a new Joox client instance
func NewJooxClient(config UpstreamConfig, log logger.Logger) *JooxClient {
	return &JooxClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		logger: log,
	}
}

// GetSongURL gets song play URL with fallback support
func (c *JooxClient) GetSongURL(ctx context.Context, songMid, songName string) (*SongURL, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

// SearchSong searches for a song by name and artist
func (c *JooxClient) SearchSong(ctx context.Context, songName, artistName string) (*SearchResult, error) {
	// Build query
	query := songName
	if artistName != "" {
		query = fmt.Sprintf("%s %s", songName, artistName)
	}

	// Build request URL
	reqURL := fmt.Sprintf("%s/v1/search/song", c.config.BaseURL)
	params := url.Values{}
	params.Add("keyword", query)
	params.Add("limit", "10")
	reqURL += "?" + params.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("User-Agent", "ListenStream/1.0")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response
	var result struct {
		Code int           `json:"code"`
		Data *SearchResult `json:"data"`
		Msg  string        `json:"msg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s (code: %d)", result.Msg, result.Code)
	}

	if result.Data == nil || len(result.Data.Songs) == 0 {
		return nil, ErrSongNotFound
	}

	return result.Data, nil
}

// GetBanners gets homepage banners
func (c *JooxClient) GetBanners(ctx context.Context) ([]Banner, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

// HealthCheck checks if the Joox API is healthy
func (c *JooxClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("%s/v1/health", c.config.BaseURL)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy: status code %d", resp.StatusCode)
	}

	return nil
}

// Stub implementations for ClientInterface (not implemented for Joox)
func (c *JooxClient) GetDailyRecommendPlaylists(ctx context.Context) ([]Playlist, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetRecommendPlaylists(ctx context.Context) ([]Playlist, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetNewSongs(ctx context.Context, typ int) ([]Song, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetNewAlbums(ctx context.Context, typ int) ([]Album, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetPlaylistCategories(ctx context.Context) ([]PlaylistCategory, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetPlaylistsByCategory(ctx context.Context, req GetPlaylistsByCategoryRequest) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetPlaylistDetail(ctx context.Context, dissID string) (*PlaylistDetail, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetSongDetail(ctx context.Context, songMid string) (*SongDetail, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetLyric(ctx context.Context, songMid string) (*Lyric, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetAlbumDetail(ctx context.Context, albumMid string) (*Album, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetAlbumSongs(ctx context.Context, albumMid string) ([]Song, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetHotKeys(ctx context.Context) ([]HotKey, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) SearchSongs(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) SearchSingers(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) SearchAlbums(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) SearchMVs(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetSingerCategories(ctx context.Context) ([]CategoryFilter, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetSingerList(ctx context.Context, req GetSingerListRequest) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetSingerDetail(ctx context.Context, singerMid string, page int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetSingerAlbums(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetSingerMVs(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetSingerSongs(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetRankingList(ctx context.Context) ([]RankingCategory, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetRankingDetail(ctx context.Context, topID, page, size int, period string) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetRadioCategories(ctx context.Context) ([]Radio, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetRadioSongs(ctx context.Context, radioID int) ([]Song, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetMVCategories(ctx context.Context) (*MVCategory, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetMVList(ctx context.Context, area, version, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}

func (c *JooxClient) GetMVDetail(ctx context.Context, vid string) (*MVDetail, error) {
	return nil, fmt.Errorf("not implemented for Joox")
}
