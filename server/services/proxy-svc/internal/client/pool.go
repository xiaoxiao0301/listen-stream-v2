package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// ClientPool gRPC客户端连接池
type ClientPool struct {
	connections map[string]*grpc.ClientConn
	mu          sync.RWMutex
	log         logger.Logger
}

// NewClientPool 创建连接池
func NewClientPool(log logger.Logger) *ClientPool {
	return &ClientPool{
		connections: make(map[string]*grpc.ClientConn),
		log:         log,
	}
}

// GetConnection 获取或创建连接
func (p *ClientPool) GetConnection(ctx context.Context, service, address string) (*grpc.ClientConn, error) {
	// 先尝试读锁
	p.mu.RLock()
	conn, exists := p.connections[service]
	p.mu.RUnlock()

	if exists {
		// 检查连接状态
		state := conn.GetState()
		if state == connectivity.Ready || state == connectivity.Idle {
			return conn, nil
		}

		// 连接已断开，需要重新创建
		p.log.WithFields(
			logger.String("service", service),
			logger.String("state", state.String()),
		).Warn("Connection is not ready, reconnecting")
	}

	// 加写锁创建新连接
	p.mu.Lock()
	defer p.mu.Unlock()

	// 双重检查
	conn, exists = p.connections[service]
	if exists {
		state := conn.GetState()
		if state == connectivity.Ready || state == connectivity.Idle {
			return conn, nil
		}

		// 关闭旧连接
		_ = conn.Close()
	}

	// 创建新连接
	conn, err := p.createConnection(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection to %s: %w", service, err)
	}

	p.connections[service] = conn

	p.log.WithFields(
		logger.String("service", service),
		logger.String("address", address),
	).Info("gRPC connection established")

	return conn, nil
}

// createConnection 创建gRPC连接
func (p *ClientPool) createConnection(ctx context.Context, address string) (*grpc.ClientConn, error) {
	// 配置Keep-Alive
	kacp := keepalive.ClientParameters{
		Time:                10 * time.Second, // 每10秒发送keep-alive ping
		Timeout:             3 * time.Second,  // 3秒超时
		PermitWithoutStream: true,             // 即使没有活动流也发送ping
	}

	// 创建连接（添加超时）
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // 生产环境应该使用TLS
		grpc.WithKeepaliveParams(kacp),
		grpc.WithBlock(),                       // 阻塞等待连接建立
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`), // 负载均衡
	)

	if err != nil {
		return nil, err
	}

	return conn, nil
}

// Close 关闭所有连接
func (p *ClientPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errors []error
	for service, conn := range p.connections {
		if err := conn.Close(); err != nil {
			p.log.WithFields(
				logger.String("service", service),
				logger.String("error", err.Error()),
			).Error("Failed to close connection")
			errors = append(errors, err)
		}
	}

	p.connections = make(map[string]*grpc.ClientConn)

	if len(errors) > 0 {
		return fmt.Errorf("failed to close %d connections", len(errors))
	}

	return nil
}

// CloseConnection 关闭指定服务的连接
func (p *ClientPool) CloseConnection(service string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	conn, exists := p.connections[service]
	if !exists {
		return nil // 连接不存在，直接返回
	}

	if err := conn.Close(); err != nil {
		p.log.WithFields(
			logger.String("service", service),
			logger.String("error", err.Error()),
		).Error("Failed to close connection")
		return err
	}

	delete(p.connections, service)

	p.log.WithFields(
		logger.String("service", service),
	).Info("Connection closed")

	return nil
}

// HealthCheck 健康检查
func (p *ClientPool) HealthCheck(ctx context.Context) map[string]bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]bool)
	for service, conn := range p.connections {
		state := conn.GetState()
		result[service] = state == connectivity.Ready || state == connectivity.Idle
	}

	return result
}
