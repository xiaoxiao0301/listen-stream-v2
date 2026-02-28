package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/proxy-svc/internal/cache"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/proxy-svc/internal/client"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/proxy-svc/internal/handler"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/proxy-svc/internal/middleware"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/proxy-svc/internal/upstream"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/consul"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

func main() {
	// 初始化日志
	log := logger.Default()
	log.Info("Starting proxy-svc")

	ctx := context.Background()

	// 初始化Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("Failed to connect to Redis", logger.String("error", err.Error()))
	}
	log.Info("Connected to Redis")

	// 初始化三级缓存
	redisCache := cache.NewRedisCache(redisClient, "proxy")
	cacheConfig := cache.DefaultCacheConfig()
	cacheLayer := cache.NewCacheLayer(redisCache, cacheConfig, log)

	// 初始化预热服务
	warmupService := cache.NewWarmUpService(cacheLayer, log)

	// 启动时预热热点数据（非阻塞）
	go func() {
		if err := warmUpHotData(ctx, warmupService); err != nil {
			log.Error("Failed to warm up cache", logger.String("error", err.Error()))
		}
	}()

	// 启动定期预热任务（每小时预热一次）
	go warmupService.ScheduledWarmUp(ctx, 1*time.Hour, func(ctx context.Context) error {
		return warmUpHotData(ctx, warmupService)
	})

	// 启动定期清理过期缓存（L1 内存缓存）
	go startCacheCleanup(ctx, cacheLayer, log)

	// Consul服务发现
	consulAddr := getEnv("CONSUL_ADDR", "localhost:8500")
	serviceDiscovery, err := consul.NewServiceDiscovery(consulAddr, 30*time.Second, log)
	if err != nil {
		log.Warn("Failed to create Consul service discovery, using static addresses", logger.String("error", err.Error()))
	}

	// 初始化gRPC客户端池
	grpcPool := client.NewClientPool(log)
	defer grpcPool.Close()

	// 获取服务地址（优先使用Consul，失败则使用环境变量）
	authAddr := getServiceAddr(serviceDiscovery, "auth-svc", getEnv("AUTH_SERVICE_ADDR", "localhost:9001"), log)
	userAddr := getServiceAddr(serviceDiscovery, "user-svc", getEnv("USER_SERVICE_ADDR", "localhost:9003"), log)

	var authClient *client.AuthClient
	var userClient *client.UserClient

	// 尝试连接auth-svc（非阻塞，失败不影响启动）
	authConn, err := grpcPool.GetConnection(ctx, "auth-svc", authAddr)
	if err != nil {
		log.Warn("Failed to connect to auth-svc, will retry later", logger.String("error", err.Error()))
	} else {
		authClient = client.NewAuthClient(authConn, authAddr, log)
		log.Info("Connected to auth-svc", logger.String("addr", authAddr))
	}

	// 尝试连接user-svc（非阻塞，失败不影响启动）
	userConn, err := grpcPool.GetConnection(ctx, "user-svc", userAddr)
	if err != nil {
		log.Warn("Failed to connect to user-svc, will retry later", logger.String("error", err.Error()))
	} else {
		userClient = client.NewUserClient(userConn, userAddr, log)
		log.Info("Connected to user-svc", logger.String("addr", userAddr))
	}

	// 启动服务发现监听（动态更新服务地址）
	if serviceDiscovery != nil {
		go watchServiceChanges(serviceDiscovery, "auth-svc", grpcPool, &authClient, log)
		go watchServiceChanges(serviceDiscovery, "user-svc", grpcPool, &userClient, log)
	}

	// 初始化上游客户端 - QQ Music (主)
	qqMusicConfig := upstream.UpstreamConfig{
		Name:    "QQ Music",
		BaseURL: getEnv("QQ_MUSIC_BASE_URL", "https://api.qq.music.example.com"),
		Timeout: 5 * time.Second,
		MaxRetries:  3,
		RateLimit:   20, // 20 req/s
		Cookie:      getEnv("QQ_MUSIC_COOKIE", ""),
		FallbackURL: getEnv("FALLBACK_API_URL", ""),
	}

	qqMusic := upstream.NewQQMusicClient(qqMusicConfig, log)
	log.Info("QQ Music client initialized", logger.String("base_url", qqMusicConfig.BaseURL))

	// 初始化Joox客户端 (备选1)
	jooxConfig := upstream.UpstreamConfig{
		Name:    "Joox Music",
		BaseURL: getEnv("JOOX_BASE_URL", "https://api.joox.com"),
		Timeout: 5 * time.Second,
		MaxRetries: 3,
		RateLimit:  15,
		Cookie:     getEnv("JOOX_COOKIE", ""),
	}
	jooxClient := upstream.NewJooxClient(jooxConfig, log)
	log.Info("Joox Music client initialized", logger.String("base_url", jooxConfig.BaseURL))

	// 初始化NetEase客户端 (备选2)
	neteaseConfig := upstream.UpstreamConfig{
		Name:    "NetEase Cloud Music",
		BaseURL: getEnv("NETEASE_BASE_URL", "https://music.163.com/api"),
		Timeout: 5 * time.Second,
		MaxRetries: 3,
		RateLimit:  15,
		Cookie:     getEnv("NETEASE_COOKIE", ""),
	}
	neteaseClient := upstream.NewNetEaseClient(neteaseConfig, log)
	log.Info("NetEase Cloud Music client initialized", logger.String("base_url", neteaseConfig.BaseURL))

	// 初始化Kugou客户端 (备选3)
	kugouConfig := upstream.UpstreamConfig{
		Name:    "Kugou Music",
		BaseURL: getEnv("KUGOU_BASE_URL", "https://m.kugou.com"),
		Timeout: 5 * time.Second,
		MaxRetries: 3,
		RateLimit:  15,
		Cookie:     getEnv("KUGOU_COOKIE", ""),
	}
	kugouClient := upstream.NewKugouClient(kugouConfig, log)
	log.Info("Kugou Music client initialized", logger.String("base_url", kugouConfig.BaseURL))

	// 创建FallbackManager (四源智能fallback)
	upstreamClients := []upstream.ClientInterface{qqMusic, jooxClient, neteaseClient, kugouClient}
	upstreamNames := []string{"qq-music", "joox", "netease", "kugou"}
	fallbackManager := upstream.NewFallbackManager(upstreamClients, upstreamNames, log)
	log.Info("FallbackManager initialized with 4 sources")

	// 创建健康检查器
	healthChecker := NewHealthChecker(redisClient, authClient, userClient, upstreamClients, upstreamNames, log)
	log.Info("HealthChecker initialized")

	// 注册服务到Consul
	var registry *consul.ServiceRegistry
	if getEnv("ENABLE_CONSUL", "false") == "true" {
		serviceAddr := getEnv("SERVICE_ADDR", getLocalIP()+":8002")
		registryConfig := consul.RegistryConfig{
			ConsulAddr:  consulAddr,
			ServiceName: "proxy-svc",
			ServiceAddr: serviceAddr,
			ServiceTags: []string{"api-gateway", "http"},
			HealthCheck: consul.HealthCheckConfig{
				HTTP:                           fmt.Sprintf("http://%s/api/health", serviceAddr),
				Interval:                       10 * time.Second,
				Timeout:                        5 * time.Second,
				DeregisterCriticalServiceAfter: 30 * time.Second,
			},
		}

		registry, err = consul.NewServiceRegistry(registryConfig, log)
		if err != nil {
			log.Warn("Failed to register service to Consul", logger.String("error", err.Error()))
		}
	}

	router := setupRouter(fallbackManager, authClient, userClient, cacheLayer, healthChecker, log)

	// 启动HTTP服务器
	httpAddr := getEnv("HTTP_ADDR", ":8002")
	log.Info("Starting HTTP server", logger.String("addr", httpAddr))

	srv := &http.Server{
		Addr:         httpAddr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start HTTP server", logger.String("error", err.Error()))
		}
	}()

	<-quit
	log.Info("Shutting down proxy-svc")

	// 创建关闭上下文（最多等待5秒）
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 从Consul注销服务
	if registry != nil {
		if err := registry.Deregister(shutdownCtx); err != nil {
			log.Error("Failed to deregister service from Consul", logger.String("error", err.Error()))
		}
	}

	// 优雅关闭HTTP服务器

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Failed to shutdown HTTP server gracefully", logger.String("error", err.Error()))
	}

	// 清理
	if err := redisClient.Close(); err != nil {
		log.Error("Failed to close Redis client", logger.String("error", err.Error()))
	}

	log.Info("proxy-svc stopped")
}

