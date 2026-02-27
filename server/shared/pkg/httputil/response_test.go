package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/listen-stream/server/shared/pkg/errors"
)

func init() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
}

func TestResponse_Structure(t *testing.T) {
	resp := Response{
		Success:   true,
		Data:      map[string]string{"key": "value"},
		RequestID: "test-request-id",
	}
	
	if !resp.Success {
		t.Error("Success should be true")
	}
	if resp.Data == nil {
		t.Error("Data should not be nil")
	}
	if resp.RequestID != "test-request-id" {
		t.Errorf("RequestID = %v, want test-request-id", resp.RequestID)
	}
}

func TestErrorInfo_Structure(t *testing.T) {
	errInfo := ErrorInfo{
		Code:    "TEST_ERROR",
		Message: "This is a test error",
		Details: map[string]string{"field": "email"},
	}
	
	if errInfo.Code != "TEST_ERROR" {
		t.Errorf("Code = %v, want TEST_ERROR", errInfo.Code)
	}
	if errInfo.Message != "This is a test error" {
		t.Errorf("Message = %v, want 'This is a test error'", errInfo.Message)
	}
	if errInfo.Details == nil {
		t.Error("Details should not be nil")
	}
}

func TestSuccessResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-123")
	
	data := map[string]string{"status": "ok"}
	SuccessResponse(c, data)
	
	if w.Code != http.StatusOK {
		t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
	}
	
	if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %v, want application/json", w.Header().Get("Content-Type"))
	}
}

func TestErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-456")
	
	err := errors.ErrInvalidInput
	ErrorResponse(c, err)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestErrorResponse_StandardError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-789")
	
	err := errors.New("CUSTOM_ERROR", "Custom error message", http.StatusInternalServerError)
	ErrorResponse(c, err)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %v, want %v", w.Code, http.StatusInternalServerError)
	}
}

func TestPaginationResponse_Structure(t *testing.T) {
	resp := PaginationResponse{
		Success: true,
		Data:    []string{"item1", "item2", "item3"},
		Pagination: PaginationInfo{
			Page:       1,
			PageSize:   10,
			TotalItems: 50,
			TotalPages: 5,
		},
		RequestID: "page-request",
	}
	
	if !resp.Success {
		t.Error("Success should be true")
	}
	if resp.Pagination.Page != 1 {
		t.Errorf("Pagination.Page = %v, want 1", resp.Pagination.Page)
	}
	if resp.Pagination.PageSize != 10 {
		t.Errorf("Pagination.PageSize = %v, want 10", resp.Pagination.PageSize)
	}
	if resp.Pagination.TotalItems != 50 {
		t.Errorf("Pagination.TotalItems = %v, want 50", resp.Pagination.TotalItems)
	}
	if resp.Pagination.TotalPages != 5 {
		t.Errorf("Pagination.TotalPages = %v, want 5", resp.Pagination.TotalPages)
	}
}

func TestPaginatedResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "page-123")
	
	data := []map[string]string{
		{"id": "1", "name": "Item 1"},
		{"id": "2", "name": "Item 2"},
	}
	
	PaginatedResponse(c, data, 1, 10, 25)
	
	if w.Code != http.StatusOK {
		t.Errorf("Status code = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestGetRequestID_Exists(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("request_id", "existing-id")
	
	requestID := GetRequestID(c)
	if requestID != "existing-id" {
		t.Errorf("RequestID = %v, want existing-id", requestID)
	}
}

func TestGetRequestID_Generated(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	// Don't set request_id
	
	requestID := GetRequestID(c)
	if requestID == "" {
		t.Error("RequestID should be generated if not exists")
	}
	
	// Check if it's a valid UUID format (simple check)
	if len(requestID) < 10 {
		t.Error("Generated RequestID seems too short")
	}
}

func TestGetUserID(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("user_id", "user-123")
	
	userID := GetUserID(c)
	if userID != "user-123" {
		t.Errorf("UserID = %v, want user-123", userID)
	}
}

func TestGetUserID_NotSet(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	
	userID := GetUserID(c)
	if userID != "" {
		t.Errorf("UserID should be empty when not set, got %v", userID)
	}
}

func TestGetDeviceID(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("device_id", "device-456")
	
	deviceID := GetDeviceID(c)
	if deviceID != "device-456" {
		t.Errorf("DeviceID = %v, want device-456", deviceID)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	
	middleware := RequestIDMiddleware()
	middleware(c)
	
	requestID, exists := c.Get("request_id")
	if !exists {
		t.Error("request_id should be set by middleware")
	}
	
	if requestID == "" {
		t.Error("request_id should not be empty")
	}
}

func TestCORSMiddleware(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("OPTIONS", "/test", nil)
	
	middleware := CORSMiddleware()
	middleware(c)
	
	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Access-Control-Allow-Origin header should be set")
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	
	middleware := SecurityHeadersMiddleware()
	middleware(c)
	
	// Check security headers
	headers := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
	}
	
	for _, header := range headers {
		if w.Header().Get(header) == "" {
			t.Errorf("%s header should be set", header)
		}
	}
}

func TestPaginationCalculation(t *testing.T) {
	tests := []struct {
		name          string
		page          int
		pageSize      int
		total         int64
		expectedPages int
	}{
		{"ExactPages", 1, 10, 50, 5},
		{"PartialPage", 1, 10, 55, 6},
		{"SinglePage", 1, 20, 15, 1},
		{"EmptyResult", 1, 10, 0, 0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalPages := int(tt.total+int64(tt.pageSize)-1) / tt.pageSize
			if tt.total == 0 {
				totalPages = 0
			}
			
			if totalPages != tt.expectedPages {
				t.Errorf("TotalPages = %v, want %v", totalPages, tt.expectedPages)
			}
		})
	}
}

func TestResponse_WithError(t *testing.T) {
	resp := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    "VALIDATION_ERROR",
			Message: "Invalid input",
			Details: map[string]string{"field": "email", "reason": "invalid format"},
		},
		RequestID: "error-request",
	}
	
	if resp.Success {
		t.Error("Success should be false for error response")
	}
	if resp.Error == nil {
		t.Error("Error should not be nil")
	}
	if resp.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("Error.Code = %v, want VALIDATION_ERROR", resp.Error.Code)
	}
}

func TestContextHelpers(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	
	// Set context values
	c.Set("user_id", "user-test")
	c.Set("device_id", "device-test")
	c.Set("request_id", "request-test")
	
	// Test all getters
	if GetUserID(c) != "user-test" {
		t.Error("GetUserID failed")
	}
	if GetDeviceID(c) != "device-test" {
		t.Error("GetDeviceID failed")
	}
	if GetRequestID(c) != "request-test" {
		t.Error("GetRequestID failed")
	}
}
