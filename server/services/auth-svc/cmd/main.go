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
	"github.com/listen-stream/server/shared/pkg/grpc"
	"github.com/listen-stream/server/shared/pkg/logger"
	authv1 "github.com/listen-stream/server/shared/proto/auth/v1"
	authgrpc "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/grpc"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/handler"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/middleware"
)

const (
	serviceName = "auth-svc"
	httpPort    = 8001
	grpcPort    = 9002
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

	// TODO: Initialize database connection
	// db, err := db.NewPostgresDB(cfg.Database)
	// if err != nil {
	//     log.Fatal("Failed to connect to database", logger.String("error", err.Error()))
	// }
	// defer db.Close()

	// TODO: Initialize Redis connection
	// redisClient := redis.NewClient(cfg.Redis)
	// defer redisClient.Close()

	// TODO: Initialize services
	// userRepo := repository.NewUserRepository(db)
	// smsService := smsservice.New(redisClient, cfg.SMS)
	// jwtService := jwtservice.New(cfg.JWT, redisClient)
	// deviceService := deviceservice.New(repository.NewDeviceRepository(db), redisClient)

	// Initialize handlers (placeholder - will be wired after services are ready)
	// loginHandler := handler.NewLoginHandler(smsService, jwtService, deviceService, userRepo)
	// deviceHandler := handler.NewDeviceHandler(deviceService, jwtService)

	// Initialize gRPC server implementation
	// authServer := authgrpc.NewAuthServer(jwtService, deviceService)

	// Mark unused imports temporarily
	_ = handler.NewLoginHandler
	_ = authgrpc.NewAuthServer

	// Start HTTP server
	httpServer := startHTTPServer(log, httpPort, nil, nil)

	// Start gRPC server
	grpcServer, err := startGRPCServer(log, grpcPort, nil)
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

	// TODO: Close database and Redis connections
	log.Info("Auth service stopped")
}

// startHTTPServer starts the HTTP server for client-facing APIs
func startHTTPServer(log logger.Logger, port int, loginHandler *handler.LoginHandler, deviceHandler *handler.DeviceHandler) *http.Server {
	// Create Gin router
	router := gin.New()

	// Apply middleware stack (order matters!)
	router.Use(middleware.RequestID())       // 1. Inject request ID
	router.Use(middleware.Recovery(log))     // 2. Panic recovery
	router.Use(middleware.Logging(log))      // 3. Request logging
	router.Use(middleware.CORS())            // 4. CORS headers
	router.Use(middleware.SecurityHeaders()) // 5. Security headers

	// Health check endpoint
	router.GET("/health", healthCheckHandler(log))

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

// healthCheckHandler returns HTTP health check handler
func healthCheckHandler(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Check database connectivity
		// TODO: Check Redis connectivity

		status := gin.H{
			"status":    "healthy",
			"service":   serviceName,
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
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
