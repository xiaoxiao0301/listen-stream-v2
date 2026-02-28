package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestRequestID tests the RequestID middleware
func TestRequestID(t *testing.T) {
	t.Run("generates new request ID when not provided", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		middleware := RequestID()
		middleware(c)

		requestID, exists := c.Get("request_id")
		assert.True(t, exists, "request_id should be set in context")
		assert.NotEmpty(t, requestID, "request_id should not be empty")
		assert.NotEmpty(t, w.Header().Get("X-Request-ID"), "X-Request-ID header should be set")
		assert.Equal(t, requestID, w.Header().Get("X-Request-ID"), "header should match context value")
	})

	t.Run("uses provided request ID from header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.Header.Set("X-Request-ID", "custom-request-id")

		middleware := RequestID()
		middleware(c)

		requestID, exists := c.Get("request_id")
		assert.True(t, exists)
		assert.Equal(t, "custom-request-id", requestID)
		assert.Equal(t, "custom-request-id", w.Header().Get("X-Request-ID"))
	})
}

// TestGetRequestID tests the GetRequestID helper
func TestGetRequestID(t *testing.T) {
	t.Run("returns request ID when set", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("request_id", "test-id-123")

		requestID := GetRequestID(c)
		assert.Equal(t, "test-id-123", requestID)
	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		requestID := GetRequestID(c)
		assert.Empty(t, requestID)
	})

	t.Run("returns empty string for invalid type", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("request_id", 12345)

		requestID := GetRequestID(c)
		assert.Empty(t, requestID)
	})
}

// TestCORS tests the CORS middleware
func TestCORS(t *testing.T) {
	t.Run("sets CORS headers for regular requests", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		middleware := CORS()
		middleware(c)

		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Origin"))
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Headers"))
		assert.NotEmpty(t, w.Header().Get("Access-Control-Expose-Headers"))
		assert.NotEmpty(t, w.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("handles OPTIONS preflight requests", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("OPTIONS", "/test", nil)

		middleware := CORS()
		middleware(c)

		assert.True(t, c.IsAborted())
	})
}

// TestSecurityHeaders tests the SecurityHeaders middleware
func TestSecurityHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	middleware := SecurityHeaders()
	middleware(c)

	assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"))
	assert.NotEmpty(t, w.Header().Get("X-Frame-Options"))
	assert.NotEmpty(t, w.Header().Get("X-XSS-Protection"))
	assert.NotEmpty(t, w.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
}
