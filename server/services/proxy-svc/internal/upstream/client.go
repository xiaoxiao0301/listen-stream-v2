package upstream

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// Client HTTP客户端基类
type Client struct {
	name        string
	baseURL     string
	httpClient  *http.Client
	breaker     *CircuitBreaker
	rateLimiter *RateLimiter
	retry       *RetryConfig
	logger      logger.Logger
}

// ClientConfig 客户端配置
type ClientConfig struct {
	Name            string
	BaseURL         string
	Timeout         time.Duration
	MaxRetries      int
	RateLimit       int // 每秒请求数
	BreakerSettings BreakerSettings
}

// NewClient 创建HTTP客户端
func NewClient(config ClientConfig, log logger.Logger) *Client {
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	breaker := NewCircuitBreaker(config.BreakerSettings, log)
	rateLimiter := NewRateLimiter(config.RateLimit)
	retry := &RetryConfig{
		MaxRetries:  config.MaxRetries,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     2 * time.Second,
		Multiplier:  2.0,
	}

	return &Client{
		name:        config.Name,
		baseURL:     config.BaseURL,
		httpClient:  httpClient,
		breaker:     breaker,
		rateLimiter: rateLimiter,
		retry:       retry,
		logger:      log,
	}
}

// Get 发送GET请求（带熔断、限流、重试）
func (c *Client) Get(ctx context.Context, path string, headers map[string]string) ([]byte, error) {
	// 检查速率限制
	if !c.rateLimiter.Allow() {
		c.logger.Warn("Rate limit exceeded", logger.String("upstream", c.name))
		return nil, ErrRateLimitExceeded
	}

	// 使用熔断器执行请求
	result, err := c.breaker.Execute(func() (interface{}, error) {
		// 带重试的请求
		return c.doRequestWithRetry(ctx, "GET", path, headers)
	})

	if err != nil {
		return nil, err
	}

	return result.([]byte), nil
}

// Post 发送POST请求
func (c *Client) Post(ctx context.Context, path string, body []byte, headers map[string]string) ([]byte, error) {
	if !c.rateLimiter.Allow() {
		c.logger.Warn("Rate limit exceeded", logger.String("upstream", c.name))
		return nil, ErrRateLimitExceeded
	}

	result, err := c.breaker.Execute(func() (interface{}, error) {
		return c.doRequestWithRetry(ctx, "POST", path, headers)
	})

	if err != nil {
		return nil, err
	}

	return result.([]byte), nil
}

// doRequestWithRetry 执行带重试的HTTP请求
func (c *Client) doRequestWithRetry(ctx context.Context, method, path string, headers map[string]string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= c.retry.MaxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避
			waitTime := c.retry.CalculateBackoff(attempt)
			c.logger.Debug("Retrying request",
				logger.String("upstream", c.name),
				logger.String("path", path),
				logger.Int("attempt", attempt),
				logger.Duration("wait", waitTime),
			)

			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// 执行请求
		data, err := c.doRequest(ctx, method, path, headers)
		if err == nil {
			if attempt > 0 {
				c.logger.Info("Request succeeded after retry",
					logger.String("upstream", c.name),
					logger.Int("attempt", attempt),
				)
			}
			return data, nil
		}

		lastErr = err

		// 检查是否应该重试
		if !c.shouldRetry(err) {
			c.logger.Debug("Error not retryable",
				logger.String("upstream", c.name),
				logger.Error(err),
			)
			break
		}
	}

	c.logger.Warn("Request failed after all retries",
		logger.String("upstream", c.name),
		logger.Int("maxRetries", c.retry.MaxRetries),
		logger.Error(lastErr),
	)

	return nil, fmt.Errorf("request failed after %d retries: %w", c.retry.MaxRetries, lastErr)
}

// doRequest 执行单次HTTP请求
func (c *Client) doRequest(ctx context.Context, method, path string, headers map[string]string) ([]byte, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 设置headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 设置默认headers
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "ListenStream/1.0")
	}

	// 执行请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	// 检查状态码
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("%w: status code %d", ErrUpstreamUnavailable, resp.StatusCode)
	}

	if resp.StatusCode == 404 {
		return nil, ErrSongNotFound
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// shouldRetry 判断错误是否应该重试
func (c *Client) shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// 超时可以重试
	if err == context.DeadlineExceeded || err == ErrTimeout {
		return true
	}

	// 上游不可用可以重试
	if err == ErrUpstreamUnavailable {
		return true
	}

	// 歌曲未找到不重试
	if err == ErrSongNotFound {
		return false
	}

	return false
}

// Name 获取客户端名称
func (c *Client) Name() string {
	return c.name
}

// Stats 获取统计信息
func (c *Client) Stats() ClientStats {
	return ClientStats{
		Name:          c.name,
		BreakerState:  c.breaker.State(),
		BreakerStats:  c.breaker.Stats(),
		RateLimitUsed: c.rateLimiter.Used(),
	}
}

// ClientStats 客户端统计
type ClientStats struct {
	Name          string
	BreakerState  string
	BreakerStats  BreakerStats
	RateLimitUsed int
}
