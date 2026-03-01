package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"user-svc/internal/cron"
	"user-svc/internal/grpc"
	"user-svc/internal/handler"
	"user-svc/internal/middleware"
	"user-svc/internal/repository"
	"user-svc/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	userv1 "github.com/listen-stream/server/shared/proto/user/v1"
	grpc_server "google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
httpPort = ":8003"
grpcPort = ":9003"
)

func main() {
	log.Println("Starting user-svc...")

	db, err := initDatabase()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	favoriteService, historyService, playlistService, cleanupService := initServices(db)

	cronManager := cron.NewCronManager(cleanupService)
	if err := cronManager.Start(); err != nil {
		log.Fatalf("Failed to start cron manager: %v", err)
	}
	defer cronManager.Stop()

	httpServer := startHTTPServer(favoriteService, historyService, playlistService)
	grpcServer := startGRPCServer(favoriteService, historyService, playlistService)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down user-svc...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server forced to shutdown: %v", err)
	}

	grpcServer.GracefulStop()
	log.Println("user-svc stopped")
}

func initDatabase() (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/listen_stream?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connected successfully")
	return pool, nil
}

func initServices(db *pgxpool.Pool) (*service.FavoriteService, *service.PlayHistoryService, *service.PlaylistService, *service.CleanupService) {
	// 初始化仓储层
	favoriteRepo := repository.NewFavoriteRepository(db)
	historyRepo := repository.NewPlayHistoryRepository(db)
	playlistRepo := repository.NewPlaylistRepository(db)
	playlistSongRepo := repository.NewPlaylistSongRepository(db)

	// 初始化服务层
	favoriteService := service.NewFavoriteService(favoriteRepo)
	historyService := service.NewPlayHistoryService(historyRepo)
	playlistService := service.NewPlaylistService(playlistRepo, playlistSongRepo)
	cleanupService := service.NewCleanupService(historyRepo)

	return favoriteService, historyService, playlistService, cleanupService
}

func startHTTPServer(
favoriteService *service.FavoriteService,
historyService *service.PlayHistoryService,
playlistService *service.PlaylistService,
) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())

	router.GET("/health", func(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{"status": "ok"})
})

	api := router.Group("/api/v1")
	api.Use(middleware.JWTAuth())
	{
		favoriteHandler := handler.NewFavoriteHandler(favoriteService)
		api.POST("/favorites", favoriteHandler.AddFavorite)
		api.GET("/favorites", favoriteHandler.ListFavorites)
		api.GET("/favorites/check", favoriteHandler.CheckFavorite)
		api.DELETE("/favorites/:id", favoriteHandler.RemoveFavorite)

		historyHandler := handler.NewPlayHistoryHandler(historyService)
		api.POST("/history", historyHandler.AddPlayHistory)
		api.GET("/history", historyHandler.ListPlayHistories)
		api.DELETE("/history/:id", historyHandler.DeletePlayHistory)

		playlistHandler := handler.NewPlaylistHandler(playlistService)
		api.POST("/playlists", playlistHandler.CreatePlaylist)
		api.GET("/playlists", playlistHandler.ListUserPlaylists)
		api.GET("/playlists/:id", playlistHandler.GetPlaylist)
		api.PUT("/playlists/:id", playlistHandler.UpdatePlaylist)
		api.DELETE("/playlists/:id", playlistHandler.DeletePlaylist)
		api.POST("/playlists/:id/songs", playlistHandler.AddSongToPlaylist)
		api.GET("/playlists/:id/songs", playlistHandler.ListPlaylistSongs)
		api.DELETE("/playlists/:id/songs/:song_id", playlistHandler.RemoveSongFromPlaylist)
	}

	server := &http.Server{
		Addr:    httpPort,
		Handler: router,
	}

	go func() {
		log.Printf("HTTP server listening on %s", httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	return server
}

func startGRPCServer(
favoriteService *service.FavoriteService,
historyService *service.PlayHistoryService,
playlistService *service.PlaylistService,
) *grpc_server.Server {
	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", grpcPort, err)
	}

	grpcServer := grpc_server.NewServer()

	userServer := grpc.NewUserServer(favoriteService, historyService, playlistService)
	userv1.RegisterUserServiceServer(grpcServer, userServer)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	go func() {
		log.Printf("gRPC server listening on %s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	return grpcServer
}
