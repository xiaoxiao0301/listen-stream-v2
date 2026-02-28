package main

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/proxy-svc/internal/client"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/proxy-svc/internal/upstream"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// HealthChecker 健康检查器
type HealthChecker struct {
	redisClient      *redis.Client
	authClient       *client.AuthClient
	userClient       *client.UserClient
	upstreamClients  []upstream.ClientInterface
	upstreamNames    []string
	log              logger.Logger
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(
	redisClient *redis.Client,
	authClient *client.AuthClient,
	userClient *client.UserClient,
	upstreamClients []upstream.ClientInterface,
	upstreamNames []string,
	log logger.Logger,
) *HealthChecker {
	return &HealthChecker{
		redisClient:     redisClient,
		authClient:      authClient,
		userClient:      userClient,
		upstreamClients: upstreamClients,
		upstreamNames:   upstreamNames,
		log:             log,
	}
}

// HealthCheckResponse 健康检查响应
type HealthCheckResponse struct {
	Status       string                       `json:"status"`       // healthy, degraded, unhealthy
	Service      string                       `json:"service"`
	Version      string                       `json:"version"`
	Timestamp    int64                        `json:"timestamp"`
	Dependencies map[string]DependencyStatus  `json:"dependencies"`
	Upstreams    map[string]UpstreamStatus    `json:"upstreams"`
}

// DependencyStatus 依赖状态
type DependencyStatus struct {
	Status  string `json:"status"`  // up, down
	Latency int64  `json:"latency"` // ms
	Error   string `json:"error,omitempty"`
}

// UpstreamStatus 上游服务状态
type UpstreamStatus struct {
	Status string `json:"status"` // up, down
	Error  string `json:"error,omitempty"`
}

// Check 执行健康检查
func (hc *HealthChecker) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response := HealthCheckResponse{
		Status:       "healthy",
		Service:      "proxy-svc",
		Version:      "1.0.0",
		Timestamp:    time.Now().Unix(),
		Dependencies: make(map[string]DependencyStatus),
		Upstreams:    make(map[string]UpstreamStatus),
	}

	healthyCount := 0
	totalCount := 0

	// 检查Redis
	redisStatus := hc.checkRedis(ctx)
	response.Dependencies["redis"] = redisStatus
	totalCount++
	if redisStatus.Status == "up" {
		healthyCount++
	}

	// 检查auth-svc
	if hc.authClient != nil {
		authStatus := hc.checkGRPCService(ctx, "auth-svc", hc.authClient)
		response.Dependencies["auth-svc"] = authStatus
		totalCount++
		if authStatus.Status == "up" {
			healthyCount++
		}
	}

	// 检查user-svc
	if hc.userClient != nil {
		userStatus := hc.checkGRPCService(ctx, "user-svc", hc.userClient)
		response.Dependencies["user-svc"] = userStatus
		totalCount++
		if userStatus.Status == "up" {
			healthyCount++
		}
	}

	// 检查上游服务
	upstreamHealthyCount := 0
	for i, upstreamClient := range hc.upstreamClients {
		name := hc.upstreamNames[i]
		status := hc.checkUpstream(ctx, name, upstreamClient)
		response.Upstreams[name] = status
		if status.Status == "up" {
			upstreamHealthyCount++
		}
	}

	// 计算整体健康状态
	if healthyCount == totalCount && upstreamHealthyCount > 0 {
		response.Status = "healthy"
		c.JSON(200, response)
	} else if healthyCount > 0 || upstreamHealthyCount > 0 {
		response.Status = "degraded"
		c.JSON(200, response)
	} else {
		response.Status = "unhealthy"
		c.JSON(503, response)
	}
}

// checkRedis 检查Redis连接
func (hc *HealthChecker) checkRedis(ctx context.Context) DependencyStatus {
	start := time.Now()
	
	if hc.redisClient == nil {
		return DependencyStatus{
			Status: "down",
			Error:  "redis client not initialized",
		}
	}

	err := hc.redisClient.Ping(ctx).Err()
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return DependencyStatus{
			Status:  "down",
			Latency: latency,
			Error:   err.Error(),
		}
	}

	return DependencyStatus{
		Status:  "up",
		Latency: latency,
	}
}

// checkGRPCService 检查gRPC服务
func (hc *HealthChecker) checkGRPCService(ctx context.Context, name string, clientInterface interface{}) DependencyStatus {
	start := time.Now()
	
	// TODO: 实现实际的gRPC健康检查
	// 这里可以调用标准的grpc.health.v1.Health/Check
	// 暂时返回基本状态
	
	latency := time.Since(start).Milliseconds()
	
	return DependencyStatus{
		Status:  "up",
		Latency: latency,
	}
}

// checkUpstream 检查上游服务
func (hc *HealthChecker) checkUpstream(ctx context.Context, name string, client upstream.ClientInterface) UpstreamStatus {
	if client == nil {
		return UpstreamStatus{
			Status: "down",
			Error:  "client not initialized",
		}
	}

	err := client.HealthCheck(ctx)
	if err != nil {
		return UpstreamStatus{
			Status: "down",
			Error:  err.Error(),
		}
	}

	return UpstreamStatus{
		Status: "up",
	}
}
