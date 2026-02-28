package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/proxy-svc/internal/cache"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

func main() {
	// 初始化日志
	log := logger.Default()
	log.Info("Starting proxy-svc")

	// 初始化Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("Failed to connect to Redis", logger.Error(err))
	}
	log.Info("Connected to Redis")

	// 初始化三级缓存
	redisCache := cache.NewRedisCache(redisClient, "proxy")
	cacheConfig := cache.DefaultCacheConfig()
	cacheLayer := cache.NewCacheLayer(redisCache, cacheConfig, log)

	// 初始化预热服务
	warmupService := cache.NewWarmUpService(cacheLayer, log)

	// 启动时预热热点数据
	if err := warmUpHotData(ctx, warmupService); err != nil {
		log.Error("Failed to warm up cache", logger.Error(err))
	}

	// 启动定期预热任务（每小时预热一次）
	go warmupService.ScheduledWarmUp(ctx, 1*time.Hour, func(ctx context.Context) error {
		return warmUpHotData(ctx, warmupService)
	})

	// 启动定期清理过期缓存（L1 内存缓存）
	go startCacheCleanup(ctx, cacheLayer, log)

	// 初始化HTTP服务器
	router := setupRouter(cacheLayer, log)

	// 启动服务器
	addr := getEnv("HTTP_ADDR", ":8002")
	log.Info("Starting HTTP server", logger.String("addr", addr))

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatal("Failed to start HTTP server", logger.Error(err))
		}
	}()

	<-quit
	log.Info("Shutting down proxy-svc")

	// 清理
	if err := redisClient.Close(); err != nil {
		log.Error("Failed to close Redis client", logger.Error(err))
	}
}

// setupRouter 设置路由
func setupRouter(cacheLayer *cache.CacheLayer, log logger.Logger) *gin.Engine {
	router := gin.Default()

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 缓存统计
	router.GET("/cache/stats", func(c *gin.Context) {
		stats := cacheLayer.Stats()
		c.JSON(200, stats)
	})

	// 示例：获取歌曲URL（带三级缓存）
	router.GET("/song/:id/url", func(c *gin.Context) {
		songID := c.Param("id")
		cacheKey := fmt.Sprintf("song:url:%s", songID)

		// 使用三级缓存获取数据
		data, err := cacheLayer.GetWithFallback(
			c.Request.Context(),
			cacheKey,
			func(ctx context.Context) ([]byte, error) {
				// 模拟从上游API获取数据
					log.Info("Fetching song URL from upstream", logger.String("songID", songID))
				// 实际应该调用上游API
				songURL := map[string]string{
					"songID": songID,
					"url":    fmt.Sprintf("http://music.example.com/%s.mp3", songID),
				}
				return json.Marshal(songURL)
			},
			30*time.Minute,
		)

		if err != nil {
			log.Error("Failed to get song URL", logger.String("songID", songID), logger.Error(err))
			c.JSON(500, gin.H{"error": "Failed to get song URL"})
			return
		}

		var result map[string]string
		if err := json.Unmarshal(data, &result); err != nil {
			log.Error("Failed to unmarshal data", logger.Error(err))
			c.JSON(500, gin.H{"error": "Internal error"})
			return
		}

		c.JSON(200, result)
	})

	return router
}

// warmUpHotData 预热热点数据
func warmUpHotData(ctx context.Context, warmupService *cache.WarmUpService) error {
	// 预热Banner（示例）
	banners := []interface{}{
		map[string]string{"id": "1", "image": "banner1.jpg", "link": "/playlist/hot"},
		map[string]string{"id": "2", "image": "banner2.jpg", "link": "/playlist/new"},
	}
	if err := warmupService.WarmUpBanner(ctx, banners); err != nil {
		return err
	}

	// 预热热门歌单（示例）
	playlists := []interface{}{
		map[string]string{"id": "100", "name": "热门歌单"},
		map[string]string{"id": "200", "name": "新歌推荐"},
	}
	if err := warmupService.WarmUpHotPlaylists(ctx, playlists); err != nil {
		return err
	}

	return nil
}

// startCacheCleanup 启动定期清理任务
func startCacheCleanup(ctx context.Context, cacheLayer *cache.CacheLayer, log logger.Logger) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 清理L1过期条目
			expired := cacheLayer.CleanExpired()
			if expired > 0 {
				log.Debug("Cleaned expired cache entries", logger.Int("count", expired))
			}
		}
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
