package device

import (
	"context"
	"fmt"
	"time"

	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/repository"
)

// DeviceService 设备管理服务接口
type DeviceService interface {
	// RegisterDevice 注册或更新设备
	// 如果设备已存在（相同指纹），则更新登录信息
	// 如果设备不存在，则创建新设备（可能触发设备限制检查）
	RegisterDevice(ctx context.Context, req *RegisterDeviceRequest) (*domain.Device, error)

	// VerifyDevice 验证设备（检测异常登录）
	// 检查设备指纹是否发生变化，返回验证结果和风险等级
	VerifyDevice(ctx context.Context, req *VerifyDeviceRequest) (*DeviceVerificationResult, error)

	// ListDevices 获取用户的所有设备
	ListDevices(ctx context.Context, userID string) ([]*domain.Device, error)

	// RemoveDevice 删除设备（用户主动踢出设备）
	RemoveDevice(ctx context.Context, userID, deviceID string) error

	// RemoveInactiveDevices 删除不活跃设备（自动清理）
	RemoveInactiveDevices(ctx context.Context, days int) (int, error)
}

// RegisterDeviceRequest 设备注册请求
type RegisterDeviceRequest struct {
	UserID      string
	DeviceName  string
	DeviceID    string // 客户端生成的设备唯一标识
	Platform    string
	OSVersion   string
	AppVersion  string
	ClientIP    string
}

// VerifyDeviceRequest 设备验证请求
type VerifyDeviceRequest struct {
	UserID       string
	DeviceID     string // 客户端生成的设备唯一标识
	DeviceName   string
	Platform     string
	OSVersion    string
	ClientIP     string
}

// DeviceVerificationResult 设备验证结果
type DeviceVerificationResult struct {
	IsValid            bool     // 是否通过验证
	IsSuspicious       bool     // 是否可疑
	Risk               RiskLevel // 风险等级
	Reason             string   // 不通过或可疑的原因
	Device             *domain.Device
	FingerprintChanged bool     // 指纹是否变化
}

// RiskLevel 风险等级
type RiskLevel string

const (
	RiskLevelNone   RiskLevel = "none"   // 无风险
	RiskLevelLow    RiskLevel = "low"    // 低风险
	RiskLevelMedium RiskLevel = "medium" // 中风险
	RiskLevelHigh   RiskLevel = "high"   // 高风险
)

// deviceService 设备管理服务实现
type deviceService struct {
	deviceRepo repository.DeviceRepository
	userRepo   repository.UserRepository
}

// NewDeviceService 创建设备管理服务
func NewDeviceService(
	deviceRepo repository.DeviceRepository,
	userRepo repository.UserRepository,
) DeviceService {
	return &deviceService{
		deviceRepo: deviceRepo,
		userRepo:   userRepo,
	}
}

// RegisterDevice 注册或更新设备
func (s *deviceService) RegisterDevice(ctx context.Context, req *RegisterDeviceRequest) (*domain.Device, error) {
	// 1. 验证用户是否存在
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}
	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	// 2. 生成设备指纹
	fingerprint := domain.GenerateFingerprint(req.DeviceName, req.Platform, req.DeviceID, req.OSVersion)

	// 3. 检查设备是否已存在
	existingDevice, err := s.deviceRepo.GetByFingerprint(ctx, req.UserID, fingerprint)
	if err != nil && err != domain.ErrDeviceNotFound {
		return nil, fmt.Errorf("failed to get device by fingerprint: %w", err)
	}

	// 4. 如果设备已存在，更新登录信息
	if existingDevice != nil {
		existingDevice.UpdateLoginInfo(req.ClientIP)
		existingDevice.AppVersion = req.AppVersion
		err = s.deviceRepo.UpdateLoginInfo(ctx, existingDevice.ID, req.ClientIP, existingDevice.LastLoginAt)
		if err != nil {
			return nil, fmt.Errorf("failed to update device login info: %w", err)
		}
		return existingDevice, nil
	}

	// 5. 新设备注册：检查设备数量限制
	deviceCount, err := s.deviceRepo.CountByUserID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to count user devices: %w", err)
	}

	// 6. 如果达到设备上限，尝试清理不活跃设备
	if deviceCount >= domain.MaxDevicesPerUser {
		cleaned, err := s.cleanInactiveDevices(ctx, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to clean inactive devices: %w", err)
		}

		// 如果清理后仍然达到上限，返回错误
		if !cleaned {
			return nil, domain.ErrMaxDevicesExceeded
		}
	}

	// 7. 创建新设备
	newDevice := domain.NewDevice(
		req.UserID,
		req.DeviceName,
		req.Platform,
		req.AppVersion,
		req.ClientIP,
		fingerprint,
	)

	err = s.deviceRepo.Create(ctx, newDevice)
	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	return newDevice, nil
}

