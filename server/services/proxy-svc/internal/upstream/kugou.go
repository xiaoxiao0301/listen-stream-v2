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

// KugouClient is the Kugou Music API client
type KugouClient struct {
	config UpstreamConfig
	client *http.Client
	logger logger.Logger
}

// NewKugouClient creates a new Kugou client instance
func NewKugouClient(config UpstreamConfig, log logger.Logger) *KugouClient {
	return &KugouClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		logger: log,
	}
}

// GetSongURL gets song play URL with fallback support
func (c *KugouClient) GetSongURL(ctx context.Context, songMid, songName string) (*SongURL, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

// SearchSong searches for a song by name and artist
func (c *KugouClient) SearchSong(ctx context.Context, songName, artistName string) (*SearchResult, error) {
	// Build query
	query := songName
	if artistName != "" {
		query = fmt.Sprintf("%s - %s", songName, artistName)
	}

	// Build request URL
	reqURL := fmt.Sprintf("%s/v1/search/song", c.config.BaseURL)
	params := url.Values{}
	params.Add("keyword", query)
	params.Add("page", "1")
	params.Add("pagesize", "10")
	reqURL += "?" + params.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ListenStream/1.0)")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response
	var result struct {
		Status int    `json:"status"`
		Error  string `json:"error"`
		Data   struct {
			Lists []struct {
				FileHash   string `json:"FileHash"`
				SongName   string `json:"SongName"`
				SingerName string `json:"SingerName"`
				AlbumName  string `json:"AlbumName"`
			} `json:"lists"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != 1 {
		if result.Error != "" {
			return nil, fmt.Errorf("API error: %s", result.Error)
		}
		return nil, ErrSongNotFound
	}

	if len(result.Data.Lists) == 0 {
		return nil, ErrSongNotFound
	}

	// Convert to standard format
	searchResult := &SearchResult{
		Songs: make([]Song, 0, len(result.Data.Lists)),
	}

	for _, song := range result.Data.Lists {
		searchResult.Songs = append(searchResult.Songs, Song{
			ID:     song.FileHash,
			Name:   song.SongName,
			Artist: song.SingerName,
			Album:  song.AlbumName,
		})
	}

	return searchResult, nil
}

// GetBanners gets homepage banners (not implemented for Kugou)
func (c *KugouClient) GetBanners(ctx context.Context) ([]Banner, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

// HealthCheck checks if the Kugou API is healthy
func (c *KugouClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("%s/v1/banner", c.config.BaseURL)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ListenStream/1.0)")

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

// Stub implementations for ClientInterface (not implemented for Kugou)
func (c *KugouClient) GetDailyRecommendPlaylists(ctx context.Context) ([]Playlist, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetRecommendPlaylists(ctx context.Context) ([]Playlist, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetNewSongs(ctx context.Context, typ int) ([]Song, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetNewAlbums(ctx context.Context, typ int) ([]Album, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetPlaylistCategories(ctx context.Context) ([]PlaylistCategory, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetPlaylistsByCategory(ctx context.Context, req GetPlaylistsByCategoryRequest) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetPlaylistDetail(ctx context.Context, dissID string) (*PlaylistDetail, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetSongDetail(ctx context.Context, songMid string) (*SongDetail, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetLyric(ctx context.Context, songMid string) (*Lyric, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetAlbumDetail(ctx context.Context, albumMid string) (*Album, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetAlbumSongs(ctx context.Context, albumMid string) ([]Song, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetHotKeys(ctx context.Context) ([]HotKey, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) SearchSongs(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) SearchSingers(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) SearchAlbums(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) SearchMVs(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetSingerCategories(ctx context.Context) ([]CategoryFilter, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetSingerList(ctx context.Context, req GetSingerListRequest) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetSingerDetail(ctx context.Context, singerMid string, page int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetSingerAlbums(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetSingerMVs(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetSingerSongs(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetRankingList(ctx context.Context) ([]RankingCategory, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetRankingDetail(ctx context.Context, topID, page, size int, period string) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetRadioCategories(ctx context.Context) ([]Radio, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetRadioSongs(ctx context.Context, radioID int) ([]Song, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetMVCategories(ctx context.Context) (*MVCategory, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetMVList(ctx context.Context, area, version, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}

func (c *KugouClient) GetMVDetail(ctx context.Context, vid string) (*MVDetail, error) {
	return nil, fmt.Errorf("not implemented for Kugou")
}
