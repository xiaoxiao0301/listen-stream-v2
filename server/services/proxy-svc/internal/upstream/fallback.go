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

// FallbackClient Fallback API客户端
// 用于从Joox、NetEase、Kugou等源获取歌曲
type FallbackClient struct {
	baseURL    string
	httpClient *http.Client
	logger     logger.Logger
}

// NewFallbackClient 创建Fallback客户端
func NewFallbackClient(baseURL string, log logger.Logger) *FallbackClient {
	return &FallbackClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: log,
	}
}

// SearchSong 搜索歌曲
// source: joox, netease, kugou
func (f *FallbackClient) SearchSong(ctx context.Context, source, songName string) (*FallbackSongInfo, error) {
	// 构建URL: fallbackURL?types=search&source=joox&name=歌名
	reqURL := fmt.Sprintf("%s?types=search&source=%s&name=%s",
		f.baseURL,
		source,
		url.QueryEscape(songName),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	// 解析响应
	var results []FallbackSongInfo
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	if len(results) == 0 {
		return nil, ErrSongNotFound
	}

	// 返回第一个匹配结果
	return &results[0], nil
}

// GetSongURL 获取歌曲播放URL
func (f *FallbackClient) GetSongURL(ctx context.Context, source, songID string) (*FallbackSongURL, error) {
	// 构建URL: fallbackURL?types=url&source=joox&id=歌曲ID
	reqURL := fmt.Sprintf("%s?types=url&source=%s&id=%s",
		f.baseURL,
		source,
		url.QueryEscape(songID),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	// 解析响应
	var result FallbackSongURL
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	// 检查URL是否为空
	if result.URL == "" {
		return nil, ErrSongNotFound
	}

	return &result, nil
}
