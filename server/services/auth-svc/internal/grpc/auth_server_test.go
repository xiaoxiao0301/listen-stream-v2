package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authv1 "github.com/listen-stream/server/shared/proto/auth/v1"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
	deviceservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/device"
	jwtservice "github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/service/jwt"
)

// MockJWTService JWT服务Mock
type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) GenerateTokenPair(ctx context.Context, userID, deviceID, clientIP string) (*jwtservice.TokenPair, error) {
	args := m.Called(ctx, userID, deviceID, clientIP)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtservice.TokenPair), args.Error(1)
}

func (m *MockJWTService) ValidateAccessToken(ctx context.Context, token, clientIP string) (*jwtservice.TokenClaims, error) {
	args := m.Called(ctx, token, clientIP)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtservice.TokenClaims), args.Error(1)
}

func (m *MockJWTService) ValidateRefreshToken(ctx context.Context, token string) (*jwtservice.TokenClaims, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtservice.TokenClaims), args.Error(1)
}

func (m *MockJWTService) RefreshAccessToken(ctx context.Context, refreshToken, clientIP string) (*jwtservice.TokenPair, error) {
	args := m.Called(ctx, refreshToken, clientIP)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwtservice.TokenPair), args.Error(1)
}

func (m *MockJWTService) RevokeUserTokens(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockJWTService) GetTokenExpiry() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

// MockDeviceService 设备服务Mock
type MockDeviceService struct {
	mock.Mock
}

func (m *MockDeviceService) RegisterDevice(ctx context.Context, req *deviceservice.RegisterDeviceRequest) (*domain.Device, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Device), args.Error(1)
}

func (m *MockDeviceService) VerifyDevice(ctx context.Context, req *deviceservice.VerifyDeviceRequest) (*deviceservice.DeviceVerificationResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*deviceservice.DeviceVerificationResult), args.Error(1)
}

func (m *MockDeviceService) ListDevices(ctx context.Context, userID string) ([]*domain.Device, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Device), args.Error(1)
}

func (m *MockDeviceService) RemoveDevice(ctx context.Context, userID, deviceID string) error {
	args := m.Called(ctx, userID, deviceID)
	return args.Error(0)
}

func (m *MockDeviceService) RemoveInactiveDevices(ctx context.Context, days int) (int, error) {
	args := m.Called(ctx, days)
	return args.Get(0).(int), args.Error(1)
}

// TestVerifyToken_Success 测试Token验证成功
func TestVerifyToken_Success(t *testing.T) {
	ctx := context.Background()
	mockJWT := new(MockJWTService)
	mockDevice := new(MockDeviceService)

	server := NewAuthServer(mockJWT, mockDevice)

	req := &authv1.VerifyTokenRequest{
		AccessToken: "valid-token",
		ClientIp:    "192.168.1.1",
	}

	claims := &jwtservice.TokenClaims{
		UserID:       "user-123",
		DeviceID:     "device-456",
		TokenVersion: 1,
	}

	mockJWT.On("ValidateAccessToken", ctx, "valid-token", "192.168.1.1").Return(claims, nil)

	resp, err := server.VerifyToken(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Valid)
	assert.Equal(t, "user-123", resp.User.Id)
	assert.Equal(t, "device-456", resp.Device.DeviceId)

	mockJWT.AssertExpectations(t)
}

// TestVerifyToken_InvalidToken 测试Token验证失败
func TestVerifyToken_InvalidToken(t *testing.T) {
	ctx := context.Background()
	mockJWT := new(MockJWTService)
	mockDevice := new(MockDeviceService)

	server := NewAuthServer(mockJWT, mockDevice)

	req := &authv1.VerifyTokenRequest{
		AccessToken: "invalid-token",
	}

	mockJWT.On("ValidateAccessToken", ctx, "invalid-token", "").Return(nil, jwtservice.ErrTokenVersionMismatch)

	resp, err := server.VerifyToken(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.Valid)
	assert.Equal(t, authv1.ErrorCode_ERROR_CODE_VERSION_MISMATCH, resp.ErrorCode)

	mockJWT.AssertExpectations(t)
}

