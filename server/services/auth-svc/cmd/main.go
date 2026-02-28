package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	authgrpc "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/grpc"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/handler"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/middleware"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/repository"
	deviceservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/device"
	jwtservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/jwt"
	smsservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/sms"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/consul"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/db"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/grpc"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
	authv1 "github.com/xiaoxiao0301/listen-stream-v2/server/shared/proto/auth/v1"
)

const (
	serviceName = "auth-svc"
	httpPort    = 8001
	grpcPort    = 9001
)

func main() {
	// Initialize logger
	log := logger.Default()
	log.Info("Starting auth-svc...",
		logger.String("http_port", fmt.Sprintf(":%d", httpPort)),
		logger.String("grpc_port", fmt.Sprintf(":%d", grpcPort)))

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection
	dbConfig := db.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", ""),
		Database: getEnv("DB_NAME", "auth_db"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
		MaxConns: getEnvInt("DB_MAX_CONNS", 25),
		MinConns: getEnvInt("DB_MIN_CONNS", 5),
	}

	database, err := db.NewPostgresDB(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database", logger.String("error", err.Error()))
	}
	defer database.Close()
	log.Info("Connected to PostgreSQL database")

	// Initialize Redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       getEnvInt("REDIS_DB", 0),
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("Failed to connect to Redis", logger.String("error", err.Error()))
	}
	defer redisClient.Close()
	log.Info("Connected to Redis")

	// Initialize repositories
	userRepo := repository.NewUserRepository(database)
	deviceRepo := repository.NewDeviceRepository(database)
	smsRepo := repository.NewSMSRepository(database)

	// Initialize SMS service with Fallback chain
	smsProviders := []smsservice.Provider{
		smsservice.NewAliyunProvider(smsservice.AliyunConfig{
			AccessKeyID:     getEnv("ALIYUN_ACCESS_KEY_ID", ""),
			AccessKeySecret: getEnv("ALIYUN_ACCESS_KEY_SECRET", ""),
			SignName:        getEnv("ALIYUN_SMS_SIGN_NAME", ""),
			TemplateCode:    getEnv("ALIYUN_SMS_TEMPLATE_CODE", ""),
		}),
		smsservice.NewTencentProvider(smsservice.TencentConfig{
			SecretID:     getEnv("TENCENT_SECRET_ID", ""),
			SecretKey:    getEnv("TENCENT_SECRET_KEY", ""),
			SDKAppID:     getEnv("TENCENT_SMS_SDK_APP_ID", ""),
			SignName:     getEnv("TENCENT_SMS_SIGN_NAME", ""),
			TemplateID:   getEnv("TENCENT_SMS_TEMPLATE_ID", ""),
		}),
		smsservice.NewTwilioProvider(smsservice.TwilioConfig{
			AccountSID: getEnv("TWILIO_ACCOUNT_SID", ""),
			AuthToken:  getEnv("TWILIO_AUTH_TOKEN", ""),
			FromNumber: getEnv("TWILIO_FROM_NUMBER", ""),
		}),
	}

	smsServiceInstance := smsservice.NewService(smsProviders, smsRepo, redisClient, log)

	// Initialize JWT service
	jwtServiceInstance := jwtservice.NewService(jwtservice.Config{
		Secret:           getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		AccessTokenTTL:   time.Duration(getEnvInt("JWT_ACCESS_TTL_HOURS", 24)) * time.Hour,
		RefreshTokenTTL:  time.Duration(getEnvInt("JWT_REFRESH_TTL_DAYS", 30)) * 24 * time.Hour,
		EnableIPBinding:  getEnvBool("JWT_ENABLE_IP_BINDING", false),
	}, redisClient, log)

	// Initialize device service
	deviceServiceInstance := deviceservice.NewService(deviceRepo, redisClient, log)

	// Initialize handlers
	loginHandler := handler.NewLoginHandler(smsServiceInstance, jwtServiceInstance, deviceServiceInstance, userRepo, log)
	deviceHandler := handler.NewDeviceHandler(deviceServiceInstance, jwtServiceInstance, log)

	// Initialize gRPC server implementation
	authServer := authgrpc.NewAuthServer(jwtServiceInstance, deviceServiceInstance, userRepo, log)

	// Register service to Consul (if enabled)
	var registry *consul.ServiceRegistry
	if getEnvBool("ENABLE_CONSUL", false) {
		serviceAddr := getEnv("SERVICE_ADDR", getLocalIP()+fmt.Sprintf(":%d", httpPort))
		registryConfig := consul.RegistryConfig{
			ConsulAddr:  getEnv("CONSUL_ADDR", "localhost:8500"),
			ServiceName: serviceName,
			ServiceAddr: serviceAddr,
			ServiceTags: []string{"auth", "grpc", "http"},
			HealthCheck: consul.HealthCheckConfig{
				HTTP:                           fmt.Sprintf("http://%s/health", serviceAddr),
				Interval:                       10 * time.Second,
				Timeout:                        5 * time.Second,
				DeregisterCriticalServiceAfter: 30 * time.Second,
			},
		}

		registry, err = consul.NewServiceRegistry(registryConfig, log)
		if err != nil {
			log.Warn("Failed to register service to Consul", logger.String("error", err.Error()))
		} else {
			log.Info("Service registered to Consul", logger.String("service", serviceName))
		}
	}

	// Start HTTP server
	httpServer := startHTTPServer(log, httpPort, loginHandler, deviceHandler, database, redisClient)

	// Start gRPC server
	grpcServer, err := startGRPCServer(log, grpcPort, authServer)
	if err != nil {
		log.Fatal("Failed to start gRPC server", logger.String("error", err.Error()))
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	log.Info("Received shutdown signal", logger.String("signal", sig.String()))

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	// Deregister from Consul
	if registry != nil {
		if err := registry.Deregister(shutdownCtx); err != nil {
			log.Error("Failed to deregister from Consul", logger.String("error", err.Error()))
		}
	}

	// Shutdown HTTP server
	log.Info("Shutting down HTTP server...")
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", logger.String("error", err.Error()))
	}

	// Shutdown gRPC server
	log.Info("Shutting down gRPC server...")
	if err := grpcServer.Shutdown(shutdownCtx); err != nil {
		log.Error("gRPC server shutdown error", logger.String("error", err.Error()))
	}

	log.Info("Auth service stopped gracefully")
}

