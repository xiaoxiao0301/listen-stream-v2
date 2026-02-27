// Package errors provides standardized error definitions for the Listen Stream system.
package errors

import (
	"fmt"
	"net/http"
)

// Error represents a structured application error.
type Error struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Details    interface{} `json:"details,omitempty"`
	Err        error  `json:"-"` // Wrapped error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error.
func (e *Error) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error.
func (e *Error) WithDetails(details interface{}) *Error {
	e.Details = details
	return e
}

// WithError wraps another error.
func (e *Error) WithError(err error) *Error {
	e.Err = err
	return e
}

// New creates a new Error.
func New(code, message string, httpStatus int) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Wrap wraps an existing error with error code and message.
func Wrap(err error, code, message string, httpStatus int) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

// Common error codes
const (
	// General errors (1xxx)
	ErrCodeInternal       = "INTERNAL_ERROR"
	ErrCodeInvalidRequest = "INVALID_REQUEST"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeConflict       = "CONFLICT"
	ErrCodeForbidden      = "FORBIDDEN"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
	ErrCodeBadRequest     = "BAD_REQUEST"
	ErrCodeTooManyRequests = "TOO_MANY_REQUESTS"
	
	// Authentication errors (2xxx)
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired       = "TOKEN_EXPIRED"
	ErrCodeTokenInvalid       = "TOKEN_INVALID"
	ErrCodeTokenRevoked       = "TOKEN_REVOKED"
	ErrCodeSMSVerifyFailed    = "SMS_VERIFICATION_FAILED"
	ErrCodeSMSSendFailed      = "SMS_SEND_FAILED"
	ErrCodePhoneInvalid       = "PHONE_INVALID"
	ErrCodeDeviceLimitExceeded = "DEVICE_LIMIT_EXCEEDED"
	
	// User errors (3xxx)
	ErrCodeUserNotFound      = "USER_NOT_FOUND"
	ErrCodeUserAlreadyExists = "USER_ALREADY_EXISTS"
	ErrCodeUserBlocked       = "USER_BLOCKED"
	ErrCodeUserDeleted       = "USER_DELETED"
	
	// Resource errors (4xxx)
	ErrCodeResourceNotFound    = "RESOURCE_NOT_FOUND"
	ErrCodeResourceExists      = "RESOURCE_EXISTS"
	ErrCodeResourceUnavailable = "RESOURCE_UNAVAILABLE"
	ErrCodePlaylistNotFound    = "PLAYLIST_NOT_FOUND"
	ErrCodeSongNotFound        = "SONG_NOT_FOUND"
	ErrCodeFavoriteExists      = "FAVORITE_EXISTS"
	
	// Validation errors (5xxx)
	ErrCodeValidationFailed = "VALIDATION_FAILED"
	ErrCodeInvalidInput     = "INVALID_INPUT"
	ErrCodeMissingField     = "MISSING_FIELD"
	ErrCodeInvalidFormat    = "INVALID_FORMAT"
	
	// Service errors (6xxx)
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeUpstreamError      = "UPSTREAM_ERROR"
	ErrCodeCircuitOpen        = "CIRCUIT_OPEN"
	ErrCodeTimeout            = "TIMEOUT"
	ErrCodeDatabaseError      = "DATABASE_ERROR"
	ErrCodeCacheError         = "CACHE_ERROR"
)

// Predefined errors
var (
	// General errors
	ErrInternal       = New(ErrCodeInternal, "Internal server error", http.StatusInternalServerError)
	ErrInvalidRequest = New(ErrCodeInvalidRequest, "Invalid request", http.StatusBadRequest)
	ErrNotFound       = New(ErrCodeNotFound, "Resource not found", http.StatusNotFound)
	ErrConflict       = New(ErrCodeConflict, "Resource conflict", http.StatusConflict)
	ErrForbidden      = New(ErrCodeForbidden, "Access forbidden", http.StatusForbidden)
	ErrUnauthorized   = New(ErrCodeUnauthorized, "Unauthorized", http.StatusUnauthorized)
	ErrBadRequest     = New(ErrCodeBadRequest, "Bad request", http.StatusBadRequest)
	ErrTooManyRequests = New(ErrCodeTooManyRequests, "Too many requests", http.StatusTooManyRequests)
)

