package middleware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
	"github.com/stretchr/testify/assert"
)

// MockLogger for testing
type MockLogger struct {
	InfoCalls  []LogCall
	WarnCalls  []LogCall
	ErrorCalls []LogCall
}

type LogCall struct {
	Message string
	Fields  []logger.Field
}

func (m *MockLogger) Debug(msg string, fields ...logger.Field) {}

func (m *MockLogger) Info(msg string, fields ...logger.Field) {
	m.InfoCalls = append(m.InfoCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockLogger) Warn(msg string, fields ...logger.Field) {
	m.WarnCalls = append(m.WarnCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockLogger) Error(msg string, fields ...logger.Field) {
	m.ErrorCalls = append(m.ErrorCalls, LogCall{Message: msg, Fields: fields})
}

func (m *MockLogger) Fatal(msg string, fields ...logger.Field) {}

func (m *MockLogger) WithContext(ctx context.Context) logger.Logger {
	return m
}

func (m *MockLogger) WithFields(fields ...logger.Field) logger.Logger {
	return m
}

func (m *MockLogger) Writer() io.Writer {
	return nil
}

func (m *MockLogger) SetLevel(level logger.Level) {}

func (m *MockLogger) GetLevel() logger.Level {
	return logger.InfoLevel
}

// TestLogging tests the Logging middleware
func TestLogging(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("logs successful requests", func(t *testing.T) {
		mockLog := &MockLogger{}
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		router.Use(RequestID())
		router.Use(Logging(mockLog))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		c.Request = httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, c.Request)

		assert.Len(t, mockLog.InfoCalls, 1)
		assert.Contains(t, mockLog.InfoCalls[0].Message, "HTTP request completed")
		assert.Len(t, mockLog.ErrorCalls, 0)
	})

	t.Run("logs client errors as warnings", func(t *testing.T) {
		mockLog := &MockLogger{}
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		router.Use(RequestID())
		router.Use(Logging(mockLog))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(400, gin.H{"error": "bad request"})
		})

		c.Request = httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, c.Request)

		assert.Len(t, mockLog.WarnCalls, 1)
		assert.Contains(t, mockLog.WarnCalls[0].Message, "client error")
	})

	t.Run("logs server errors", func(t *testing.T) {
		mockLog := &MockLogger{}
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		router.Use(RequestID())
		router.Use(Logging(mockLog))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(500, gin.H{"error": "server error"})
		})

		c.Request = httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, c.Request)

		assert.Len(t, mockLog.ErrorCalls, 1)
		assert.Contains(t, mockLog.ErrorCalls[0].Message, "server error")
	})

	t.Run("includes request metadata", func(t *testing.T) {
		mockLog := &MockLogger{}
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		router.Use(RequestID())
		router.Use(Logging(mockLog))
		router.POST("/test", func(c *gin.Context) {
			c.Set("user_id", "user-123")
			c.Set("device_id", "device-456")
			c.JSON(200, gin.H{"status": "ok"})
		})

		c.Request = httptest.NewRequest("POST", "/test?foo=bar", nil)
		router.ServeHTTP(w, c.Request)

		assert.Len(t, mockLog.InfoCalls, 1)
		fields := mockLog.InfoCalls[0].Fields

		// Find specific fields
		hasMethod := false
		hasPath := false
		hasQuery := false
		hasUserID := false
		hasDeviceID := false

		for _, field := range fields {
			switch field.Key {
			case "method":
				hasMethod = field.Value == "POST"
			case "path":
				hasPath = field.Value == "/test"
			case "query":
				hasQuery = field.Value == "foo=bar"
			case "user_id":
				hasUserID = field.Value == "user-123"
			case "device_id":
				hasDeviceID = field.Value == "device-456"
			}
		}

		assert.True(t, hasMethod, "should log method")
		assert.True(t, hasPath, "should log path")
		assert.True(t, hasQuery, "should log query")
		assert.True(t, hasUserID, "should log user_id")
		assert.True(t, hasDeviceID, "should log device_id")
	})
}

// TestRecovery tests the Recovery middleware
func TestRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("recovers from panic", func(t *testing.T) {
		mockLog := &MockLogger{}
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		router.Use(RequestID())
		router.Use(Recovery(mockLog))
		router.GET("/test", func(c *gin.Context) {
			panic("test panic")
		})

		c.Request = httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.False(t, response["success"].(bool))
		assert.NotNil(t, response["error"])

		assert.Len(t, mockLog.ErrorCalls, 1)
		assert.Contains(t, mockLog.ErrorCalls[0].Message, "Panic recovered")

		// Find panic field
		hasPanic := false
		hasStack := false
		for _, field := range mockLog.ErrorCalls[0].Fields {
			if field.Key == "panic" {
				hasPanic = true
				assert.Contains(t, field.Value, "test panic")
			}
			if field.Key == "stack" {
				hasStack = true
			}
		}
		assert.True(t, hasPanic, "should log panic message")
		assert.True(t, hasStack, "should log stack trace")
	})

	t.Run("does not affect normal requests", func(t *testing.T) {
		mockLog := &MockLogger{}
		w := httptest.NewRecorder()
		c, router := gin.CreateTestContext(w)

		router.Use(RequestID())
		router.Use(Recovery(mockLog))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		c.Request = httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, c.Request)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Len(t, mockLog.ErrorCalls, 0)
	})
}