// cleanInactiveDevices 清理用户的不活跃设备
// 返回是否成功清理出空间
func (s *deviceService) cleanInactiveDevices(ctx context.Context, userID string) (bool, error) {
	devices, err := s.deviceRepo.ListByUserID(ctx, userID)
	if err != nil {
		return false, err
	}

	// 找到最旧的不活跃设备并删除
	for _, device := range devices {
		if device.IsInactive() {
			err = s.deviceRepo.Delete(ctx, device.ID)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	// 没有不活跃设备可以清理
	return false, nil
}

// VerifyDevice 验证设备（检测异常登录）
func (s *deviceService) VerifyDevice(ctx context.Context, req *VerifyDeviceRequest) (*DeviceVerificationResult, error) {
	// 1. 生成当前设备指纹
	currentFingerprint := domain.GenerateFingerprint(req.DeviceName, req.Platform, req.DeviceID, req.OSVersion)

	// 2. 查找用户已注册的设备
	existingDevice, err := s.deviceRepo.GetByFingerprint(ctx, req.UserID, currentFingerprint)
	if err != nil && err != domain.ErrDeviceNotFound {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// 3. 如果设备不存在，这是一个新设备（需要注册）
	if existingDevice == nil {
		return &DeviceVerificationResult{
			IsValid:      false,
			IsSuspicious: false,
			Risk:         RiskLevelNone,
			Reason:       "device not registered",
		}, nil
	}

	// 4. 检查设备指纹是否发生变化（可疑登录）
	if existingDevice.IsFingerprintChanged(currentFingerprint) {
		return &DeviceVerificationResult{
			IsValid:            false,
			IsSuspicious:       true,
			Risk:               RiskLevelHigh,
			Reason:             "device fingerprint mismatch - possible device spoofing",
			Device:             existingDevice,
			FingerprintChanged: true,
		}, domain.ErrFingerprintMismatch
	}

	// 5. 检查IP地址是否突然变化（可选的启发式检查）
	riskLevel := s.evaluateLoginRisk(existingDevice, req.ClientIP)

	// 6. 验证通过
	return &DeviceVerificationResult{
		IsValid:            true,
		IsSuspicious:       riskLevel == RiskLevelMedium || riskLevel == RiskLevelHigh,
		Risk:               riskLevel,
		Reason:             getRiskReason(riskLevel),
		Device:             existingDevice,
		FingerprintChanged: false,
	}, nil
}

// evaluateLoginRisk 评估登录风险
func (s *deviceService) evaluateLoginRisk(device *domain.Device, currentIP string) RiskLevel {
	// 检查IP地址是否变化
	if device.LastIP != currentIP {
		// IP变化是正常的（移动网络、WiFi切换等）
		// 这里可以添加更复杂的逻辑，如：
		// - 检查IP地理位置是否跨国
		// - 检查短时间内是否多次IP变化
		// - 检查IP是否在黑名单中
		return RiskLevelLow
	}

	return RiskLevelNone
}

// getRiskReason 获取风险原因
func getRiskReason(level RiskLevel) string {
	switch level {
	case RiskLevelNone:
		return "no risk detected"
	case RiskLevelLow:
		return "IP address changed (normal behavior)"
	case RiskLevelMedium:
		return "suspicious activity detected"
	case RiskLevelHigh:
		return "high risk - device spoofing suspected"
	default:
		return "unknown risk"
	}
}

// ListDevices 获取用户的所有设备
func (s *deviceService) ListDevices(ctx context.Context, userID string) ([]*domain.Device, error) {
	// 验证用户是否存在
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, domain.ErrUserNotFound
	}

	// 获取设备列表
	devices, err := s.deviceRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	return devices, nil
}

// RemoveDevice 删除设备
func (s *deviceService) RemoveDevice(ctx context.Context, userID, deviceID string) error {
	// 1. 获取设备
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	// 2. 验证设备是否属于该用户
	if device.UserID != userID {
		return fmt.Errorf("device does not belong to user")
	}

	// 3. 删除设备
	err = s.deviceRepo.Delete(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	return nil
}

// RemoveInactiveDevices 删除不活跃设备（自动清理）
func (s *deviceService) RemoveInactiveDevices(ctx context.Context, days int) (int, error) {
	if days <= 0 {
		days = 90 // 默认90天
	}

	before := time.Now().AddDate(0, 0, -days)
	err := s.deviceRepo.DeleteInactive(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete inactive devices: %w", err)
	}

	// TODO: 可以添加返回删除数量的逻辑
	return 0, nil
}
