package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// MaxDevicesPerUser 每个用户最多允许的设备数量
	MaxDevicesPerUser = 5
)

// Device 设备实体
type Device struct {
	ID          string    // UUID
	UserID      string    // 用户ID
	DeviceName  string    // 设备名称，如 "iPhone 13 Pro"
	Fingerprint string    // 设备指纹，用于检测异常登录
	Platform    string    // 平台：iOS/Android/Web
	AppVersion  string    // 应用版本号
	LastIP      string    // 最后登录IP
	LastLoginAt time.Time // 最后登录时间
	CreatedAt   time.Time // 创建时间
}

// NewDevice 创建新设备
func NewDevice(userID, deviceName, platform, appVersion, ip string, fingerprint string) *Device {
	now := time.Now()
	return &Device{
		ID:          uuid.New().String(),
		UserID:      userID,
		DeviceName:  deviceName,
		Fingerprint: fingerprint,
		Platform:    platform,
		AppVersion:  appVersion,
		LastIP:      ip,
		LastLoginAt: now,
		CreatedAt:   now,
	}
}

// GenerateFingerprint 生成设备指纹
// 基于设备名称、平台、设备ID等信息生成唯一指纹
func GenerateFingerprint(deviceName, platform, deviceID, osVersion string) string {
	data := fmt.Sprintf("%s:%s:%s:%s", deviceName, platform, deviceID, osVersion)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// UpdateLoginInfo 更新登录信息
func (d *Device) UpdateLoginInfo(ip string) {
	d.LastIP = ip
	d.LastLoginAt = time.Now()
}

// IsFingerprintChanged 检查设备指纹是否发生变化（可能是设备伪造）
func (d *Device) IsFingerprintChanged(newFingerprint string) bool {
	return d.Fingerprint != newFingerprint
}

// IsSuspiciousLogin 检查是否为可疑登录
// 判断条件：
// 1. 设备指纹发生变化
// 2. IP地址突然变化（可选，需要结合地理位置判断）
func (d *Device) IsSuspiciousLogin(newFingerprint, newIP string) bool {
	// 指纹变化是强信号
	if d.IsFingerprintChanged(newFingerprint) {
		return true
	}

	// TODO: 可以添加更多启发式规则，如：
	// - IP地理位置突然跨国
	// - 短时间内多次登录
	return false
}

// Validate 验证设备数据
func (d *Device) Validate() error {
	if d.ID == "" {
		return ErrInvalidDeviceID
	}
	if d.UserID == "" {
		return ErrInvalidUserID
	}
	if d.DeviceName == "" {
		return ErrInvalidDeviceName
	}
	if d.Fingerprint == "" {
		return ErrInvalidFingerprint
	}
	if d.Platform == "" {
		return ErrInvalidPlatform
	}
	if !isValidPlatform(d.Platform) {
		return ErrInvalidPlatform
	}
	return nil
}

// isValidPlatform 检查平台是否有效
func isValidPlatform(platform string) bool {
	validPlatforms := map[string]bool{
		"iOS":     true,
		"Android": true,
		"Web":     true,
		"Desktop": true,
	}
	return validPlatforms[platform]
}

// GetDisplayName 获取设备显示名称
func (d *Device) GetDisplayName() string {
	return fmt.Sprintf("%s (%s)", d.DeviceName, d.Platform)
}

// IsInactive 检查设备是否不活跃（超过90天未登录）
func (d *Device) IsInactive() bool {
	return time.Since(d.LastLoginAt) > 90*24*time.Hour
}
