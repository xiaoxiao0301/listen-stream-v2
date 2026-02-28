package upstream

import "errors"

var (
	// ErrUpstreamUnavailable 上游服务不可用
	ErrUpstreamUnavailable = errors.New("upstream service unavailable")

	// ErrCircuitOpen 熔断器打开
	ErrCircuitOpen = errors.New("circuit breaker is open")

	// ErrRateLimitExceeded 速率限制超出
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrSongNotFound 歌曲未找到
	ErrSongNotFound = errors.New("song not found")

	// ErrInvalidResponse 无效响应
	ErrInvalidResponse = errors.New("invalid response from upstream")

	// ErrTimeout 请求超时
	ErrTimeout = errors.New("request timeout")
)
