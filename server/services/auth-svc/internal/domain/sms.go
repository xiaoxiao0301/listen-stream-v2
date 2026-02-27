package domain

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// SMSCodeLength 短信验证码长度
	SMSCodeLength = 6
	// SMSCodeExpiration 短信验证码有效期（5分钟）
	SMSCodeExpiration = 5 * time.Minute
	// SMSCodeRateLimit 同一手机号发送间隔（60秒）
	SMSCodeRateLimit = 60 * time.Second
)

// SMSVerification 短信验证实体
type SMSVerification struct {
	ID        string     // UUID
	Phone     string     // 手机号
	Code      string     // 6位数字验证码
	ExpiresAt time.Time  // 过期时间（5分钟有效）
	UsedAt    *time.Time // 使用时间（验证成功后设置）
	CreatedAt time.Time  // 创建时间
}

// NewSMSVerification 创建新的短信验证
func NewSMSVerification(phone string) (*SMSVerification, error) {
	code, err := generateSMSCode()
	if err != nil {
		return nil, fmt.Errorf("generate SMS code: %w", err)
	}

	now := time.Now()
	return &SMSVerification{
		ID:        uuid.New().String(),
		Phone:     phone,
		Code:      code,
		ExpiresAt: now.Add(SMSCodeExpiration),
		UsedAt:    nil,
		CreatedAt: now,
	}, nil
}

// generateSMSCode 生成6位随机数字验证码
func generateSMSCode() (string, error) {
	// 生成6位随机数字
	bytes := make([]byte, 3) // 3字节可以生成6位数字
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// 转换为6位数字字符串
	code := fmt.Sprintf("%06d", int(bytes[0])<<16|int(bytes[1])<<8|int(bytes[2])%1000000)
	return code, nil
}

// IsExpired 检查验证码是否已过期
func (s *SMSVerification) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsUsed 检查验证码是否已使用
func (s *SMSVerification) IsUsed() bool {
	return s.UsedAt != nil
}

// MarkAsUsed 标记验证码为已使用
func (s *SMSVerification) MarkAsUsed() error {
	if s.IsUsed() {
		return ErrSMSCodeAlreadyUsed
	}
	if s.IsExpired() {
		return ErrSMSCodeExpired
	}

	now := time.Now()
	s.UsedAt = &now
	return nil
}

// Verify 验证验证码
func (s *SMSVerification) Verify(code string) error {
	if s.IsUsed() {
		return ErrSMSCodeAlreadyUsed
	}
	if s.IsExpired() {
		return ErrSMSCodeExpired
	}
	if s.Code != code {
		return ErrSMSCodeInvalid
	}

	return nil
}

// Validate 验证短信验证数据
func (s *SMSVerification) Validate() error {
	if s.ID == "" {
		return ErrInvalidSMSID
	}
	if s.Phone == "" {
		return ErrInvalidPhone
	}
	if len(s.Phone) != 11 {
		return ErrInvalidPhoneFormat
	}
	if s.Code == "" {
		return ErrInvalidSMSCode
	}
	if len(s.Code) != SMSCodeLength {
		return ErrInvalidSMSCodeLength
	}
	return nil
}

// GetRemainingTime 获取剩余有效时间
func (s *SMSVerification) GetRemainingTime() time.Duration {
	if s.IsExpired() {
		return 0
	}
	return time.Until(s.ExpiresAt)
}

// CanResend 检查是否可以重新发送（基于创建时间）
func (s *SMSVerification) CanResend() bool {
	return time.Since(s.CreatedAt) >= SMSCodeRateLimit
}
