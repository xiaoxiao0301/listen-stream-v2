package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Predefined gRPC errors for common scenarios.
var (
	ErrInternal          = status.Error(codes.Internal, "internal server error")
	ErrInvalidArgument   = status.Error(codes.InvalidArgument, "invalid argument")
	ErrNotFound          = status.Error(codes.NotFound, "not found")
	ErrAlreadyExists     = status.Error(codes.AlreadyExists, "already exists")
	ErrPermissionDenied  = status.Error(codes.PermissionDenied, "permission denied")
	ErrUnauthenticated   = status.Error(codes.Unauthenticated, "unauthenticated")
	ErrResourceExhausted = status.Error(codes.ResourceExhausted, "resource exhausted")
	ErrFailedPrecondition = status.Error(codes.FailedPrecondition, "failed precondition")
	ErrAborted           = status.Error(codes.Aborted, "aborted")
	ErrOutOfRange        = status.Error(codes.OutOfRange, "out of range")
	ErrUnimplemented     = status.Error(codes.Unimplemented, "unimplemented")
	ErrUnavailable       = status.Error(codes.Unavailable, "service unavailable")
	ErrDeadlineExceeded  = status.Error(codes.DeadlineExceeded, "deadline exceeded")
)

// WrapError converts a regular error to a gRPC status error.
//
// If the error is already a gRPC error, it returns it unchanged.
// Otherwise, it wraps the error as codes.Internal.
func WrapError(err error) error {
	if err == nil {
		return nil
	}

	// Already a gRPC error
	if _, ok := status.FromError(err); ok {
		return err
	}

	// Convert to gRPC error
	return status.Error(codes.Internal, err.Error())
}

// IsGRPCError checks if an error is a gRPC status error.
func IsGRPCError(err error) bool {
	_, ok := status.FromError(err)
	return ok
}

// GetErrorCode extracts the gRPC status code from an error.
//
// Returns codes.Unknown if the error is not a gRPC error.
func GetErrorCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}

	st, ok := status.FromError(err)
	if !ok {
		return codes.Unknown
	}

	return st.Code()
}

// GetErrorMessage extracts the error message from a gRPC error.
func GetErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	st, ok := status.FromError(err)
	if !ok {
		return err.Error()
	}

	return st.Message()
}

// IsRetryable returns true if the error is retryable.
//
// Retryable errors include:
// - Unavailable
// - DeadlineExceeded
// - ResourceExhausted (rate limit)
func IsRetryable(err error) bool {
	code := GetErrorCode(err)
	return code == codes.Unavailable ||
		code == codes.DeadlineExceeded ||
		code == codes.ResourceExhausted
}

// HTTPStatusFromGRPC converts a gRPC status code to HTTP status code.
//
// This is useful for API gateways that need to translate gRPC errors
// to HTTP responses.
func HTTPStatusFromGRPC(code codes.Code) int {
	switch code {
	case codes.OK:
		return 200
	case codes.Canceled:
		return 499 // Client closed request
	case codes.InvalidArgument:
		return 400
	case codes.DeadlineExceeded:
		return 504
	case codes.NotFound:
		return 404
	case codes.AlreadyExists:
		return 409
	case codes.PermissionDenied:
		return 403
	case codes.ResourceExhausted:
		return 429
	case codes.FailedPrecondition:
		return 400
	case codes.Aborted:
		return 409
	case codes.OutOfRange:
		return 400
	case codes.Unimplemented:
		return 501
	case codes.Internal:
		return 500
	case codes.Unavailable:
		return 503
	case codes.Unauthenticated:
		return 401
	default:
		return 500
	}
}

// GRPCCodeFromHTTP converts an HTTP status code to gRPC status code.
//
// This is useful for translating upstream HTTP API errors to gRPC.
func GRPCCodeFromHTTP(httpStatus int) codes.Code {
	switch httpStatus {
	case 200, 201, 204:
		return codes.OK
	case 400:
		return codes.InvalidArgument
	case 401:
		return codes.Unauthenticated
	case 403:
		return codes.PermissionDenied
	case 404:
		return codes.NotFound
	case 409:
		return codes.AlreadyExists
	case 429:
		return codes.ResourceExhausted
	case 499:
		return codes.Canceled
	case 500:
		return codes.Internal
	case 501:
		return codes.Unimplemented
	case 503:
		return codes.Unavailable
	case 504:
		return codes.DeadlineExceeded
	default:
		return codes.Unknown
	}
}

// ErrorHandler is a function that handles errors during request processing.
type ErrorHandler func(ctx context.Context, err error) error

// DefaultErrorHandler is the default error handler that masks internal errors.
//
// It prevents leaking sensitive error details to clients while logging
// the full error for debugging.
func DefaultErrorHandler(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	code := GetErrorCode(err)

	// Don't mask client errors
	if code == codes.InvalidArgument ||
		code == codes.NotFound ||
		code == codes.AlreadyExists ||
		code == codes.PermissionDenied ||
		code == codes.Unauthenticated ||
		code == codes.ResourceExhausted {
		return err
	}

	// Mask internal errors
	if code == codes.Internal || code == codes.Unknown {
		// TODO: Log the full error with request ID
		fmt.Printf("Internal error: %v\n", err)
		return ErrInternal
	}

	return err
}