// setupRouter 设置路由
func setupRouter(
	upstreamClient upstream.ClientInterface,
	authClient *client.AuthClient,
	userClient *client.UserClient,
	cacheLayer *cache.CacheLayer,
	healthChecker *HealthChecker,
	log logger.Logger,
) *gin.Engine {
	// 生产模式
	if getEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 全局中间件栈（按顺序执行）
	router.Use(middleware.Recovery(log))             // 1. Panic恢复
	router.Use(middleware.RequestID())               // 2. 注入RequestID
	router.Use(middleware.Tracing("proxy-svc"))      // 3. OpenTelemetry追踪
	router.Use(middleware.Logging(log))              // 4. 日志记录
	router.Use(middleware.CORS())                    // 5. 跨域处理
	rateLimiter := middleware.NewRateLimiter(100, 200, 500, 1000) // IP: 100/s, User: 500/s
	router.Use(rateLimiter.Limit())                  // 6. 速率限制

	// 初始化handler (使用FallbackManager，支持多源fallback)
	h := handler.NewHandler(upstreamClient, log)

	// 初始化用户handler（如果userClient可用）
	var userHandler *handler.UserHandler
	if userClient != nil {
		userHandler = handler.NewUserHandler(userClient, log)
	}

	// JWT密钥（生产环境应该从配置中心读取）
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key-change-in-production")

	// ===== 公开API（无需认证） =====
	public := router.Group("/api")
	{
		// 健康检查（增强版，检查Redis、上游服务、gRPC服务）
		public.GET("/health", healthChecker.Check)

		// 缓存统计
		public.GET("/cache/stats", func(c *gin.Context) {
			stats := cacheLayer.Stats()
			c.JSON(200, stats)
		})

		// 首页模块
		home := public.Group("/home")
		{
			home.GET("/banners", h.GetBanners)
			home.GET("/recommend-playlists", h.GetRecommendPlaylists)
			home.GET("/new-songs", h.GetNewSongs)
			home.GET("/new-albums", h.GetNewAlbums)
		}

		// 歌单模块
		playlist := public.Group("/playlist")
		{
			playlist.GET("/categories", h.GetPlaylistCategories)
			playlist.GET("/list", h.GetPlaylistsByCategory)
			playlist.GET("/detail", h.GetPlaylistDetail)
		}

		// 歌手模块
		singer := public.Group("/singer")
		{
			singer.GET("/categories", h.GetSingerCategories)
			singer.GET("/list", h.GetSingerList)
			singer.GET("/detail", h.GetSingerDetail)
			singer.GET("/albums", h.GetSingerAlbums)
			singer.GET("/mvs", h.GetSingerMVs)
			singer.GET("/songs", h.GetSingerSongs)
		}

		// 排行榜模块
		ranking := public.Group("/ranking")
		{
			ranking.GET("/list", h.GetRankingList)
			ranking.GET("/detail", h.GetRankingDetail)
		}

		// 电台模块
		radio := public.Group("/radio")
		{
			radio.GET("/categories", h.GetRadioCategories)
			radio.GET("/songs", h.GetRadioSongs)
		}

		// MV模块
		mv := public.Group("/mv")
		{
			mv.GET("/categories", h.GetMVCategories)
			mv.GET("/list", h.GetMVList)
			mv.GET("/detail", h.GetMVDetail)
		}

		// 专辑模块
		album := public.Group("/album")
		{
			album.GET("/detail", h.GetAlbumDetail)
			album.GET("/songs", h.GetAlbumSongs)
		}

		// 歌曲模块（部分接口公开）
		song := public.Group("/song")
		{
			song.GET("/detail", h.GetSongDetail)
			song.GET("/url", h.GetSongURL) // 播放URL（带Fallback）
			song.GET("/lyric", h.GetLyric)
		}

		// 搜索模块
		search := public.Group("/search")
		{
			search.GET("/hotkeys", h.GetHotKeys)
			search.GET("/songs", h.SearchSongs)
			search.GET("/singers", h.SearchSingers)
			search.GET("/albums", h.SearchAlbums)
			search.GET("/mvs", h.SearchMVs)
		}
	}

	// ===== 需要认证的API =====
	authenticated := router.Group("/api")
	authenticated.Use(middleware.RequiredAuth(jwtSecret, log))
	{
		// 每日推荐（需要登录）
		authenticated.GET("/home/daily-recommend", h.GetDailyRecommendPlaylists)

		// 用户相关接口
		if userHandler != nil {
			user := authenticated.Group("/user")
			{
				// 收藏管理
				user.POST("/favorites", userHandler.AddFavorite)
				user.DELETE("/favorites/:song_id", userHandler.RemoveFavorite)
				user.GET("/favorites", userHandler.ListFavorites)

				// 播放历史
				user.POST("/history", userHandler.AddPlayHistory)
				user.GET("/history", userHandler.ListPlayHistory)

				// 歌单管理
				user.POST("/playlists", userHandler.CreatePlaylist)
				user.PUT("/playlists/:playlist_id", userHandler.UpdatePlaylist)
				user.DELETE("/playlists/:playlist_id", userHandler.DeletePlaylist)
				user.GET("/playlists", userHandler.ListPlaylists)

				// 歌单歌曲管理
				user.POST("/playlists/:playlist_id/songs", userHandler.AddSongToPlaylist)
				user.DELETE("/playlists/:playlist_id/songs/:song_id", userHandler.RemoveSongFromPlaylist)
				user.GET("/playlists/:playlist_id/songs", userHandler.GetPlaylistSongs)
			}
		}
	}

	return router
}

