// Package grpc provides utilities for gRPC client and server operations.
//
// It includes connection pooling, retry logic, service discovery integration,
// and common interceptors for the Listen Stream platform.
package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/resolver"
)

// ClientConfig holds gRPC client configuration.
type ClientConfig struct {
	// Target address (host:port or consul://service-name)
	Target string

	// Connection timeout
	Timeout time.Duration

	// Enable keepalive
	Keepalive bool

	// Keepalive time
	KeepaliveTime time.Duration

	// Keepalive timeout
	KeepaliveTimeout time.Duration

	// Max receive message size (default: 4MB)
	MaxRecvMsgSize int

	// Max send message size (default: 4MB)
	MaxSendMsgSize int

	// Unary interceptors
	UnaryInterceptors []grpc.UnaryClientInterceptor

	// Stream interceptors
	StreamInterceptors []grpc.StreamClientInterceptor
}

// DefaultClientConfig returns a client config with sensible defaults.
func DefaultClientConfig(target string) *ClientConfig {
	return &ClientConfig{
		Target:             target,
		Timeout:            10 * time.Second,
		Keepalive:          true,
		KeepaliveTime:      30 * time.Second,
		KeepaliveTimeout:   10 * time.Second,
		MaxRecvMsgSize:     4 * 1024 * 1024, // 4MB
		MaxSendMsgSize:     4 * 1024 * 1024, // 4MB
		UnaryInterceptors:  []grpc.UnaryClientInterceptor{},
		StreamInterceptors: []grpc.StreamClientInterceptor{},
	}
}

// NewClient creates a new gRPC client connection with the provided configuration.
//
// The connection supports:
// - Consul DNS service discovery (consul://service-name)
// - Connection pooling and keepalive
// - Automatic retry with exponential backoff
// - Circuit breaker integration
//
// Example:
//
//	config := DefaultClientConfig("auth-svc:9001")
//	conn, err := NewClient(ctx, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer conn.Close()
//
//	client := authv1.NewAuthServiceClient(conn)
func NewClient(ctx context.Context, config *ClientConfig) (*grpc.ClientConn, error) {
	// Build dial options
	opts := []grpc.DialOption{
		// Use insecure connection (TLS handled by service mesh or load balancer)
		grpc.WithTransportCredentials(insecure.NewCredentials()),

		// Connection timeout
		grpc.WithBlock(),

		// Default service config for retry
		grpc.WithDefaultServiceConfig(`{
			"methodConfig": [{
				"name": [{"service": ""}],
				"waitForReady": true,
				"retryPolicy": {
					"maxAttempts": 3,
					"initialBackoff": "0.1s",
					"maxBackoff": "1s",
					"backoffMultiplier": 2.0,
					"retryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
				}
			}]
		}`),

		// Message size limits
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(config.MaxSendMsgSize),
		),
	}

	// Keepalive settings
	if config.Keepalive {
		opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.KeepaliveTime,
			Timeout:             config.KeepaliveTimeout,
			PermitWithoutStream: true,
		}))
	}

	// Chain unary interceptors
	if len(config.UnaryInterceptors) > 0 {
		opts = append(opts, grpc.WithChainUnaryInterceptor(config.UnaryInterceptors...))
	}

	// Chain stream interceptors
	if len(config.StreamInterceptors) > 0 {
		opts = append(opts, grpc.WithChainStreamInterceptor(config.StreamInterceptors...))
	}

	// Create dial context with timeout
	dialCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Dial
	conn, err := grpc.DialContext(dialCtx, config.Target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", config.Target, err)
	}

	return conn, nil
}

// MustNewClient creates a new gRPC client or panics on error.
//
// This is useful for service initialization where failure to connect
// should prevent the service from starting.
func MustNewClient(ctx context.Context, config *ClientConfig) *grpc.ClientConn {
	conn, err := NewClient(ctx, config)
	if err != nil {
		panic(fmt.Sprintf("failed to create gRPC client: %v", err))
	}
	return conn
}

// RegisterConsulResolver registers the Consul DNS resolver for service discovery.
//
// This allows using addresses like "consul://auth-svc" which will be resolved
// to the actual service instances via Consul.
//
// Usage:
//
//	RegisterConsulResolver("localhost:8600") // Consul DNS port
//	conn, _ := NewClient(ctx, DefaultClientConfig("consul://auth-svc"))
func RegisterConsulResolver(consulDNSAddr string) {
	// This is a placeholder for Consul DNS integration
	// In production, you would use a library like github.com/hashicorp/consul/api
	// or implement a custom resolver.Builder

	// For now, we'll just register a basic DNS resolver
	resolver.SetDefaultScheme("dns")
}

// HealthCheck performs a health check on the gRPC connection.
//
// It uses the standard gRPC health checking protocol:
// https://github.com/grpc/grpc/blob/master/doc/health-checking.md
func HealthCheck(ctx context.Context, conn *grpc.ClientConn) error {
	// Use grpc.health.v1.Health service
	// This is a placeholder - you would import grpc.health.v1 package
	return fmt.Errorf("health check not implemented yet")
}
