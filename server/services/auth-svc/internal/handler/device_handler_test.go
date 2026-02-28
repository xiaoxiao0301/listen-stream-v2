package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
	jwtservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/jwt"
)

// TestListDevices_Success 测试获取设备列表成功
func TestListDevices_Success(t *testing.T) {
	mockJWT := new(MockJWTService)
	mockDevice := new(MockDeviceService)

	handler := NewDeviceHandler(mockDevice, mockJWT)

	claims := &jwtservice.TokenClaims{
		UserID:   "user-123",
		DeviceID: "device-456",
	}

	devices := []*domain.Device{
		{
			ID:          "device-456",
			UserID:      "user-123",
			DeviceName:  "iPhone 13",
			Platform:    "iOS",
			AppVersion:  "1.0.0",
			LastIP:      "192.168.1.1",
			LastLoginAt: time.Now(),
			CreatedAt:   time.Now(),
		},
		{
			ID:          "device-789",
			UserID:      "user-123",
			DeviceName:  "MacBook Pro",
			Platform:    "Desktop",
			AppVersion:  "1.0.0",
			LastIP:      "192.168.1.2",
			LastLoginAt: time.Now().Add(-1 * time.Hour),
			CreatedAt:   time.Now().Add(-24 * time.Hour),
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/devices", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	mockJWT.On("ValidateAccessToken", mock.Anything, "valid-token", "").Return(claims, nil)
	mockDevice.On("ListDevices", mock.Anything, "user-123").Return(devices, nil)

	handler.ListDevices(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp ListDevicesResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, 2, resp.Total)
	assert.Len(t, resp.Devices, 2)
	assert.True(t, resp.Devices[0].IsCurrent)   // 当前设备
	assert.False(t, resp.Devices[1].IsCurrent) // 其他设备

	mockJWT.AssertExpectations(t)
	mockDevice.AssertExpectations(t)
}

// TestListDevices_Unauthorized 测试未授权访问
func TestListDevices_Unauthorized(t *testing.T) {
	mockJWT := new(MockJWTService)
	mockDevice := new(MockDeviceService)

	handler := NewDeviceHandler(mockDevice, mockJWT)

	req := httptest.NewRequest(http.MethodGet, "/devices", nil)
	// 没有Authorization头
	w := httptest.NewRecorder()

	handler.ListDevices(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp["success"].(bool))
}

// TestRemoveDevice_Success 测试删除设备成功
func TestRemoveDevice_Success(t *testing.T) {
	mockJWT := new(MockJWTService)
	mockDevice := new(MockDeviceService)

	handler := NewDeviceHandler(mockDevice, mockJWT)

	claims := &jwtservice.TokenClaims{
		UserID:   "user-123",
		DeviceID: "device-456", // 当前设备
	}

	reqBody := RemoveDeviceRequest{
		DeviceID: "device-789", // 删除另一个设备
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/devices/remove", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	mockJWT.On("ValidateAccessToken", mock.Anything, "valid-token", "").Return(claims, nil)
	mockDevice.On("RemoveDevice", mock.Anything, "user-123", "device-789").Return(nil)

	handler.RemoveDevice(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp RemoveDeviceResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.True(t, resp.Success)

	mockJWT.AssertExpectations(t)
	mockDevice.AssertExpectations(t)
}

// TestRemoveDevice_CurrentDevice 测试不能删除当前设备
func TestRemoveDevice_CurrentDevice(t *testing.T) {
	mockJWT := new(MockJWTService)
	mockDevice := new(MockDeviceService)

	handler := NewDeviceHandler(mockDevice, mockJWT)

	claims := &jwtservice.TokenClaims{
		UserID:   "user-123",
		DeviceID: "device-456",
	}

	reqBody := RemoveDeviceRequest{
		DeviceID: "device-456", // 尝试删除当前设备
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/devices/remove", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	mockJWT.On("ValidateAccessToken", mock.Anything, "valid-token", "").Return(claims, nil)

	handler.RemoveDevice(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp["success"].(bool))

	mockJWT.AssertExpectations(t)
	// RemoveDevice should NOT be called
	mockDevice.AssertNotCalled(t, "RemoveDevice")
}

// TestRemoveDevice_Unauthorized 测试未授权删除设备
func TestRemoveDevice_Unauthorized(t *testing.T) {
	mockJWT := new(MockJWTService)
	mockDevice := new(MockDeviceService)

	handler := NewDeviceHandler(mockDevice, mockJWT)

	reqBody := RemoveDeviceRequest{
		DeviceID: "device-789",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/devices/remove", bytes.NewReader(body))
	// 没有Authorization头
	w := httptest.NewRecorder()

	handler.RemoveDevice(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.False(t, resp["success"].(bool))
}
