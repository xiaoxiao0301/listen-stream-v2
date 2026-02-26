package interceptor

import (
	"context"
	"fmt"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryInterceptor recovers from panics and converts them to gRPC errors.
type RecoveryInterceptor struct {
	// Optional handler to call on panic
	OnPanic func(p interface{}, stack []byte)
}

// NewRecoveryInterceptor creates a new recovery interceptor.
func NewRecoveryInterceptor() *RecoveryInterceptor {
	return &RecoveryInterceptor{
		OnPanic: defaultPanicHandler,
	}
}

// UnaryServerInterceptor returns a unary server interceptor for panic recovery.
func (r *RecoveryInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if p := recover(); p != nil {
				// Get stack trace
				stack := debug.Stack()

				// Call panic handler
				if r.OnPanic != nil {
					r.OnPanic(p, stack)
				}

				// Convert panic to gRPC error
				err = status.Errorf(codes.Internal, "panic recovered: %v", p)
			}
		}()

		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a stream server interceptor for panic recovery.
func (r *RecoveryInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if p := recover(); p != nil {
				// Get stack trace
				stack := debug.Stack()

				// Call panic handler
				if r.OnPanic != nil {
					r.OnPanic(p, stack)
				}

				// Convert panic to gRPC error
				err = status.Errorf(codes.Internal, "panic recovered: %v", p)
			}
		}()

		return handler(srv, ss)
	}
}

// defaultPanicHandler logs panics to stderr.
func defaultPanicHandler(p interface{}, stack []byte) {
	fmt.Printf("PANIC recovered: %v\n%s\n", p, stack)
}
