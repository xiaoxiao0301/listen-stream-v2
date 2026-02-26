package interceptor

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor logs all gRPC calls with request ID, duration, and errors.
type LoggingInterceptor struct {
	logger Logger
}

// Logger is an interface for logging gRPC calls.
type Logger interface {
	Info(msg string, fields ...any)
	Error(msg string, fields ...any)
	Warn(msg string, fields ...any)
}

// defaultLogger is a simple logger implementation using standard log package.
type defaultLogger struct{}

func (l *defaultLogger) Info(msg string, fields ...any) {
	log.Printf("[INFO] %s %v", msg, fields)
}

func (l *defaultLogger) Error(msg string, fields ...any) {
	log.Printf("[ERROR] %s %v", msg, fields)
}

func (l *defaultLogger) Warn(msg string, fields ...any) {
	log.Printf("[WARN] %s %v", msg, fields)
}

// NewLoggingInterceptor creates a new logging interceptor.
//
// If logger is nil, it uses the default logger.
func NewLoggingInterceptor(logger Logger) *LoggingInterceptor {
	if logger == nil {
		logger = &defaultLogger{}
	}
	return &LoggingInterceptor{logger: logger}
}

// UnaryServerInterceptor returns a unary server interceptor for logging.
func (l *LoggingInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Extract request ID from metadata
		requestID := extractRequestID(ctx)

		// Call handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Log based on error status
		code := status.Code(err)
		if err != nil {
			l.logger.Error("gRPC request failed",
				"method", info.FullMethod,
				"request_id", requestID,
				"duration", duration,
				"code", code,
				"error", err.Error(),
			)
		} else {
			l.logger.Info("gRPC request completed",
				"method", info.FullMethod,
				"request_id", requestID,
				"duration", duration,
				"code", codes.OK,
			)
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a stream server interceptor for logging.
func (l *LoggingInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// Extract request ID
		requestID := extractRequestID(ss.Context())

		// Call handler
		err := handler(srv, ss)

		// Calculate duration
		duration := time.Since(start)

		// Log based on error status
		code := status.Code(err)
		if err != nil {
			l.logger.Error("gRPC stream failed",
				"method", info.FullMethod,
				"request_id", requestID,
				"duration", duration,
				"code", code,
				"error", err.Error(),
			)
		} else {
			l.logger.Info("gRPC stream completed",
				"method", info.FullMethod,
				"request_id", requestID,
				"duration", duration,
				"code", codes.OK,
			)
		}

		return err
	}
}

// UnaryClientInterceptor returns a unary client interceptor for logging.
func (l *LoggingInterceptor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()

		// Extract request ID
		requestID := extractRequestID(ctx)

		// Call invoker
		err := invoker(ctx, method, req, reply, cc, opts...)

		// Calculate duration
		duration := time.Since(start)

		// Log
		code := status.Code(err)
		if err != nil {
			l.logger.Error("gRPC client request failed",
				"method", method,
				"request_id", requestID,
				"duration", duration,
				"code", code,
				"error", err.Error(),
			)
		} else {
			l.logger.Info("gRPC client request completed",
				"method", method,
				"request_id", requestID,
				"duration", duration,
				"code", codes.OK,
			)
		}

		return err
	}
}

// StreamClientInterceptor returns a stream client interceptor for logging.
func (l *LoggingInterceptor) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		start := time.Now()

		// Extract request ID
		requestID := extractRequestID(ctx)

		// Call streamer
		cs, err := streamer(ctx, desc, cc, method, opts...)

		// Calculate duration
		duration := time.Since(start)

		// Log
		if err != nil {
			l.logger.Error("gRPC client stream failed",
				"method", method,
				"request_id", requestID,
				"duration", duration,
				"error", err.Error(),
			)
		} else {
			l.logger.Info("gRPC client stream started",
				"method", method,
				"request_id", requestID,
				"duration", duration,
			)
		}

		return cs, err
	}
}

// extractRequestID extracts the request ID from context metadata.
func extractRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// Try outgoing context
		md, ok = metadata.FromOutgoingContext(ctx)
		if !ok {
			return generateRequestID()
		}
	}

	if ids := md.Get("x-request-id"); len(ids) > 0 {
		return ids[0]
	}

	return generateRequestID()
}

// generateRequestID generates a new request ID.
func generateRequestID() string {
	// In production, use UUID or similar
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}
