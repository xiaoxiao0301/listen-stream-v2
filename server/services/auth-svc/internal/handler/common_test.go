package handler

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

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