// startHTTPServer starts the HTTP server for client-facing APIs
func startHTTPServer(log logger.Logger, port int, loginHandler *handler.LoginHandler, deviceHandler *handler.DeviceHandler, database *sql.DB, redisClient *redis.Client) *http.Server {
	// Create Gin router
	router := gin.New()

	// Apply middleware stack (order matters!)
	router.Use(middleware.RequestID())       // 1. Inject request ID
	router.Use(middleware.Recovery(log))     // 2. Panic recovery
	router.Use(middleware.Logging(log))      // 3. Request logging
	router.Use(middleware.CORS())            // 4. CORS headers
	router.Use(middleware.SecurityHeaders()) // 5. Security headers

	// Health check endpoint
	router.GET("/health", healthCheckHandler(log, database, redisClient))

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Auth endpoints
		auth := v1.Group("/auth")
		{
			if loginHandler != nil {
				auth.POST("/send-code", wrapHandler(loginHandler.SendVerificationCode))
				auth.POST("/verify-login", wrapHandler(loginHandler.VerifyLogin))
			} else {
				// Placeholder endpoints
				auth.POST("/send-code", func(c *gin.Context) {
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": "service not initialized"})
				})
				auth.POST("/verify-login", func(c *gin.Context) {
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": "service not initialized"})
				})
			}
		}

		// Device endpoints
		devices := v1.Group("/devices")
		{
			if deviceHandler != nil {
				devices.GET("", wrapHandler(deviceHandler.ListDevices))
				devices.DELETE("/:device_id", wrapHandler(deviceHandler.RemoveDevice))
			} else {
				// Placeholder endpoints
				devices.GET("", func(c *gin.Context) {
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": "service not initialized"})
				})
				devices.DELETE("/:device_id", func(c *gin.Context) {
					c.JSON(http.StatusServiceUnavailable, gin.H{"error": "service not initialized"})
				})
			}
		}
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info("HTTP server listening", logger.Int("port", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed", logger.String("error", err.Error()))
		}
	}()

	return server
}

// startGRPCServer starts the gRPC server for internal service communication
func startGRPCServer(log logger.Logger, port int, authServer *authgrpc.AuthServer) (*grpc.Server, error) {
	// Create gRPC server config
	config := grpc.DefaultServerConfig(serviceName, port)
	config.EnableReflection = true // Enable for development

	// Create gRPC server
	server, err := grpc.NewServer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC server: %w", err)
	}

	// Register gRPC services
	if authServer != nil {
		authv1.RegisterAuthServiceServer(server.Server, authServer)
		log.Info("Registered AuthService to gRPC server")
	} else {
		log.Warn("AuthService not initialized - gRPC endpoints unavailable")
	}

	// Start server in goroutine
	go func() {
		log.Info("gRPC server listening", logger.Int("port", port))
		if err := server.Serve(); err != nil {
			log.Fatal("gRPC server failed", logger.String("error", err.Error()))
		}
	}()

	return server, nil
}

// healthCheckHandler returns HTTP health check handler with dependency checks
func healthCheckHandler(log logger.Logger, database *sql.DB, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		
		status := gin.H{
			"status":    "healthy",
			"service":   serviceName,
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		}

		// Check database connectivity
		if database != nil {
			if err := database.PingContext(ctx); err != nil {
				status["status"] = "unhealthy"
				status["database"] = "unreachable"
				log.Error("Database health check failed", logger.String("error", err.Error()))
				c.JSON(http.StatusServiceUnavailable, status)
				return
			}
			status["database"] = "connected"
		}

		// Check Redis connectivity
		if redisClient != nil {
			if err := redisClient.Ping(ctx).Err(); err != nil {
				status["status"] = "unhealthy"
				status["redis"] = "unreachable"
				log.Error("Redis health check failed", logger.String("error", err.Error()))
				c.JSON(http.StatusServiceUnavailable, status)
				return
			}
			status["redis"] = "connected"
		}

		c.JSON(http.StatusOK, status)
	}
}

// wrapHandler wraps http.HandlerFunc to gin.HandlerFunc
// This allows us to use standard net/http handlers with Gin
func wrapHandler(h func(http.ResponseWriter, *http.Request)) gin.HandlerFunc {
	return func(c *gin.Context) {
		h(c.Writer, c.Request)
	}
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt gets integer environment variable with fallback
func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return fallback
}

// getEnvBool gets boolean environment variable with fallback
func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return fallback
}

// getLocalIP gets local IP address for service registration
func getLocalIP() string {
	if ip := os.Getenv("SERVICE_IP"); ip != "" {
		return ip
	}
	return "127.0.0.1"
}