var (
	// Authentication errors
	ErrInvalidCredentials = New(ErrCodeInvalidCredentials, "Invalid credentials", http.StatusUnauthorized)
	ErrTokenExpired       = New(ErrCodeTokenExpired, "Token has expired", http.StatusUnauthorized)
	ErrTokenInvalid       = New(ErrCodeTokenInvalid, "Invalid token", http.StatusUnauthorized)
	ErrTokenRevoked       = New(ErrCodeTokenRevoked, "Token has been revoked", http.StatusUnauthorized)
	ErrSMSVerifyFailed    = New(ErrCodeSMSVerifyFailed, "SMS verification failed", http.StatusBadRequest)
	ErrSMSSendFailed      = New(ErrCodeSMSSendFailed, "Failed to send SMS", http.StatusInternalServerError)
	ErrPhoneInvalid       = New(ErrCodePhoneInvalid, "Invalid phone number", http.StatusBadRequest)
	ErrDeviceLimitExceeded = New(ErrCodeDeviceLimitExceeded, "Device limit exceeded", http.StatusForbidden)
)

var (
	// User errors
	ErrUserNotFound      = New(ErrCodeUserNotFound, "User not found", http.StatusNotFound)
	ErrUserAlreadyExists = New(ErrCodeUserAlreadyExists, "User already exists", http.StatusConflict)
	ErrUserBlocked       = New(ErrCodeUserBlocked, "User is blocked", http.StatusForbidden)
	ErrUserDeleted       = New(ErrCodeUserDeleted, "User has been deleted", http.StatusGone)
)

var (
	// Resource errors
	ErrResourceNotFound    = New(ErrCodeResourceNotFound, "Resource not found", http.StatusNotFound)
	ErrResourceExists      = New(ErrCodeResourceExists, "Resource already exists", http.StatusConflict)
	ErrResourceUnavailable = New(ErrCodeResourceUnavailable, "Resource temporarily unavailable", http.StatusServiceUnavailable)
	ErrPlaylistNotFound    = New(ErrCodePlaylistNotFound, "Playlist not found", http.StatusNotFound)
	ErrSongNotFound        = New(ErrCodeSongNotFound, "Song not found", http.StatusNotFound)
	ErrFavoriteExists      = New(ErrCodeFavoriteExists, "Already in favorites", http.StatusConflict)
)

var (
	// Validation errors
	ErrValidationFailed = New(ErrCodeValidationFailed, "Validation failed", http.StatusBadRequest)
	ErrInvalidInput     = New(ErrCodeInvalidInput, "Invalid input", http.StatusBadRequest)
	ErrMissingField     = New(ErrCodeMissingField, "Required field missing", http.StatusBadRequest)
	ErrInvalidFormat    = New(ErrCodeInvalidFormat, "Invalid format", http.StatusBadRequest)
)

var (
	// Service errors
	ErrServiceUnavailable = New(ErrCodeServiceUnavailable, "Service temporarily unavailable", http.StatusServiceUnavailable)
	ErrUpstreamError      = New(ErrCodeUpstreamError, "Upstream service error", http.StatusBadGateway)
	ErrCircuitOpen        = New(ErrCodeCircuitOpen, "Circuit breaker is open", http.StatusServiceUnavailable)
	ErrTimeout            = New(ErrCodeTimeout, "Request timeout", http.StatusGatewayTimeout)
	ErrDatabaseError      = New(ErrCodeDatabaseError, "Database error", http.StatusInternalServerError)
	ErrCacheError         = New(ErrCodeCacheError, "Cache error", http.StatusInternalServerError)
)

// IsError checks if an error is a specific application error.
func IsError(err error, target *Error) bool {
	if err == nil {
		return false
	}
	appErr, ok := err.(*Error)
	if !ok {
		return false
	}
	return appErr.Code == target.Code
}

// GetHTTPStatus returns the HTTP status code for an error.
// If the error is not an *Error, returns 500.
func GetHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	appErr, ok := err.(*Error)
	if !ok {
		return http.StatusInternalServerError
	}
	return appErr.HTTPStatus
}

// GetCode returns the error code for an error.
// If the error is not an *Error, returns INTERNAL_ERROR.
func GetCode(err error) string {
	if err == nil {
		return ""
	}
	appErr, ok := err.(*Error)
	if !ok {
		return ErrCodeInternal
	}
	return appErr.Code
}