package device

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
)

// MockDeviceRepository 设备仓储Mock
type MockDeviceRepository struct {
	mock.Mock
}

func (m *MockDeviceRepository) Create(ctx context.Context, device *domain.Device) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetByID(ctx context.Context, id string) (*domain.Device, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetByFingerprint(ctx context.Context, userID, fingerprint string) (*domain.Device, error) {
	args := m.Called(ctx, userID, fingerprint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Device), args.Error(1)
}

func (m *MockDeviceRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.Device, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Device), args.Error(1)
}

func (m *MockDeviceRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDeviceRepository) UpdateLoginInfo(ctx context.Context, id string, ip string, loginAt time.Time) error {
	args := m.Called(ctx, id, ip, loginAt)
	return args.Error(0)
}

func (m *MockDeviceRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDeviceRepository) DeleteByUserID(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockDeviceRepository) DeleteInactive(ctx context.Context, before time.Time) error {
	args := m.Called(ctx, before)
	return args.Error(0)
}

// MockUserRepository 用户仓储Mock (需要复用或在此定义)
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	args := m.Called(ctx, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateTokenVersion(ctx context.Context, userID string, version int) error {
	args := m.Called(ctx, userID, version)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *MockUserRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepository) CountActive(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepository) UpdateActive(ctx context.Context, id string, isActive bool) error {
	args := m.Called(ctx, id, isActive)
	return args.Error(0)
}

// TestRegisterDevice_NewDevice 测试注册新设备
func TestRegisterDevice_NewDevice(t *testing.T) {
	ctx := context.Background()
	mockDeviceRepo := new(MockDeviceRepository)
	mockUserRepo := new(MockUserRepository)

	service := NewDeviceService(mockDeviceRepo, mockUserRepo)

	userID := "user-123"
	user := &domain.User{
		ID:       userID,
		Phone:    "13800138000",
		IsActive: true,
	}

	req := &RegisterDeviceRequest{
		UserID:     userID,
		DeviceName: "iPhone 13 Pro",
		DeviceID:   "device-abc-123",
		Platform:   "iOS",
		OSVersion:  "16.0",
		AppVersion: "1.0.0",
		ClientIP:   "192.168.1.1",
	}

	// Mock expectations
	mockUserRepo.On("GetByID", ctx, userID).Return(user, nil)
	mockDeviceRepo.On("GetByFingerprint", ctx, userID, mock.AnythingOfType("string")).Return(nil, domain.ErrDeviceNotFound)
	mockDeviceRepo.On("CountByUserID", ctx, userID).Return(int64(2), nil)
	mockDeviceRepo.On("Create", ctx, mock.AnythingOfType("*domain.Device")).Return(nil)

	// Execute
	device, err := service.RegisterDevice(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, device)
	assert.Equal(t, userID, device.UserID)
	assert.Equal(t, "iPhone 13 Pro", device.DeviceName)
	assert.Equal(t, "iOS", device.Platform)
	assert.NotEmpty(t, device.Fingerprint)

	mockUserRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

// TestVerifyDevice_Success 测试设备验证成功
func TestVerifyDevice_Success(t *testing.T) {
	ctx := context.Background()
	mockDeviceRepo := new(MockDeviceRepository)
	mockUserRepo := new(MockUserRepository)

	service := NewDeviceService(mockDeviceRepo, mockUserRepo)

	userID := "user-123"
	req := &VerifyDeviceRequest{
		UserID:     userID,
		DeviceID:   "device-abc-123",
		DeviceName: "iPhone 13 Pro",
		Platform:   "iOS",
		OSVersion:  "16.0",
		ClientIP:   "192.168.1.1",
	}

	fingerprint := domain.GenerateFingerprint(req.DeviceName, req.Platform, req.DeviceID, req.OSVersion)
	existingDevice := &domain.Device{
		ID:          "device-id-1",
		UserID:      userID,
		DeviceName:  req.DeviceName,
		Fingerprint: fingerprint,
		Platform:    req.Platform,
		LastIP:      "192.168.1.1",
		LastLoginAt: time.Now().Add(-1 * time.Hour),
	}

	// Mock expectations
	mockDeviceRepo.On("GetByFingerprint", ctx, userID, fingerprint).Return(existingDevice, nil)

	// Execute
	result, err := service.VerifyDevice(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsValid)
	assert.False(t, result.IsSuspicious)
	assert.Equal(t, RiskLevelNone, result.Risk)
	assert.False(t, result.FingerprintChanged)

	mockDeviceRepo.AssertExpectations(t)
}

// TestVerifyDevice_FingerprintMismatch 测试设备指纹不匹配
func TestVerifyDevice_FingerprintMismatch(t *testing.T) {
	ctx := context.Background()
	mockDeviceRepo := new(MockDeviceRepository)
	mockUserRepo := new(MockUserRepository)

	service := NewDeviceService(mockDeviceRepo, mockUserRepo)

	userID := "user-123"
	req := &VerifyDeviceRequest{
		UserID:     userID,
		DeviceID:   "device-abc-123",
		DeviceName: "iPhone 13 Pro",
		Platform:   "iOS",
		OSVersion:  "16.0",
		ClientIP:   "192.168.1.1",
	}

	// 当前设备指纹
	currentFingerprint := domain.GenerateFingerprint(req.DeviceName, req.Platform, req.DeviceID, req.OSVersion)

	// 已注册设备使用不同的指纹（伪造）
	differentFingerprint := "different-fingerprint-hash"
	existingDevice := &domain.Device{
		ID:          "device-id-1",
		UserID:      userID,
		DeviceName:  req.DeviceName,
		Fingerprint: differentFingerprint,
		Platform:    req.Platform,
		LastIP:      "192.168.1.1",
		LastLoginAt: time.Now().Add(-1 * time.Hour),
	}

	// Mock expectations - 按当前指纹查找，返回不同指纹的设备
	mockDeviceRepo.On("GetByFingerprint", ctx, userID, currentFingerprint).Return(existingDevice, nil)

	// Execute
	result, err := service.VerifyDevice(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, domain.ErrFingerprintMismatch, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsValid)
	assert.True(t, result.IsSuspicious)
	assert.Equal(t, RiskLevelHigh, result.Risk)
	assert.True(t, result.FingerprintChanged)

	mockDeviceRepo.AssertExpectations(t)
}

// TestListDevices 测试获取设备列表
func TestListDevices(t *testing.T) {
	ctx := context.Background()
	mockDeviceRepo := new(MockDeviceRepository)
	mockUserRepo := new(MockUserRepository)

	service := NewDeviceService(mockDeviceRepo, mockUserRepo)

	userID := "user-123"
	user := &domain.User{
		ID:       userID,
		Phone:    "13800138000",
		IsActive: true,
	}

	devices := []*domain.Device{
		{
			ID:         "device-1",
			UserID:     userID,
			DeviceName: "iPhone 13 Pro",
			Platform:   "iOS",
		},
		{
			ID:         "device-2",
			UserID:     userID,
			DeviceName: "MacBook Pro",
			Platform:   "Desktop",
		},
	}

	// Mock expectations
	mockUserRepo.On("GetByID", ctx, userID).Return(user, nil)
	mockDeviceRepo.On("ListByUserID", ctx, userID).Return(devices, nil)

	// Execute
	result, err := service.ListDevices(ctx, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Equal(t, "device-1", result[0].ID)
	assert.Equal(t, "device-2", result[1].ID)

	mockUserRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}
