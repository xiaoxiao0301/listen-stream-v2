// Package httputil provides HTTP utility functions.
package httputil

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/listen-stream/server/shared/pkg/errors"
)

// Response represents a standard API response.
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	RequestID string      `json:"request_id"`
}

// ErrorInfo represents error information in the response.
type ErrorInfo struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// SuccessResponse sends a successful response.
func SuccessResponse(c *gin.Context, data interface{}) {
	requestID := GetRequestID(c)
	c.JSON(200, Response{
		Success:   true,
		Data:      data,
		RequestID: requestID,
	})
}

// ErrorResponse sends an error response.
func ErrorResponse(c *gin.Context, err error) {
	requestID := GetRequestID(c)
	
	// Try to cast to application error
	appErr, ok := err.(*errors.Error)
	if !ok {
		// Unknown error - treat as internal error
		appErr = errors.ErrInternal.WithError(err)
	}
	
	c.JSON(appErr.HTTPStatus, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		},
		RequestID: requestID,
	})
}

// PaginationResponse represents a paginated response.
type PaginationResponse struct {
	Success    bool            `json:"success"`
	Data       interface{}     `json:"data"`
	Pagination PaginationInfo  `json:"pagination"`
	RequestID  string          `json:"request_id"`
}

// PaginationInfo holds pagination metadata.
type PaginationInfo struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
	TotalItems int64 `json:"total_items"`
}

// PaginatedResponse sends a paginated response.
func PaginatedResponse(c *gin.Context, data interface{}, page, pageSize int, totalItems int64) {
	requestID := GetRequestID(c)
	
	totalPages := int(totalItems) / pageSize
	if int(totalItems)%pageSize != 0 {
		totalPages++
	}
	
	c.JSON(200, PaginationResponse{
		Success: true,
		Data:    data,
		Pagination: PaginationInfo{
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
			TotalItems: totalItems,
		},
		RequestID: requestID,
	})
}

// GetRequestID retrieves or generates a request ID.
func GetRequestID(c *gin.Context) string {
	requestID := c.GetString("request_id")
	if requestID == "" {
		requestID = uuid.New().String()
	}
	return requestID
}

// RequestIDMiddleware injects a request ID into the context.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		
		c.Next()
	}
}

// CORSMiddleware sets CORS headers.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "3600")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}

// SecurityHeadersMiddleware sets security headers.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		
		c.Next()
	}
}

// GetUserID retrieves the user ID from context.
func GetUserID(c *gin.Context) string {
	userID, _ := c.Get("user_id")
	if userID == nil {
		return ""
	}
	return userID.(string)
}

// GetDeviceID retrieves the device ID from context.
func GetDeviceID(c *gin.Context) string {
	deviceID, _ := c.Get("device_id")
	if deviceID == nil {
		return ""
	}
	return deviceID.(string)
}

// BindAndValidate binds and validates request data.
func BindAndValidate(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return errors.ErrInvalidInput.WithError(err)
	}
	return nil
}