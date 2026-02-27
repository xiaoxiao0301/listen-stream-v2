package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	// SMS提供商常量
	ProviderAliyun  = "aliyun"
	ProviderTencent = "tencent"
	ProviderTwilio  = "twilio"
)

// SMSRecord 短信发送记录实体（用于统计与审计）
type SMSRecord struct {
	ID        string    // UUID
	Phone     string    // 手机号
	Provider  string    // 短信提供商：aliyun/tencent/twilio
	Success   bool      // 发送是否成功
	ErrorMsg  string    // 错误信息（如果失败）
	CreatedAt time.Time // 创建时间
}

// NewSMSRecord 创建新的短信发送记录
func NewSMSRecord(phone, provider string, success bool, errorMsg string) *SMSRecord {
	return &SMSRecord{
		ID:        uuid.New().String(),
		Phone:     phone,
		Provider:  provider,
		Success:   success,
		ErrorMsg:  errorMsg,
		CreatedAt: time.Now(),
	}
}

// NewSuccessSMSRecord 创建成功发送记录
func NewSuccessSMSRecord(phone, provider string) *SMSRecord {
	return NewSMSRecord(phone, provider, true, "")
}

// NewFailedSMSRecord 创建失败发送记录
func NewFailedSMSRecord(phone, provider string, errorMsg string) *SMSRecord {
	return NewSMSRecord(phone, provider, false, errorMsg)
}

// Validate 验证短信记录数据
func (s *SMSRecord) Validate() error {
	if s.ID == "" {
		return ErrInvalidSMSRecordID
	}
	if s.Phone == "" {
		return ErrInvalidPhone
	}
	if len(s.Phone) != 11 {
		return ErrInvalidPhoneFormat
	}
	if s.Provider == "" {
		return ErrInvalidSMSProvider
	}
	if !isValidProvider(s.Provider) {
		return ErrInvalidSMSProvider
	}
	return nil
}

// isValidProvider 检查提供商是否有效
func isValidProvider(provider string) bool {
	validProviders := map[string]bool{
		ProviderAliyun:  true,
		ProviderTencent: true,
		ProviderTwilio:  true,
	}
	return validProviders[provider]
}

// IsSuccess 检查发送是否成功
func (s *SMSRecord) IsSuccess() bool {
	return s.Success
}

// GetProviderName 获取提供商名称
func (s *SMSRecord) GetProviderName() string {
	switch s.Provider {
	case ProviderAliyun:
		return "阿里云"
	case ProviderTencent:
		return "腾讯云"
	case ProviderTwilio:
		return "Twilio"
	default:
		return s.Provider
	}
}
