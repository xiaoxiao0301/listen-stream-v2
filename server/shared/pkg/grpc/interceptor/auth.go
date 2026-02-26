package interceptor

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor validates JWT tokens and extracts user information.
type AuthInterceptor struct {
	// TokenValidator validates JWT tokens and returns user ID
	TokenValidator func(token string) (userID string, err error)

	// SkipMethods is a list of methods that don't require authentication
	SkipMethods []string

	// Optional flag to allow requests without tokens (for public APIs)
	Optional bool
}

// NewAuthInterceptor creates a new auth interceptor.
//
// tokenValidator should verify the JWT token and return the user ID.
func NewAuthInterceptor(tokenValidator func(string) (string, error)) *AuthInterceptor {
	return &AuthInterceptor{
		TokenValidator: tokenValidator,
		SkipMethods:    []string{},
		Optional:       false,
	}
}

// WithSkipMethods adds methods that don't require authentication.
func (a *AuthInterceptor) WithSkipMethods(methods ...string) *AuthInterceptor {
	a.SkipMethods = append(a.SkipMethods, methods...)
	return a
}

// WithOptional makes authentication optional.
func (a *AuthInterceptor) WithOptional() *AuthInterceptor {
	a.Optional = true
	return a
}

// UnaryServerInterceptor returns a unary server interceptor for authentication.
func (a *AuthInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Check if method should skip auth
		if a.shouldSkipAuth(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract token from metadata
		token, err := extractToken(ctx)
		if err != nil {
			if a.Optional {
				return handler(ctx, req)
			}
			return nil, status.Error(codes.Unauthenticated, "missing or invalid token")
		}

		// Validate token
		userID, err := a.TokenValidator(token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Add user ID to context
		ctx = withUserID(ctx, userID)

		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a stream server interceptor for authentication.
func (a *AuthInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Check if method should skip auth
		if a.shouldSkipAuth(info.FullMethod) {
			return handler(srv, ss)
		}

		ctx := ss.Context()

		// Extract token
		token, err := extractToken(ctx)
		if err != nil {
			if a.Optional {
				return handler(srv, ss)
			}
			return status.Error(codes.Unauthenticated, "missing or invalid token")
		}

		// Validate token
		userID, err := a.TokenValidator(token)
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		}

		// Add user ID to context
		ctx = withUserID(ctx, userID)

		// Wrap stream with authenticated context
		wrapped := &authenticatedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrapped)
	}
}

// shouldSkipAuth checks if the method should skip authentication.
func (a *AuthInterceptor) shouldSkipAuth(method string) bool {
	for _, skip := range a.SkipMethods {
		if method == skip || strings.HasPrefix(method, skip) {
			return true
		}
	}
	return false
}

// extractToken extracts the JWT token from metadata.
//
// It looks for the token in the "authorization" header with "Bearer " prefix.
func extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	token := values[0]

	// Remove "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

	if token == "" {
		return "", status.Error(codes.Unauthenticated, "empty token")
	}

	return token, nil
}

// Context keys for user information.
type contextKey string

const (
	userIDKey contextKey = "user_id"
)

// withUserID adds user ID to context.
func withUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID extracts user ID from context.
//
// Returns empty string if user ID is not present.
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}

// authenticatedServerStream wraps grpc.ServerStream with authenticated context.
type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedServerStream) Context() context.Context {
	return s.ctx
}
