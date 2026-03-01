package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sync-svc/internal/handler"
	"sync-svc/internal/middleware"
	"sync-svc/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	defaultPort           = "8004"
	defaultMaxConnections = 10000
	defaultJWTSecret      = "your-secret-key-change-in-production"
	defaultRedisAddr      = "localhost:6379"
	defaultInstanceID     = "sync-svc-1"
)

func main() {
	log.Println("Starting sync-svc...")

	// 从环境变量获取配置
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = defaultJWTSecret
		log.Println("Warning: Using default JWT secret, please set JWT_SECRET in production")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = defaultRedisAddr
	}

	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = defaultInstanceID
		log.Printf("Using default instance ID: %s", instanceID)
	} else {
		log.Printf("Instance ID: %s", instanceID)
	}

	// 创建Redis客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// 测试Redis连接
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")

	// 创建WebSocket管理器
	wsManager := ws.NewManager(defaultMaxConnections, redisClient, instanceID)

	// 启动管理器
	managerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go wsManager.Start(managerCtx)

	// 创建HTTP服务器
	server := startHTTPServer(port, jwtSecret, wsManager)

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down sync-svc...")

	// 取消context，触发manager关闭
	cancel()

	// 关闭Redis连接
	if err := redisClient.Close(); err != nil {
		log.Printf("Failed to close Redis connection: %v", err)
	}

	// 优雅关闭HTTP服务器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("sync-svc stopped")
}

func startHTTPServer(port, jwtSecret string, wsManager *ws.Manager) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New() // 使用gin.New()而不是gin.Default()

	// 全局中间件
	router.Use(gin.Recovery())                    // 恢复panic
	router.Use(handler.RequestLogger())          // 请求日志
	router.Use(middleware.CORS())                // CORS

	// 创建速率限制器（每个客户端每分钟最多100个请求）
	rateLimiter := handler.NewRateLimiter(100, time.Minute)

	// 健康检查（不需要限流）
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"instance_id": wsManager.GetInstanceID(),
			"timestamp":   time.Now(),
		})
	})

	// WebSocket处理器
	wsHandler := handler.NewWSHandler(wsManager)
	
	// 事件处理器
	eventHandler := handler.NewEventHandler(wsManager)

	// WebSocket路由（需要JWT认证，每个用户每分钟最多10次连接尝试）
	wsRateLimiter := handler.NewRateLimiter(10, time.Minute)
	router.GET("/ws", 
		middleware.JWTAuth(jwtSecret),
		handler.RateLimitMiddleware(wsRateLimiter),
		wsHandler.HandleWebSocket)

	// 管理API（无需认证，但有限流）
	api := router.Group("/api/v1")
	api.Use(handler.RateLimitMiddleware(rateLimiter))
	{
		// 统计信息（内部调用）
		api.GET("/stats", wsHandler.GetStats)
		api.GET("/online-users", wsHandler.GetOnlineUsers)
		api.GET("/users/:user_id/online", wsHandler.CheckUserOnline)

		// 离线消息统计
		api.GET("/offline/stats", wsHandler.GetOfflineStats)
		
		// Pub/Sub统计
		api.GET("/stats/pubsub", eventHandler.GetPubSubStats)
	}
	
	// 事件API（需要JWT认证 + 限流 + 请求验证）
	eventAPI := router.Group("/api/v1/events")
	eventAPI.Use(
		middleware.JWTAuth(jwtSecret),
		handler.RateLimitMiddleware(rateLimiter),
		handler.ValidateEventRequest,
	)
	{
		eventAPI.POST("", eventHandler.PublishEvent)              // 发布单用户事件
		eventAPI.POST("/batch", eventHandler.BatchPublishEvent)   // 批量发布事件
		eventAPI.POST("/broadcast", eventHandler.BroadcastEvent)  // 全局广播
	}

	// 离线消息API（需要JWT认证 + 限流）
	offlineAPI := router.Group("/api/v1/offline")
	offlineAPI.Use(
		middleware.JWTAuth(jwtSecret),
		handler.RateLimitMiddleware(rateLimiter),
	)
	{
		offlineAPI.GET("/messages", wsHandler.GetOfflineMessages)           // 拉取离线消息
		offlineAPI.POST("/ack", wsHandler.AckOfflineMessage)                // 确认单条消息
		offlineAPI.POST("/ack/batch", wsHandler.BatchAckOfflineMessages)    // 批量确认消息
		offlineAPI.GET("/count", wsHandler.GetOfflineMessageCount)          // 获取消息数量
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("HTTP server listening on :%s", port)
		log.Printf("WebSocket endpoint: ws://localhost:%s/ws", port)
		log.Printf("Health check: http://localhost:%s/health", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	return server
}
