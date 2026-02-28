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

// NetEaseClient is the NetEase Cloud Music API client
type NetEaseClient struct {
	config UpstreamConfig
	client *http.Client
	logger logger.Logger
}

// NewNetEaseClient creates a new NetEase client instance
func NewNetEaseClient(config UpstreamConfig, log logger.Logger) *NetEaseClient {
	return &NetEaseClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		logger: log,
	}
}

// GetSongURL gets song play URL with fallback support
func (c *NetEaseClient) GetSongURL(ctx context.Context, songMid, songName string) (*SongURL, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

// SearchSong searches for a song by name and artist
func (c *NetEaseClient) SearchSong(ctx context.Context, songName, artistName string) (*SearchResult, error) {
	// Build query
	query := songName
	if artistName != "" {
		query = fmt.Sprintf("%s %s", songName, artistName)
	}

	// Build request URL
	reqURL := fmt.Sprintf("%s/search", c.config.BaseURL)
	params := url.Values{}
	params.Add("keywords", query)
	params.Add("type", "1") // 1 = single track
	params.Add("limit", "10")
	reqURL += "?" + params.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ListenStream/1.0)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://music.163.com")

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response
	var result struct {
		Code   int    `json:"code"`
		Result struct {
			Songs []struct {
				ID     int64  `json:"id"`
				Name   string `json:"name"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
				Album struct {
					Name   string `json:"name"`
					PicURL string `json:"picUrl"`
				} `json:"album"`
			} `json:"songs"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("API error code: %d", result.Code)
	}

	if len(result.Result.Songs) == 0 {
		return nil, ErrSongNotFound
	}

	// Convert to standard format
	searchResult := &SearchResult{
		Songs: make([]Song, 0, len(result.Result.Songs)),
	}

	for _, song := range result.Result.Songs {
		var artistNames string
		if len(song.Artists) > 0 {
			artistNames = song.Artists[0].Name
		}
		
		searchResult.Songs = append(searchResult.Songs, Song{
			ID:       fmt.Sprintf("%d", song.ID),
			Name:     song.Name,
			Artist:   artistNames,
			Album:    song.Album.Name,
			CoverURL: song.Album.PicURL,
		})
	}

	return searchResult, nil
}

// GetBanners gets homepage banners (not implemented for NetEase)
func (c *NetEaseClient) GetBanners(ctx context.Context) ([]Banner, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

// HealthCheck checks if the NetEase API is healthy
func (c *NetEaseClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// NetEase doesn't have a dedicated health endpoint, so we check banner API
	reqURL := fmt.Sprintf("%s/banner", c.config.BaseURL)
	
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

// qualityToBitrate converts quality string to NetEase bitrate
func (c *NetEaseClient) qualityToBitrate(quality string) string {
	switch quality {
	case "high":
		return "320000"
	case "medium":
		return "192000"
	case "low":
		return "128000"
	default:
		return "192000"
	}
}

// Stub implementations for ClientInterface (not implemented for NetEase)
func (c *NetEaseClient) GetDailyRecommendPlaylists(ctx context.Context) ([]Playlist, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetRecommendPlaylists(ctx context.Context) ([]Playlist, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetNewSongs(ctx context.Context, typ int) ([]Song, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetNewAlbums(ctx context.Context, typ int) ([]Album, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetPlaylistCategories(ctx context.Context) ([]PlaylistCategory, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetPlaylistsByCategory(ctx context.Context, req GetPlaylistsByCategoryRequest) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetPlaylistDetail(ctx context.Context, dissID string) (*PlaylistDetail, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetSongDetail(ctx context.Context, songMid string) (*SongDetail, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetLyric(ctx context.Context, songMid string) (*Lyric, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetAlbumDetail(ctx context.Context, albumMid string) (*Album, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetAlbumSongs(ctx context.Context, albumMid string) ([]Song, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetHotKeys(ctx context.Context) ([]HotKey, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) SearchSongs(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) SearchSingers(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) SearchAlbums(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) SearchMVs(ctx context.Context, keyword string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetSingerCategories(ctx context.Context) ([]CategoryFilter, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetSingerList(ctx context.Context, req GetSingerListRequest) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetSingerDetail(ctx context.Context, singerMid string, page int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetSingerAlbums(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetSingerMVs(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetSingerSongs(ctx context.Context, singerMid string, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetRankingList(ctx context.Context) ([]RankingCategory, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetRankingDetail(ctx context.Context, topID, page, size int, period string) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetRadioCategories(ctx context.Context) ([]Radio, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetRadioSongs(ctx context.Context, radioID int) ([]Song, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetMVCategories(ctx context.Context) (*MVCategory, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetMVList(ctx context.Context, area, version, page, size int) (*PageResponse, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}

func (c *NetEaseClient) GetMVDetail(ctx context.Context, vid string) (*MVDetail, error) {
	return nil, fmt.Errorf("not implemented for NetEase")
}
