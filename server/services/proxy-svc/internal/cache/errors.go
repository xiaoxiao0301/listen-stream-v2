package cache

import "errors"

var (
	// ErrCacheMiss 缓存未命中
	ErrCacheMiss = errors.New("cache miss")

	// ErrCacheExpired 缓存已过期
	ErrCacheExpired = errors.New("cache expired")

	// ErrInvalidData 无效的缓存数据
	ErrInvalidData = errors.New("invalid cache data")
)
