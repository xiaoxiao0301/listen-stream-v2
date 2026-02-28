package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// WarmUpService 缓存预热服务
type WarmUpService struct {
	cache  *CacheLayer
	logger logger.Logger
}

// NewWarmUpService 创建预热服务
func NewWarmUpService(cache *CacheLayer, log logger.Logger) *WarmUpService {
	return &WarmUpService{
		cache:  cache,
		logger: log,
	}
}

// WarmUpEntry 预热条目
type WarmUpEntry struct {
	Key  string
	Data interface{}
	TTL  time.Duration
}

// WarmUp 执行预热
func (w *WarmUpService) WarmUp(ctx context.Context, entries []WarmUpEntry) error {
	w.logger.Info("Starting cache warm up", logger.Int("count", len(entries)))
	start := time.Now()

	success := 0
	failed := 0

	for _, entry := range entries {
		// 序列化数据
		data, err := json.Marshal(entry.Data)
		if err != nil {
			w.logger.Error("Failed to marshal warm up data", logger.String("key", entry.Key), logger.Error(err))
			failed++
			continue
		}

		// 写入缓存
		if err := w.cache.Set(ctx, entry.Key, data, entry.TTL); err != nil {
			w.logger.Error("Failed to warm up cache", logger.String("key", entry.Key), logger.Error(err))
			failed++
			continue
		}

		success++
	}

	elapsed := time.Since(start)
	w.logger.Info("Cache warm up completed",
		logger.Int("success", success),
		logger.Int("failed", failed),
		logger.Duration("elapsed", elapsed),
	)

	if failed > 0 {
		return fmt.Errorf("warm up partially failed: %d/%d entries failed", failed, len(entries))
	}

	return nil
}

// WarmUpFromLoader 从加载器预热（惰性加载）
func (w *WarmUpService) WarmUpFromLoader(
	ctx context.Context,
	keys []string,
	loader func(ctx context.Context, key string) (interface{}, error),
	ttl time.Duration,
) error {
	w.logger.Info("Starting cache warm up from loader", logger.Int("count", len(keys)))
	start := time.Now()

	success := 0
	failed := 0

	for _, key := range keys {
		// 调用loader获取数据
		data, err := loader(ctx, key)
		if err != nil {
			w.logger.Error("Loader failed during warm up", logger.String("key", key), logger.Error(err))
			failed++
			continue
		}

		// 序列化
		bytes, err := json.Marshal(data)
		if err != nil {
			w.logger.Error("Failed to marshal data", logger.String("key", key), logger.Error(err))
			failed++
			continue
		}

		// 写入缓存
		if err := w.cache.Set(ctx, key, bytes, ttl); err != nil {
			w.logger.Error("Failed to set cache", logger.String("key", key), logger.Error(err))
			failed++
			continue
		}

		success++
	}

	elapsed := time.Since(start)
	w.logger.Info("Cache warm up from loader completed",
		logger.Int("success", success),
		logger.Int("failed", failed),
		logger.Duration("elapsed", elapsed),
	)

	if failed > 0 {
		return fmt.Errorf("warm up partially failed: %d/%d keys failed", failed, len(keys))
	}

	return nil
}

// WarmUpBanner 预热Banner数据（示例）
func (w *WarmUpService) WarmUpBanner(ctx context.Context, banners []interface{}) error {
	entries := make([]WarmUpEntry, len(banners))
	for i, banner := range banners {
		entries[i] = WarmUpEntry{
			Key:  fmt.Sprintf("banner:%d", i),
			Data: banner,
			TTL:  24 * time.Hour,
		}
	}
	return w.WarmUp(ctx, entries)
}

// WarmUpHotPlaylists 预热热门歌单（示例）
func (w *WarmUpService) WarmUpHotPlaylists(ctx context.Context, playlists []interface{}) error {
	entries := make([]WarmUpEntry, len(playlists))
	for i, playlist := range playlists {
		entries[i] = WarmUpEntry{
			Key:  fmt.Sprintf("playlist:%d", i),
			Data: playlist,
			TTL:  1 * time.Hour,
		}
	}
	return w.WarmUp(ctx, entries)
}

// ScheduledWarmUp 定期预热任务
func (w *WarmUpService) ScheduledWarmUp(
	ctx context.Context,
	interval time.Duration,
	warmUpFunc func(ctx context.Context) error,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	w.logger.Info("Starting scheduled warm up", logger.Duration("interval", interval))

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Scheduled warm up stopped")
			return
		case <-ticker.C:
			if err := warmUpFunc(ctx); err != nil {
				w.logger.Error("Scheduled warm up failed", logger.Error(err))
			}
		}
	}
}