// TestRefreshToken_Success 测试刷新Token成功
func TestRefreshToken_Success(t *testing.T) {
	ctx := context.Background()
	mockJWT := new(MockJWTService)
	mockDevice := new(MockDeviceService)

	server := NewAuthServer(mockJWT, mockDevice)

	req := &authv1.RefreshTokenRequest{
		RefreshToken: "refresh-token",
		DeviceId:     "device-456",
	}

	tokenPair := &jwtservice.TokenPair{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		TokenType:    "Bearer",
	}

	mockJWT.On("RefreshAccessToken", ctx, "refresh-token", "").Return(tokenPair, nil)

	resp, err := server.RefreshToken(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "new-access-token", resp.AccessToken)
	assert.Equal(t, "new-refresh-token", resp.RefreshToken)

	mockJWT.AssertExpectations(t)
}

// TestRevokeToken_Success 测试撤销Token成功
func TestRevokeToken_Success(t *testing.T) {
	ctx := context.Background()
	mockJWT := new(MockJWTService)
	mockDevice := new(MockDeviceService)

	server := NewAuthServer(mockJWT, mockDevice)

	req := &authv1.RevokeTokenRequest{
		AccessToken: "token-to-revoke",
		Reason:      "user logout",
	}

	claims := &jwtservice.TokenClaims{
		UserID:   "user-123",
		DeviceID: "device-456",
	}

	mockJWT.On("ValidateAccessToken", ctx, "token-to-revoke", "").Return(claims, nil)
	mockJWT.On("RevokeUserTokens", ctx, "user-123").Return(nil)

	resp, err := server.RevokeToken(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)

	mockJWT.AssertExpectations(t)
}

// TestRevokeDevice_Success 测试撤销设备成功
func TestRevokeDevice_Success(t *testing.T) {
	ctx := context.Background()
	mockJWT := new(MockJWTService)
	mockDevice := new(MockDeviceService)

	server := NewAuthServer(mockJWT, mockDevice)

	req := &authv1.RevokeDeviceRequest{
		UserId:   "user-123",
		DeviceId: "device-456",
		Reason:   "suspicious activity",
	}

	mockDevice.On("RemoveDevice", ctx, "user-123", "device-456").Return(nil)
	mockJWT.On("RevokeUserTokens", ctx, "user-123").Return(nil)

	resp, err := server.RevokeDevice(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)

	mockJWT.AssertExpectations(t)
	mockDevice.AssertExpectations(t)
}

// TestGetUserDevices_Success 测试获取设备列表成功
func TestGetUserDevices_Success(t *testing.T) {
	ctx := context.Background()
	mockJWT := new(MockJWTService)
	mockDevice := new(MockDeviceService)

	server := NewAuthServer(mockJWT, mockDevice)

	req := &authv1.GetUserDevicesRequest{
		UserId: "user-123",
	}

	devices := []*domain.Device{
		{
			ID:          "device-1",
			UserID:      "user-123",
			DeviceName:  "iPhone 13",
			Platform:    "iOS",
			Fingerprint: "fp1",
			LastIP:      "192.168.1.1",
			LastLoginAt: time.Now(),
			CreatedAt:   time.Now(),
		},
		{
			ID:          "device-2",
			UserID:      "user-123",
			DeviceName:  "MacBook Pro",
			Platform:    "Desktop",
			Fingerprint: "fp2",
			LastIP:      "192.168.1.2",
			LastLoginAt: time.Now(),
			CreatedAt:   time.Now(),
		},
	}

	mockDevice.On("ListDevices", ctx, "user-123").Return(devices, nil)

	resp, err := server.GetUserDevices(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Devices, 2)
	assert.Equal(t, "device-1", resp.Devices[0].Id)
	assert.Equal(t, "device-2", resp.Devices[1].Id)

	mockDevice.AssertExpectations(t)
}
