package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"admin-svc/internal/handler"
	"admin-svc/internal/middleware"
	"admin-svc/internal/service"

	"github.com/gin-gonic/gin"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"
)

func main() {
	// 初始化Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})
	defer redisClient.Close()

	// 测试Redis连接
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	log.Println("Connected to Redis")

	// 初始化Consul
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = getEnv("CONSUL_ADDR", "localhost:8500")
	consulClient, err := consulapi.NewClient(consulConfig)
	if err != nil {
		log.Fatalf("failed to create consul client: %v", err)
	}
	log.Println("Connected to Consul")

	// 初始化服务
	totpSvc := service.NewTOTPService("Listen Stream Admin")
	configSvc := service.NewConfigService(consulClient, redisClient, "listen-stream/")
	auditSvc := service.NewAuditService(redisClient)
	statsSvc := service.NewStatsService(redisClient)
	exportSvc := service.NewExportService()

	// 初始化处理器
	adminHandler := handler.NewAdminHandler(totpSvc, auditSvc)
	configHandler := handler.NewConfigHandler(configSvc, auditSvc)
	statsHandler := handler.NewStatsHandler(statsSvc, exportSvc)
	auditHandler := handler.NewAuditHandler(auditSvc, exportSvc)

	// 创建Gin路由
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS())

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "admin-svc",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// API路由（需要认证）
	api := router.Group("/api/v1")
	api.Use(middleware.JWTAuth())
	{
		// 管理员管理
		admins := api.Group("/admins")
		{
			admins.POST("/login", adminHandler.Login)
			admins.GET("", adminHandler.ListAdmins)
			admins.POST("/2fa/enable", adminHandler.Enable2FA)
			admins.POST("/2fa/verify", adminHandler.Verify2FA)
			admins.POST("/2fa/disable", adminHandler.Disable2FA)
		}

		// 配置管理
		configs := api.Group("/configs")
		{
			configs.GET("", configHandler.ListConfigs)
			configs.GET("/:key", configHandler.GetConfig)
			configs.PUT("/:key", configHandler.UpdateConfig)
			configs.DELETE("/:key", configHandler.DeleteConfig)
			configs.POST("/cache/clear", configHandler.ClearConfigCache)
			configs.GET("/:key/history", configHandler.GetConfigHistory)
		}

		// 统计
		stats := api.Group("/stats")
		{
			stats.GET("/realtime", statsHandler.GetRealtimeStats)
			stats.GET("/daily", statsHandler.GetDailyStats)
			stats.GET("/daily/export", statsHandler.ExportDailyStats)
		}

		// 审计日志
		audit := api.Group("/audit")
		{
			audit.GET("/logs", auditHandler.ListOperationLogs)
			audit.GET("/logs/export", auditHandler.ExportOperationLogs)
			audit.GET("/anomalies", auditHandler.ListAnomalousActivities)
			audit.POST("/anomalies/:id/resolve", auditHandler.ResolveAnomalousActivity)
		}
	}

	// 启动HTTP服务器
	httpPort := getEnv("HTTP_PORT", "8005")
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", httpPort),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 服务注册到Consul
	instanceID := fmt.Sprintf("admin-svc-%s", getHostname())
	registration := &consulapi.AgentServiceRegistration{
		ID:      instanceID,
		Name:    "admin-svc",
		Port:    mustParseInt(httpPort),
		Address: getLocalIP(),
		Check: &consulapi.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%s/health", getLocalIP(), httpPort),
			Interval:                       "10s",
			Timeout:                        "3s",
			DeregisterCriticalServiceAfter: "30s",
		},
	}

	if err := consulClient.Agent().ServiceRegister(registration); err != nil {
		log.Printf("Warning: failed to register to consul: %v", err)
	} else {
		log.Printf("Registered to Consul as %s", instanceID)
		// 注销服务
		defer func() {
			if err := consulClient.Agent().ServiceDeregister(instanceID); err != nil {
				log.Printf("Warning: failed to deregister from consul: %v", err)
			}
		}()
	}

	// 优雅关闭
	go func() {
		log.Printf("admin-svc HTTP server started on :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 5秒超时关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getHostname() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	return hostname
}

func getLocalIP() string {
	// 简化实现，实际应该获取真实IP
	return "127.0.0.1"
}

func mustParseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}
