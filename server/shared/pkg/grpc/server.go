package grpc

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// ServerConfig holds gRPC server configuration.
type ServerConfig struct {
	// Port to listen on
	Port int
	
	// Service name (for health checks and logging)
	ServiceName string
	
	// Max concurrent streams per connection
	MaxConcurrentStreams uint32
	
	// Connection timeout
	ConnectionTimeout time.Duration
	
	// Max connection idle time
	MaxConnectionIdle time.Duration
	
	// Max connection age
	MaxConnectionAge time.Duration
	
	// Keepalive enforcement policy
	KeepaliveMinTime time.Duration
	
	// Max receive message size (default: 4MB)
	MaxRecvMsgSize int
	
	// Max send message size (default: 4MB)
	MaxSendMsgSize int
	
	// Unary interceptors
	UnaryInterceptors []grpc.UnaryServerInterceptor
	
	// Stream interceptors
	StreamInterceptors []grpc.StreamServerInterceptor
	
	// Enable reflection (for debugging, disable in production)
	EnableReflection bool
}

// DefaultServerConfig returns a server config with sensible defaults.
func DefaultServerConfig(serviceName string, port int) *ServerConfig {
	return &ServerConfig{
		Port:                 port,
		ServiceName:          serviceName,
		MaxConcurrentStreams: 1000,
		ConnectionTimeout:    120 * time.Second,
		MaxConnectionIdle:    300 * time.Second,
		MaxConnectionAge:     600 * time.Second,
		KeepaliveMinTime:     60 * time.Second,
		MaxRecvMsgSize:       4 * 1024 * 1024, // 4MB
		MaxSendMsgSize:       4 * 1024 * 1024, // 4MB
		UnaryInterceptors:    []grpc.UnaryServerInterceptor{},
		StreamInterceptors:   []grpc.StreamServerInterceptor{},
		EnableReflection:     false, // Disable in production
	}
}

// Server wraps a gRPC server with additional utilities.
type Server struct {
	*grpc.Server
	config       *ServerConfig
	healthServer *health.Server
	listener     net.Listener
}

// NewServer creates a new gRPC server with the provided configuration.
//
// The server includes:
// - Health check service (grpc.health.v1.Health)
// - Graceful shutdown handling
// - Keepalive enforcement
// - Connection limits
//
// Example:
//
//	config := DefaultServerConfig("auth-svc", 9001)
//	server, err := NewServer(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	authv1.RegisterAuthServiceServer(server.Server, &authService{})
//
//	if err := server.Serve(); err != nil {
//	    log.Fatal(err)
//	}
func NewServer(config *ServerConfig) (*Server, error) {
	// Build server options
	opts := []grpc.ServerOption{
		// Max concurrent streams
		grpc.MaxConcurrentStreams(config.MaxConcurrentStreams),
		
		// Message size limits
		grpc.MaxRecvMsgSize(config.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(config.MaxSendMsgSize),
		
		// Keepalive enforcement policy
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             config.KeepaliveMinTime,
			PermitWithoutStream: true,
		}),
		
		// Keepalive server parameters
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: config.MaxConnectionIdle,
			MaxConnectionAge:  config.MaxConnectionAge,
			Time:              30 * time.Second,
			Timeout:           10 * time.Second,
		}),
		
		// Connection timeout
		grpc.ConnectionTimeout(config.ConnectionTimeout),
	}
	
	// Chain unary interceptors
	if len(config.UnaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(config.UnaryInterceptors...))
	}
	
	// Chain stream interceptors
	if len(config.StreamInterceptors) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(config.StreamInterceptors...))
	}
	
	// Create gRPC server
	grpcServer := grpc.NewServer(opts...)
	
	// Create health server
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	
	// Set service as serving
	healthServer.SetServingStatus(config.ServiceName, healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	
	// Enable reflection if configured
	if config.EnableReflection {
		reflection.Register(grpcServer)
	}
	
	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %w", config.Port, err)
	}
	
	return &Server{
		Server:       grpcServer,
		config:       config,
		healthServer: healthServer,
		listener:     listener,
	}, nil
}

// Serve starts the gRPC server and blocks until shutdown.
//
// It handles graceful shutdown on SIGINT and SIGTERM signals.
func (s *Server) Serve() error {
	// Create error channel
	errChan := make(chan error, 1)
	
	// Start server in goroutine
	go func() {
		fmt.Printf("gRPC server listening on :%d\n", s.config.Port)
		if err := s.Server.Serve(s.listener); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()
	
	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan:
		fmt.Printf("Received signal %v, shutting down gracefully...\n", sig)
		return s.Shutdown(context.Background())
	}
}

// Shutdown gracefully shuts down the gRPC server.
//
// It stops accepting new connections and waits for existing RPCs to complete
// within the given context deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	// Mark service as not serving
	s.healthServer.SetServingStatus(s.config.ServiceName, healthpb.HealthCheckResponse_NOT_SERVING)
	s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	
	// Create shutdown timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Stop accepting new connections
	stopped := make(chan struct{})
	go func() {
		s.Server.GracefulStop()
		close(stopped)
	}()
	
	// Wait for graceful stop or force stop on timeout
	select {
	case <-stopped:
		fmt.Println("gRPC server stopped gracefully")
		return nil
	case <-shutdownCtx.Done():
		fmt.Println("gRPC server shutdown timeout, forcing stop")
		s.Server.Stop()
		return fmt.Errorf("graceful shutdown timeout")
	}
}

// SetServingStatus sets the serving status for health checks.
//
// This can be used to mark the service as unhealthy during maintenance
// or when dependencies are unavailable.
func (s *Server) SetServingStatus(serving bool) {
	status := healthpb.HealthCheckResponse_NOT_SERVING
	if serving {
		status = healthpb.HealthCheckResponse_SERVING
	}
	s.healthServer.SetServingStatus(s.config.ServiceName, status)
}
