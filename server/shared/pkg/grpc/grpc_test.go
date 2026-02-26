package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig("localhost:9001")

	if config.Target != "localhost:9001" {
		t.Errorf("expected target localhost:9001, got %s", config.Target)
	}

	if config.Timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", config.Timeout)
	}

	if !config.Keepalive {
		t.Error("expected keepalive to be enabled")
	}

	if config.KeepaliveTime != 30*time.Second {
		t.Errorf("expected keepalive time 30s, got %v", config.KeepaliveTime)
	}

	if config.MaxRecvMsgSize != 4*1024*1024 {
		t.Errorf("expected max recv msg size 4MB, got %d", config.MaxRecvMsgSize)
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected codes.Code
	}{
		{
			name:     "nil error",
			input:    nil,
			expected: codes.OK,
		},
		{
			name:     "already gRPC error",
			input:    status.Error(codes.NotFound, "not found"),
			expected: codes.NotFound,
		},
		{
			name:     "regular error",
			input:    fmt.Errorf("some error"),
			expected: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.input)

			if tt.input == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			code := GetErrorCode(result)
			if code != tt.expected {
				t.Errorf("expected code %v, got %v", tt.expected, code)
			}
		})
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected codes.Code
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: codes.OK,
		},
		{
			name:     "not found error",
			err:      ErrNotFound,
			expected: codes.NotFound,
		},
		{
			name:     "invalid argument error",
			err:      ErrInvalidArgument,
			expected: codes.InvalidArgument,
		},
		{
			name:     "regular error",
			err:      fmt.Errorf("regular error"),
			expected: codes.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GetErrorCode(tt.err)
			if code != tt.expected {
				t.Errorf("expected code %v, got %v", tt.expected, code)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "unavailable - retryable",
			err:      status.Error(codes.Unavailable, "unavailable"),
			expected: true,
		},
		{
			name:     "deadline exceeded - retryable",
			err:      status.Error(codes.DeadlineExceeded, "timeout"),
			expected: true,
		},
		{
			name:     "resource exhausted - retryable",
			err:      status.Error(codes.ResourceExhausted, "rate limit"),
			expected: true,
		},
		{
			name:     "not found - not retryable",
			err:      ErrNotFound,
			expected: false,
		},
		{
			name:     "invalid argument - not retryable",
			err:      ErrInvalidArgument,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHTTPStatusFromGRPC(t *testing.T) {
	tests := []struct {
		grpcCode   codes.Code
		httpStatus int
	}{
		{codes.OK, 200},
		{codes.Canceled, 499},
		{codes.InvalidArgument, 400},
		{codes.DeadlineExceeded, 504},
		{codes.NotFound, 404},
		{codes.AlreadyExists, 409},
		{codes.PermissionDenied, 403},
		{codes.ResourceExhausted, 429},
		{codes.Unauthenticated, 401},
		{codes.Unimplemented, 501},
		{codes.Internal, 500},
		{codes.Unavailable, 503},
	}

	for _, tt := range tests {
		t.Run(tt.grpcCode.String(), func(t *testing.T) {
			httpStatus := HTTPStatusFromGRPC(tt.grpcCode)
			if httpStatus != tt.httpStatus {
				t.Errorf("expected HTTP %d, got %d", tt.httpStatus, httpStatus)
			}
		})
	}
}

func TestGRPCCodeFromHTTP(t *testing.T) {
	tests := []struct {
		httpStatus int
		grpcCode   codes.Code
	}{
		{200, codes.OK},
		{400, codes.InvalidArgument},
		{401, codes.Unauthenticated},
		{403, codes.PermissionDenied},
		{404, codes.NotFound},
		{409, codes.AlreadyExists},
		{429, codes.ResourceExhausted},
		{500, codes.Internal},
		{501, codes.Unimplemented},
		{503, codes.Unavailable},
		{504, codes.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("HTTP_%d", tt.httpStatus), func(t *testing.T) {
			grpcCode := GRPCCodeFromHTTP(tt.httpStatus)
			if grpcCode != tt.grpcCode {
				t.Errorf("expected gRPC code %v, got %v", tt.grpcCode, grpcCode)
			}
		})
	}
}

func TestDefaultErrorHandler(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		masked   bool
	}{
		{
			name:     "nil error",
			input:    nil,
			masked:   false,
		},
		{
			name:     "invalid argument - not masked",
			input:    ErrInvalidArgument,
			masked:   false,
		},
		{
			name:     "not found - not masked",
			input:    ErrNotFound,
			masked:   false,
		},
		{
			name:     "internal error - masked",
			input:    ErrInternal,
			masked:   true,
		},
		{
			name:     "unknown error - masked",
			input:    fmt.Errorf("some internal error"),
			masked:   true,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultErrorHandler(ctx, tt.input)

			if tt.input == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if tt.masked {
				// Should be masked to Internal error
				code := GetErrorCode(result)
				if code != codes.Internal {
					t.Errorf("expected masked to Internal, got %v", code)
				}
			} else {
				// Should be unchanged
				if GetErrorCode(result) != GetErrorCode(tt.input) {
					t.Errorf("error should not be masked")
				}
			}
		})
	}
}

// Benchmarks

func BenchmarkWrapError(b *testing.B) {
	err := fmt.Errorf("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WrapError(err)
	}
}

func BenchmarkHTTPStatusFromGRPC(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HTTPStatusFromGRPC(codes.NotFound)
	}
}

func BenchmarkGetErrorCode(b *testing.B) {
	err := ErrNotFound

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetErrorCode(err)
	}
}