// warmUpHotData 预热热点数据
func warmUpHotData(ctx context.Context, warmupService *cache.WarmUpService) error {
	// TODO: 从上游API获取真实数据进行预热
	// 这里使用模拟数据作为示例

	// 预热Banner
	banners := []interface{}{
		map[string]string{"id": "1", "image": "banner1.jpg", "link": "/playlist/hot"},
		map[string]string{"id": "2", "image": "banner2.jpg", "link": "/playlist/new"},
	}
	if err := warmupService.WarmUpBanner(ctx, banners); err != nil {
		return fmt.Errorf("failed to warm up banners: %w", err)
	}

	// 预热热门歌单
	playlists := []interface{}{
		map[string]string{"id": "100", "name": "热门歌单"},
		map[string]string{"id": "200", "name": "新歌推荐"},
	}
	if err := warmupService.WarmUpHotPlaylists(ctx, playlists); err != nil {
		return fmt.Errorf("failed to warm up playlists: %w", err)
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

// getServiceAddr 获取服务地址（优先使用Consul服务发现）
func getServiceAddr(discovery *consul.ServiceDiscovery, serviceName, fallbackAddr string, log logger.Logger) string {
	if discovery != nil {
		addr, err := discovery.GetServiceAddress(serviceName)
		if err == nil {
			log.Info("Service address discovered from Consul",
				logger.String("service", serviceName),
				logger.String("address", addr),
			)
			return addr
		}

		log.Warn("Failed to discover service from Consul, using fallback address",
			logger.String("service", serviceName),
			logger.String("fallback", fallbackAddr),
			logger.String("error", err.Error()),
		)
	}

	return fallbackAddr
}

// watchServiceChanges 监听服务地址变化并更新gRPC客户端
func watchServiceChanges(
	discovery *consul.ServiceDiscovery,
	serviceName string,
	pool *client.ClientPool,
	clientPtr interface{},
	log logger.Logger,
) {
	discovery.Watch(serviceName, 30*time.Second, func(addresses []string) {
		if len(addresses) == 0 {
			log.Warn("No healthy instances found for service",
				logger.String("service", serviceName),
			)
			return
		}

		// 使用第一个可用地址
		newAddr := addresses[0]

		log.Info("Service address changed, reconnecting",
			logger.String("service", serviceName),
			logger.String("new_address", newAddr),
		)

		// 关闭旧连接以触发重连到新地址
		// gRPC客户端在下次调用时会自动重连
		pool.CloseConnection(serviceName)
	})
}

// getLocalIP 获取本机IP地址
func getLocalIP() string {
	// 尝试从环境变量获取
	if ip := os.Getenv("SERVICE_IP"); ip != "" {
		return ip
	}

	// 返回localhost作为默认值（生产环境应该配置SERVICE_IP）
	return "127.0.0.1"
}
