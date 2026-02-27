package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestError_Error(t *testing.T) {
	err := New("TEST_ERROR", "Test error message", http.StatusBadRequest)
	expected := "TEST_ERROR: Test error message"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

func TestError_WithDetails(t *testing.T) {
	err := New("TEST_ERROR", "Test", 400)
	details := map[string]interface{}{"field": "email"}
	err = err.WithDetails(details)
	
	if err.Details == nil {
		t.Error("Details should not be nil")
	}
}

func TestError_WithError(t *testing.T) {
	baseErr := errors.New("base error")
	err := New("TEST_ERROR", "Test", 400).WithError(baseErr)
	
	if err.Err != baseErr {
		t.Error("Wrapped error should be set")
	}
}

func TestWrap(t *testing.T) {
	baseErr := errors.New("database connection failed")
	wrapped := Wrap(baseErr, "DB_ERROR", "Failed to connect", http.StatusInternalServerError)
	
	if wrapped.Err != baseErr {
		t.Error("Should wrap the original error")
	}
	if wrapped.Code != "DB_ERROR" {
		t.Errorf("Code = %v, want DB_ERROR", wrapped.Code)
	}
}

func TestIsError(t *testing.T) {
	err := ErrUserNotFound
	if !IsError(err, ErrUserNotFound) {
		t.Error("Should identify error by matching target")
	}
	
	if IsError(err, ErrInvalidInput) {
		t.Error("Should not match different error")
	}
	
	standardErr := errors.New("standard error")
	if IsError(standardErr, ErrUserNotFound) {
		t.Error("Should not match non-Error types")
	}
}

func TestGetHTTPStatus(t *testing.T) {
	err := ErrInvalidInput
	status := GetHTTPStatus(err)
	if status != http.StatusBadRequest {
		t.Errorf("GetHTTPStatus() = %v, want %v", status, http.StatusBadRequest)
	}
	
	standardErr := errors.New("standard error")
	status = GetHTTPStatus(standardErr)
	if status != http.StatusInternalServerError {
		t.Errorf("Should return 500 for standard errors, got %v", status)
	}
}

func TestGetCode(t *testing.T) {
	err := ErrUnauthorized
	code := GetCode(err)
	if code != ErrCodeUnauthorized {
		t.Errorf("GetCode() = %v, want %v", code, ErrCodeUnauthorized)
	}
	
	standardErr := errors.New("standard error")
	code = GetCode(standardErr)
	if code != ErrCodeInternal {
		t.Errorf("Should return INTERNAL_ERROR for standard errors, got %v", code)
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    *Error
		code   string
		status int
	}{
		{"Internal", ErrInternal, ErrCodeInternal, http.StatusInternalServerError},
		{"InvalidInput", ErrInvalidInput, ErrCodeInvalidInput, http.StatusBadRequest},
		{"Unauthorized", ErrUnauthorized, ErrCodeUnauthorized, http.StatusUnauthorized},
		{"Forbidden", ErrForbidden, ErrCodeForbidden, http.StatusForbidden},
		{"UserNotFound", ErrUserNotFound, ErrCodeUserNotFound, http.StatusNotFound},
		{"TokenInvalid", ErrTokenInvalid, ErrCodeTokenInvalid, http.StatusUnauthorized},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Code = %v, want %v", tt.err.Code, tt.code)
			}
			if tt.err.HTTPStatus != tt.status {
				t.Errorf("HTTPStatus = %v, want %v", tt.err.HTTPStatus, tt.status)
			}
		})
	}
}